package internal

import (
	"time"

	"go.bryk.io/pkg/cli"
	drpcWS "go.bryk.io/pkg/net/drpc/ws"
	rpcWS "go.bryk.io/pkg/net/rpc/ws"
	"go.bryk.io/x/errors"
)

// WSProxy allows to configure a WebSocket proxy for RPC and DRPC servers.
type WSProxy struct {
	// Enable/Disable the use of the WebSocket proxy.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Specify if the server should attempt to negotiate per message
	// compression (RFC 7692).
	Compression bool `json:"compression" yaml:"compression" mapstructure:"compression"`

	// Specifies the duration for the handshake to complete.
	HandshakeTimeout uint `json:"handshake_timeout" yaml:"handshake_timeout" mapstructure:"handshake_timeout"`

	// Allows to map an incoming URL parameter and use it to adjust the
	// HTTP method for requests behind the proxy.
	MethodOverride string `json:"method_override" yaml:"method_override" mapstructure:"method_override"`

	// allows the proxy to load authorization credentials from a cookie,
	// when present, and forward the token on the "Authorization" header on
	// requests behind the proxy.
	AuthCookie string `json:"auth_cookie" yaml:"auth_cookie" mapstructure:"auth_cookie"`

	// SubProtocols specifies the server's supported protocols in order of preference.
	SubProtocols []string `json:"sub_protocols" yaml:"sub_protocols" mapstructure:"sub_protocols"`

	// Sets which HTTP headers (case-insensitive) should be forward in
	// requests behind the proxy.
	ForwardHeaders []string `json:"forward_headers" yaml:"forward_headers" mapstructure:"forward_headers"`
}

// Validate the provided proxy settings.
func (c *WSProxy) Validate() error {
	if c.HandshakeTimeout > 10 {
		return errors.New("maximum handshake timeout is 10 seconds")
	}
	return nil
}

// Params available when using the loader with a CLI application.
func (c *WSProxy) Params(prefix string) []cli.Param {
	return []cli.Param{
		{
			Name:      withPrefix(prefix, "wsproxy-compression", "-"),
			Usage:     "Enable compression",
			FlagKey:   withPrefix(prefix, "wsproxy.compression", "."),
			ByDefault: false,
		},
		{
			Name:      withPrefix(prefix, "wsproxy-handshake-timeout", "-"),
			Usage:     "Handshake timeout (in seconds)",
			FlagKey:   withPrefix(prefix, "wsproxy.handshake_timeout", "."),
			ByDefault: 5,
		},
		{
			Name:      withPrefix(prefix, "wsproxy-method-override", "-"),
			Usage:     "Map an URL parameter and use it to adjust the HTTP method for requests",
			FlagKey:   withPrefix(prefix, "wsproxy.method_override", "."),
			ByDefault: "",
		},
		{
			Name:      withPrefix(prefix, "wsproxy-auth-cookie", "-"),
			Usage:     "Load credentials from a cookie and forward them on the 'Authorization' header",
			FlagKey:   withPrefix(prefix, "wsproxy.auth_cookie", "."),
			ByDefault: "",
		},
		{
			Name:      withPrefix(prefix, "wsproxy-sub-protocols", "-"),
			Usage:     "Specifies the server's supported protocols in order of preference",
			FlagKey:   withPrefix(prefix, "wsproxy.sub_protocols", "."),
			ByDefault: []string{},
		},
		{
			Name:      withPrefix(prefix, "wsproxy-forward-headers", "-"),
			Usage:     "Sets which HTTP headers (case insensitive) should be forwarded (default to all headers)",
			FlagKey:   withPrefix(prefix, "wsproxy.forward_headers", "."),
			ByDefault: []string{},
		},
	}
}

// Expand the proxy settings and return them on the proper type as specified
// by `ti`.
func (c *WSProxy) Expand(ti string) interface{} {
	switch ti {
	case "rpc":
		return c.forRPC()
	case "drpc":
		return c.forDRPC()
	default:
		return nil
	}
}

func (c *WSProxy) forRPC() []rpcWS.ProxyOption {
	var list []rpcWS.ProxyOption
	if c.Compression {
		list = append(list, rpcWS.EnableCompression())
	}
	if len(c.SubProtocols) > 0 {
		list = append(list, rpcWS.SubProtocols(c.SubProtocols))
	}
	if c.HandshakeTimeout > 0 {
		list = append(list, rpcWS.HandshakeTimeout(time.Duration(c.HandshakeTimeout)*time.Second))
	}
	if c.MethodOverride != "" {
		list = append(list, rpcWS.MethodOverride(c.MethodOverride))
	}
	if c.AuthCookie != "" {
		list = append(list, rpcWS.AuthorizationCookie(c.AuthCookie))
	}
	if len(c.ForwardHeaders) > 0 {
		list = append(list, rpcWS.ForwardHeaders(c.ForwardHeaders))
	}
	return list
}

func (c *WSProxy) forDRPC() []drpcWS.ProxyOption {
	var list []drpcWS.ProxyOption
	if c.Compression {
		list = append(list, drpcWS.EnableCompression())
	}
	if len(c.SubProtocols) > 0 {
		list = append(list, drpcWS.SubProtocols(c.SubProtocols))
	}
	if c.HandshakeTimeout > 0 {
		list = append(list, drpcWS.HandshakeTimeout(time.Duration(c.HandshakeTimeout)*time.Second))
	}
	return list
}
