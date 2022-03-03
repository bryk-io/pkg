package loader

import (
	"time"

	"github.com/pkg/errors"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/rpc/ws"
)

// nolint: lll
type confWS struct {
	Compression         bool     `json:"compression" yaml:"compression" mapstructure:"compression"`
	HandshakeTimeout    uint     `json:"handshake_timeout" yaml:"handshake_timeout" mapstructure:"handshake_timeout"`
	MethodOverride      string   `json:"method_override" yaml:"method_override" mapstructure:"method_override"`
	AuthorizationCookie string   `json:"authorization_cookie" yaml:"authorization_cookie" mapstructure:"authorization_cookie"`
	SubProtocols        []string `json:"sub_protocols" yaml:"sub_protocols" mapstructure:"sub_protocols"`
	ForwardHeaders      []string `json:"forward_headers" yaml:"forward_headers" mapstructure:"forward_headers"`
}

func (c *confWS) setDefaults() {
	c.Compression = true
	c.HandshakeTimeout = 5
	c.SubProtocols = []string{}
	c.ForwardHeaders = []string{}
}

func (c *confWS) validate() error {
	if c.HandshakeTimeout > 10 {
		return errors.New("maximum handshake timeout is 10 seconds")
	}
	return nil
}

func (c *confWS) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "websocket-compression",
			Usage:     "Enable compression",
			FlagKey:   "websocket.compression",
			ByDefault: false,
		},
		{
			Name:      "websocket-handshake-timeout",
			Usage:     "Handshake timeout (in seconds)",
			FlagKey:   "websocket.handshake_timeout",
			ByDefault: 5,
		},
		{
			Name:      "websocket-method-override",
			Usage:     "Map an URL parameter and use it to adjust the HTTP method for requests",
			FlagKey:   "websocket.method_override",
			ByDefault: "",
		},
		{
			Name:      "websocket-authorization-cookie",
			Usage:     "Load credentials from a cookie and forward them on the 'Authorization' header",
			FlagKey:   "websocket.authorization_cookie",
			ByDefault: "",
		},
		{
			Name:      "websocket-sub-protocols",
			Usage:     "Specifies the server's supported protocols in order of preference",
			FlagKey:   "websocket.sub_protocols",
			ByDefault: []string{},
		},
		{
			Name:      "websocket-forward-headers",
			Usage:     "Sets which HTTP headers (case insensitive) should be forwarded (default to all headers)",
			FlagKey:   "websocket.forward_headers",
			ByDefault: []string{},
		},
	}
}

func (c *confWS) expand() []ws.ProxyOption {
	var list []ws.ProxyOption
	if c.Compression {
		list = append(list, ws.EnableCompression())
	}
	if len(c.SubProtocols) > 0 {
		list = append(list, ws.SubProtocols(c.SubProtocols))
	}
	if c.HandshakeTimeout > 0 {
		list = append(list, ws.HandshakeTimeout(time.Duration(c.HandshakeTimeout)*time.Second))
	}
	if c.MethodOverride != "" {
		list = append(list, ws.MethodOverride(c.MethodOverride))
	}
	if c.AuthorizationCookie != "" {
		list = append(list, ws.AuthorizationCookie(c.AuthorizationCookie))
	}
	if len(c.ForwardHeaders) > 0 {
		list = append(list, ws.ForwardHeaders(c.ForwardHeaders))
	}
	return list
}
