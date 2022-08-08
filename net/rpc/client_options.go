package rpc

import (
	"time"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

// ClientOption allows adjusting client settings following a functional
// pattern.
type ClientOption func(*Client) error

// WithInsecureSkipVerify controls whether a client verifies the server's
// certificate chain and host name. If InsecureSkipVerify is true, TLS accepts
// any certificate presented by the server and any host name in that certificate.
// In this mode, TLS is susceptible to MITM attacks. This should be used only
// for testing.
func WithInsecureSkipVerify() ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.skipVerify = true
		return nil
	}
}

// WithServerNameOverride adjust the identifier expected to be present on the
// upstream RPC server's certificate, when using TLS. This option is meant for
// testing only. If set to a non-empty string, it will override the virtual host
// name of authority (e.g. :authority header field) in requests.
func WithServerNameOverride(name string) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.nameOverride = name
		return nil
	}
}

// WithAuthCertificate enabled certificate-based client authentication with the
// provided credentials.
func WithAuthCertificate(cert, key []byte) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		cert, err := LoadCertificate(cert, key)
		if err != nil {
			return errors.WithStack(err)
		}
		c.cert = &cert
		return nil
	}
}

// WithAuthToken use the provided token string as bearer authentication credential.
func WithAuthToken(token string) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		ct := authToken{
			kind:  "Bearer",
			value: token,
		}
		c.dialOpts = append(c.dialOpts, grpc.WithPerRPCCredentials(ct))
		return nil
	}
}

// WithUserAgent sets the "user-agent" value use by the client instance.
func WithUserAgent(val string) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.dialOpts = append(c.dialOpts, grpc.WithUserAgent(val))
		return nil
	}
}

// WithTimeout establish a time limit when dialing a connection with the server.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.timeout = timeout
		return nil
	}
}

// WaitForReady makes the connection to block until it becomes ready.
func WaitForReady() ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.dialOpts = append(c.dialOpts, grpc.WithBlock())
		return nil
	}
}

// WithClientTLS set parameters to establish a secure connection channel with the
// server.
func WithClientTLS(opts ClientTLSConfig) ClientOption {
	return func(c *Client) (err error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.tlsOpts = opts
		c.tlsConf, err = clientTLSConf(opts)
		return errors.WithStack(err)
	}
}

// WithCompression will enable standard GZIP compression on all client requests.
func WithCompression() ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.callOpts = append(c.callOpts, grpc.UseCompressor(gzip.Name))
		return nil
	}
}

// WithRetry will enable automatic error retries on all client requests.
func WithRetry(config *RetryOptions) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.callOpts = append(c.callOpts, Retry(config)...)
		return nil
	}
}

// WithKeepalive will configure the client to send a ping message when a certain
// time (in seconds) has passed without activity in the connection. The minimum valid
// interval is 10 seconds.
func WithKeepalive(t int) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		if t < 10 {
			t = 10
		}
		c.dialOpts = append(c.dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(t) * time.Second,
			Timeout:             time.Duration(t) * time.Second,
			PermitWithoutStream: true,
		}))
		return nil
	}
}

// WithLoadBalancer configures the client connection to enable load balancing, by
// default the "round-robin" strategy is used to choose a backend for RPC requests.
// When enabling this option the provided endpoint is expected to be a DNS record that
// returns a set of reachable IP addresses.
// When deploying with Kubernetes this is done by using a "headless" service.
//
// More information:
// 	https://kubernetes.io/docs/concepts/services-networking/service/#headless-services/,/
func WithLoadBalancer() ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.useBalancer = true
		return nil
	}
}

// WithClientObservability instrument the client instance using observability
// operator provided. If no operator is set, the default global instance is used.
func WithClientObservability(oop *otel.Operator) ClientOption {
	return func(c *Client) error {
		c.mu.Lock()
		c.oop = oop
		c.mu.Unlock()
		return nil
	}
}
