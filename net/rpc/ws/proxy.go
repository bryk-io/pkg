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

// New returns a WebsocketProxy instance that expose the underlying handler as a
// bidirectional websocket stream with newline-delimited JSON as the content encoding.
// The HTTP `Authorization` header is either populated from the `Sec-Websocket-Protocol`
// field or by a cookie.
func New(opts ...ProxyOption) (*Proxy, error) {
	p := &Proxy{
		tokenCookieName:      "",
		methodOverrideParam:  "m",
		closeConnectionParam: "cc",
		forwardHeaders: []string{
			"origin",
			"referer",
		},
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

// Proxy - gRPC Gateway WebSocket proxy.
// Based on the original implementation: https://github.com/tmc/grpc-websocket-proxy
type Proxy struct {
	handler                http.Handler
	wsConf                 websocket.Upgrader
	maxRespBodyBufferBytes int
	forwardHeaders         []string
	methodOverrideParam    string
	closeConnectionParam   string
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
	for _, h := range p.forwardHeaders {
		if strings.ToLower(header) == h {
			return true
		}
	}
	return false
}

// Prepare outgoing request.
func (p *Proxy) prepareRequest(incoming http.Request, outgoing *http.Request) {
	// forward additional headers
	for header := range incoming.Header {
		if p.forwardHeader(header) {
			outgoing.Header.Set(header, incoming.Header.Get(header))
		}
	}

	// ensure authorization header is properly set
	if header := incoming.Header.Get(headerFragment); header != "" {
		outgoing.Header.Set("Authorization", fixProtocolHeader(header))
	}

	// if token cookie is present, populate Authorization header using it
	if p.tokenCookieName != "" {
		if cookie, err := incoming.Cookie(p.tokenCookieName); err == nil {
			outgoing.Header.Set("Authorization", "Bearer "+cookie.Value)
		}
	}

	// method override
	if p.methodOverrideParam != "" {
		if m := incoming.URL.Query().Get(p.methodOverrideParam); m != "" {
			outgoing.Method = strings.ToUpper(m)
		}
	}

	// final request adjustments
	if p.requestMutator != nil {
		p.requestMutator(incoming, outgoing)
	}
}

// nolint: funlen
func (p *Proxy) proxy(w http.ResponseWriter, r *http.Request) {
	var responseHeader http.Header
	var newLine = []byte("\n")

	// if `Sec-WebSocket-Protocol` header starts with "Bearer", respond in kind
	if strings.HasPrefix(r.Header.Get(headerFragment), "Bearer") {
		responseHeader = http.Header{
			headerFragment: []string{"Bearer"},
		}
	}

	// upgrade request and establish WebSocket connection
	conn, err := p.wsConf.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	// pipe request
	reqBodyR, reqBodyW := io.Pipe()
	request, err := http.NewRequest(r.Method, r.URL.String(), reqBodyR)
	if err != nil {
		return
	}

	// prepare outgoing request; last chance to adjust it in any way
	p.prepareRequest(*r, request)

	closeEarly := closeConnectionEarly(r, p.closeConnectionParam)

	// pipe response
	resBodyR, resBodyW := io.Pipe()
	response := newResponseWriter(resBodyW)

	// close request and response writers when context is done
	go func() {
		<-ctx.Done()
		_ = reqBodyW.CloseWithError(io.EOF)
		_ = resBodyW.CloseWithError(io.EOF)
		response.closed <- true
	}()

	// process request in the wrapped HTTP handler
	go func() {
		defer cancelFn()
		p.handler.ServeHTTP(response, request)
	}()

	// read loop -> take messages from websocket and write to http request
	go func() {
		defer cancelFn()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if closeEarly {
				// close request body since server doesn't expect any more data
				_ = reqBodyW.Close()
			}
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// send payload and append new line delimiter
			if _, err := reqBodyW.Write(append(payload, newLine...)); err != nil {
				return
			}
		}
	}()

	// write loop -> take messages from response and write to websocket
	scanner := bufio.NewScanner(resBodyR)

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
