package ws

import (
	"net/http"
	"strings"
	"time"
)

// ProxyOption provides functional-style configuration settings for a proxy instance.
type ProxyOption func(p *Proxy) error

// EnableCompression specify if the server should attempt to negotiate per
// message compression (RFC 7692). Setting this value to true does not guarantee
// that compression will be supported. Currently, only "no context takeover"
// modes are supported.
func EnableCompression() ProxyOption {
	return func(p *Proxy) error {
		p.wsConf.EnableCompression = true
		return nil
	}
}

// CheckOrigin should return true if the request Origin header is acceptable.
// If no setting is provided a safe default is used: return false if the Origin
// request header is present and the origin host is not equal to request Host
// header. A CheckOrigin function should carefully validate the request origin
// to prevent cross-site request forgery.
func CheckOrigin(f func(*http.Request) bool) ProxyOption {
	return func(p *Proxy) error {
		p.wsConf.CheckOrigin = f
		return nil
	}
}

// SubProtocols specifies the server's supported protocols in order of preference.
// If no value is provided, the server negotiates a sub-protocol by selecting
// the first match in this list with a protocol requested by the client. If there's
// no match, then no protocol is negotiated (the Sec-Websocket-Protocol header
// is not included in the handshake response).
func SubProtocols(list []string) ProxyOption {
	return func(p *Proxy) error {
		p.wsConf.Subprotocols = list
		return nil
	}
}

// HandshakeTimeout specifies the duration for the handshake to complete.
func HandshakeTimeout(timeout time.Duration) ProxyOption {
	return func(p *Proxy) error {
		p.wsConf.HandshakeTimeout = timeout
		return nil
	}
}

// MethodOverride allows to map an incoming URL parameter and use it to adjust the
// HTTP method for requests behind the proxy.
func MethodOverride(param string) ProxyOption {
	return func(p *Proxy) error {
		p.methodOverrideParam = param
		return nil
	}
}

// AuthorizationCookie allows the proxy to load authorization credentials from a
// cookie, when present, and forward the token on the "Authorization" header on
// requests behind the proxy.
func AuthorizationCookie(name string) ProxyOption {
	return func(p *Proxy) error {
		p.tokenCookieName = name
		return nil
	}
}

// ForwardHeaders sets which HTTP headers (case-insensitive) should be forward in
// requests behind the proxy. If no list is specified, ALL headers are forwarded
// by default.
func ForwardHeaders(list []string) ProxyOption {
	return func(p *Proxy) error {
		headers := make([]string, len(list))
		for i, h := range list {
			headers[i] = strings.ToLower(h)
		}
		p.forwardHeaders = headers
		return nil
	}
}

// RequestMutator provides a final configuration point to customize or adjust the
// outgoing HTTP request before is forwarded behind the proxy.
func RequestMutator(f func(incoming http.Request, outgoing *http.Request)) ProxyOption {
	return func(p *Proxy) error {
		p.requestMutator = f
		return nil
	}
}

// RemoveResultWrapper fixes a non-standard behavior when the gateway adds a wrapper
// to all messages received from a streaming operation. The wrapper contains 2
// fields: "result" containing the actual structure expected, and "error" containing
// any generated processing error. With this option enabled, messages will be send as
// expected and the error behavior will continue to work as described earlier. A slight
// performance penalty is introduced by having to decode/re-encode each chunk in the
// stream.
//
// For more information about the issue:
// https://github.com/grpc-ecosystem/grpc-gateway/issues/579
func RemoveResultWrapper() ProxyOption {
	return func(p *Proxy) error {
		p.removeResultWrapper = true
		return nil
	}
}
