package http

import (
	"context"
	"io"
	lib "net/http"
	"net/url"
	"strings"
)

// Client provides an HTTP client instance that's interface-compatible
// with the standard library.
type Client struct {
	mw []func(req *lib.Request)
	hc *lib.Client
}

// NewClient returns an HTTP client with the provided configuration options.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		hc: &lib.Client{
			Transport: lib.DefaultTransport,
		},
	}
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Head issues a HEAD to the specified URL.
func (c *Client) Head(url string) (resp *lib.Response, err error) {
	req, err := lib.NewRequestWithContext(context.TODO(), lib.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Get issues a GET to the specified URL.
func (c *Client) Get(url string) (*lib.Response, error) {
	req, err := lib.NewRequestWithContext(context.TODO(), lib.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Post issues a POST to the specified URL.
func (c *Client) Post(url, contentType string, body io.Reader) (resp *lib.Response, err error) {
	req, err := lib.NewRequestWithContext(context.TODO(), lib.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.do(req)
}

// PostForm issues a POST to the specified URL, with data's keys and values
// URL-encoded as the request body.
func (c *Client) PostForm(url string, data url.Values) (resp *lib.Response, err error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Do sends an HTTP request and returns an HTTP response, following
// policy (such as redirects, cookies, auth) as configured on the
// client.
func (c *Client) Do(req *lib.Request) (*lib.Response, error) {
	return c.do(req)
}

// CloseIdleConnections closes any connections on its [Transport] which
// were previously connected from previous requests but are now
// sitting idle in a "keep-alive" state. It does not interrupt any
// connections currently in use.
func (c *Client) CloseIdleConnections() {
	c.hc.CloseIdleConnections()
}

// apply interceptor(s) and execute request.
func (c *Client) do(req *lib.Request) (*lib.Response, error) {
	for _, ci := range c.mw {
		ci(req)
	}
	return c.hc.Do(req)
}
