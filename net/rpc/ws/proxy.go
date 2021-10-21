package ws

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var headerFragment = "Sec-WebSocket-Protocol"

// New returns a new WebSocket proxy instance.
func New(opts ...ProxyOption) (*Proxy, error) {
	p := &Proxy{
		forwardHeaders:      []string{},
		tokenCookieName:     "",
		methodOverrideParam: "",
		wsConf: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Proxy - WebSocket proxy based on the original implementation by @tmc.
// https://github.com/tmc/grpc-websocket-proxy
type Proxy struct {
	handler                http.Handler
	wsConf                 websocket.Upgrader
	maxRespBodyBufferBytes int
	forwardHeaders         []string
	methodOverrideParam    string
	tokenCookieName        string
	requestMutator         requestMutatorFunc
	removeResultWrapper    bool
}

// Wrap the provided HTTP handler.
func (p *Proxy) Wrap(h http.Handler) http.Handler {
	p.handler = h
	return p
}

// Allows to inspect the incoming HTTP request and adjust the outgoing instance.
type requestMutatorFunc func(incoming http.Request, outgoing *http.Request)

// ServeHTTP handles incoming requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		p.handler.ServeHTTP(w, r)
		return
	}
	p.proxy(w, r)
}

// Decide if a received header should be forwarded behind the proxy.
func (p *Proxy) forwardHeader(header string) bool {
	if len(p.forwardHeaders) == 0 {
		return true
	}
	for _, h := range p.forwardHeaders {
		if strings.ToLower(header) == h {
			return true
		}
	}
	return false
}

// Prepare outgoing request.
func (p *Proxy) prepareRequest(incoming http.Request, outgoing *http.Request) {
	// Forward headers
	for header := range incoming.Header {
		if p.forwardHeader(header) {
			outgoing.Header.Set(header, incoming.Header.Get(header))
		}
	}

	// If token cookie is present, populate Authorization header from the cookie instead.
	if p.tokenCookieName != "" {
		if cookie, err := incoming.Cookie(p.tokenCookieName); err == nil {
			outgoing.Header.Set("Authorization", "Bearer "+cookie.Value)
		}
	}

	// Method override
	if p.methodOverrideParam != "" {
		if m := incoming.URL.Query().Get(p.methodOverrideParam); m != "" {
			outgoing.Method = m
		}
	}

	// Final request adjustments
	if p.requestMutator != nil {
		p.requestMutator(incoming, outgoing)
	}
}

func (p *Proxy) proxy(w http.ResponseWriter, r *http.Request) {
	var responseHeader http.Header

	// If Sec-WebSocket-Protocol starts with "Bearer", respond in kind.
	if strings.HasPrefix(r.Header.Get(headerFragment), "Bearer") {
		responseHeader = http.Header{
			headerFragment: []string{"Bearer"},
		}
	}

	// Upgrade request
	conn, err := p.wsConf.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// Pipe request
	requestBodyR, requestBodyW := io.Pipe()
	request, err := http.NewRequest(r.Method, r.URL.String(), requestBodyR)
	if err != nil {
		return
	}
	if header := r.Header.Get(headerFragment); header != "" {
		request.Header.Set("Authorization", fixProtocolHeader(header))
	}

	// Prepare outgoing request
	p.prepareRequest(*r, request)

	// Pipe response
	responseBodyR, responseBodyW := io.Pipe()
	response := newResponseWriter(responseBodyW)

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	go func() {
		<-ctx.Done()
		_ = requestBodyW.CloseWithError(io.EOF)
		_ = responseBodyW.CloseWithError(io.EOF)
		response.closed <- true
	}()

	go func() {
		defer cancelFn()
		p.handler.ServeHTTP(response, request)
	}()

	go func() {
		// read loop -> Take messages from websocket and write to http request
		defer cancelFn()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, payload, err := conn.ReadMessage()
			if err != nil {
				if isClosedConnError(err) {
					return
				}
				return
			}
			if _, err := requestBodyW.Write(payload); err != nil {
				return
			}
			if _, err := requestBodyW.Write([]byte("\n")); err != nil {
				return
			}
		}
	}()

	// write loop -> Take messages from response and write to websocket
	scanner := bufio.NewScanner(responseBodyR)

	// if maxRespBodyBufferSize has been specified, use custom buffer for scanner
	var scannerBuf []byte
	if p.maxRespBodyBufferBytes > 0 {
		scannerBuf = make([]byte, 0, 64*1024)
		scanner.Buffer(scannerBuf, p.maxRespBodyBufferBytes)
	}
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		chunk := scanner.Bytes()
		if p.removeResultWrapper {
			chunk = removeResultWrapper(chunk)
		}
		if err = conn.WriteMessage(websocket.TextMessage, chunk); err != nil {
			return
		}
	}
	_ = scanner.Err()
}
