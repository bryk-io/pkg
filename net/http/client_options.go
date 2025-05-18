package http

import (
	lib "net/http"
	"time"
)

// ClientOption allows adjusting client settings following a functional pattern.
type ClientOption func(srv *Client) error

// WithRoundTripper adjust the transport used by the client instance.
func WithRoundTripper(rt lib.RoundTripper) ClientOption {
	return func(c *Client) error {
		c.hc.Transport = rt
		return nil
	}
}

// WithTimeout specifies a time limit for requests made by this
// Client. The timeout includes connection time, any redirects,
// and reading the response body. The timer remains running after
// Get, Head, Post, or Do return and will interrupt reading of the
// Response.Body.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.hc.Timeout = timeout
		return nil
	}
}

// WithCookieJar is used to insert relevant cookies into every outbound
// Request and is updated with the cookie values of every inbound Response.
// The Jar is consulted for every redirect that the Client follows.
func WithCookieJar(jar lib.CookieJar) ClientOption {
	return func(c *Client) error {
		c.hc.Jar = jar
		return nil
	}
}

// WithInterceptors allows to transform/adjust every outbound Request
// before being executed by the client.
func WithInterceptors(ci ...func(req *lib.Request)) ClientOption {
	return func(c *Client) error {
		c.mw = append(c.mw, ci...)
		return nil
	}
}
