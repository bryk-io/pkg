package ws

import (
	"context"
	"errors"
	"html"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"storj.io/drpc"
	"storj.io/drpc/drpchttp"
)

var headerFragment = "Sec-WebSocket-Protocol"

var supportContentTypes = []string{
	"application/protobuf",
	"application/json",
}

// Proxy provides support for bidirectional DRPC streaming via websockets.
type Proxy struct {
	handler        drpc.Handler
	fallback       http.Handler
	wsConf         websocket.Upgrader
	forwardHeaders []string
}

// New returns a new WebSocket proxy instance.
func New(opts ...ProxyOption) (*Proxy, error) {
	p := &Proxy{
		forwardHeaders: []string{},
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

// Wrap the provided DRPC handler using `fallback` as handler for HTTP
// clients not requesting a WebSocket protocol connection upgrade.
func (p *Proxy) Wrap(handler drpc.Handler, fallback http.Handler) http.Handler {
	p.handler = handler
	p.fallback = fallback
	return p
}

// ServeHTTP handles incoming HTTP requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// No protocol upgrade? use fallback handler
	if !websocket.IsWebSocketUpgrade(r) {
		p.fallback.ServeHTTP(w, r)
		return
	}

	// Upgrade connection and setup stream handler for it
	if err := p.proxy(w, r); err != nil {
		var ce *websocket.CloseError
		if !errors.As(err, &ce) || ce.Code != websocket.CloseNormalClosure {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, err.JSON(), err.status)
		}
		return
	}
}

// Manage requests using a custom DRPC stream backed by a websocket
// connection.
func (p *Proxy) proxy(w http.ResponseWriter, r *http.Request) *proxyErr {
	var responseHeader http.Header

	// Validate content-type header
	ct := r.Header.Get("Content-Type")
	if !isValidContentType(ct) {
		return newProxyErr(http.StatusUnsupportedMediaType, "invalid content type: %s", ct)
	}

	// Get the http request context while preserving metadata values
	// sent using the X-Drpc-Metadata header
	ctx, err := drpchttp.Context(r)
	if err != nil {
		return wrapErr(http.StatusInternalServerError, err)
	}

	// If Sec-WebSocket-Protocol starts with "Bearer", respond in kind
	if strings.HasPrefix(r.Header.Get(headerFragment), "Bearer") {
		responseHeader = http.Header{
			headerFragment: []string{"Bearer"},
		}
	}

	// Upgrade request
	conn, err := p.wsConf.Upgrade(w, r, responseHeader)
	if err != nil {
		return wrapErr(http.StatusInternalServerError, err)
	}

	// Custom stream handler
	rc, halt := context.WithCancel(ctx)
	defer halt()
	req := &wrappedStream{
		ctx:  rc,
		conn: conn,
		json: ct == "application/json",
	}

	// Handle the incoming RPC request
	if err := p.handler.HandleRPC(req, sanitize(r.URL.Path)); err != nil {
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code != websocket.CloseNormalClosure {
			return wrapErr(http.StatusInternalServerError, err)
		}
	}
	return nil
}

func sanitize(src string) string {
	res := strings.Replace(strings.Replace(src, "\n", "", -1), "\r", "", -1)
	res = html.EscapeString(res)
	return res
}

func isValidContentType(ct string) bool {
	for _, el := range supportContentTypes {
		if ct == el {
			return true
		}
	}
	return false
}
