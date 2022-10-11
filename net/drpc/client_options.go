package drpc

import (
	"go.bryk.io/pkg/errors"
	clmw "go.bryk.io/pkg/net/drpc/middleware/client"
)

// ClientOption allows adjusting client settings following a functional pattern.
type ClientOption func(cl *Client) error

// WithClientTLS adjust the client to establish a secure communication channel
// with the server.
func WithClientTLS(opts ClientTLS) ClientOption {
	return func(cl *Client) error {
		tc, err := opts.conf()
		if err != nil {
			return err
		}
		cl.tls = tc
		return nil
	}
}

// WithAuthCertificate enabled certificate-based client authentication with the
// provided credentials. This requires the client and the server to use a TLS
// communication channel, otherwise this option will be ignored.
func WithAuthCertificate(cert, key []byte) ClientOption {
	return func(c *Client) error {
		ct, err := LoadCertificate(cert, key)
		if err != nil {
			return errors.WithStack(err)
		}
		c.cert = &ct
		return nil
	}
}

// WithProtocolHeader ensure the client connections include the protocol selection
// header. This is required when the server supports both DRPC and HTTP requests.
func WithProtocolHeader() ClientOption {
	return func(cl *Client) error {
		cl.http = true
		return nil
	}
}

// WithPoolCapacity adjust the max limit of concurrent DRPC connections a single
// client instance can support.
func WithPoolCapacity(limit int) ClientOption {
	return func(cl *Client) error {
		cl.capacity = limit
		return nil
	}
}

// WithClientMiddleware register the provided middleware to customize/extend
// the processing of RPC requests. When providing middleware the ordering is very
// important; middleware will be applied in the same order provided.
//
//	For example:
//	  Use(foo bar baz)
//	Will be applied as:
//	  baz( bar( foo(handler) ) )
func WithClientMiddleware(mw ...clmw.Middleware) ClientOption {
	return func(cl *Client) error {
		cl.Use(mw...)
		return nil
	}
}
