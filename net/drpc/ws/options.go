package ws

import (
	"net/http"
	"time"
)

// ProxyOption provides functional-style configuration settings for a proxy instance.
type ProxyOption func(p *Proxy) error

// EnableCompression specify if the server should attempt to negotiate per
// message compression (RFC 7692). Setting this value to true does not guarantee
// that compression will be supported. Currently only "no context takeover"
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
