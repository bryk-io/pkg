package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	mw "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pkg/errors"
	"go.bryk.io/pkg/otel"
	"go.bryk.io/pkg/otel/extras"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client provides an RPC client wrapper with several utilities.
type Client struct {
	tlsOpts          ClientTLSConfig
	callOpts         []grpc.CallOption
	dialOpts         []grpc.DialOption
	middlewareUnary  []grpc.UnaryClientInterceptor
	middlewareStream []grpc.StreamClientInterceptor
	nameOverride     string
	cert             *tls.Certificate
	timeout          time.Duration
	tlsConf          *tls.Config
	oop              *otel.Operator
	mu               sync.Mutex
	useBalancer      bool
	skipVerify       bool
}

// NewClient set up a new client instance.
func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{}
	if err := c.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}

	// TLS configuration
	if c.tlsConf == nil {
		c.dialOpts = append(c.dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		c.dialOpts = append(c.dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(c.tlsConf)))
	}

	// Add the default call options set
	if len(c.callOpts) > 0 {
		c.dialOpts = append(c.dialOpts, grpc.WithDefaultCallOptions(c.callOpts...))
	}

	// Add middleware
	unary, stream := c.getMiddleware()
	c.dialOpts = append(c.dialOpts, grpc.WithUnaryInterceptor(mw.ChainUnaryClient(unary...)))
	c.dialOpts = append(c.dialOpts, grpc.WithStreamInterceptor(mw.ChainStreamClient(stream...)))
	return c, nil
}

// GetConnection returns a RPC client connection for the client instance.
func (c *Client) GetConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	// Validate endpoint
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("endpoint required")
	}

	// This is the official name resolution syntax to use for DNS targets.
	// The "authority" value is left blank since is not widely supported.
	// Another option is to leave the endpoint as-is a set a default name
	// resolution schema:
	//
	// resolver.SetDefaultScheme("dns")
	//
	// For more information:
	// https://github.com/grpc/grpc/blob/master/doc/naming.md#name-syntax
	if c.useBalancer && !strings.HasPrefix(endpoint, "dns:///") {
		endpoint = fmt.Sprintf("dns:///%s", endpoint)
	}

	// Dial and return connection
	if c.timeout == 0 {
		conn, err = grpc.Dial(endpoint, c.dialOpts...)
		return conn, errors.Wrap(err, "failed to dial")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), c.timeout)
	defer cancel()
	conn, err = grpc.DialContext(ctx, endpoint, c.dialOpts...)
	return conn, errors.Wrap(err, "failed to dial with context")
}

// Setup will apply the provided configuration settings.
func (c *Client) setup(options ...ClientOption) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return errors.WithStack(err)
		}
	}

	// Additional TLs configuration options
	if c.tlsConf != nil {
		if c.nameOverride != "" {
			c.tlsConf.ServerName = c.nameOverride
		}
		if c.skipVerify {
			c.tlsConf.InsecureSkipVerify = true
		}
		if c.cert != nil {
			c.tlsConf.Certificates = []tls.Certificate{*c.cert}
		}
	}
	return nil
}

// Return properly setup client middleware.
func (c *Client) getMiddleware() (unary []grpc.UnaryClientInterceptor, stream []grpc.StreamClientInterceptor) {
	// Setup observability before anything else
	if c.oop != nil {
		ui, si := extras.NewGRPCMonitor().Client()
		unary = append(unary, ui)
		stream = append(stream, si)
	}

	// Add registered middleware
	unary = append(unary, c.middlewareUnary...)
	stream = append(stream, c.middlewareStream...)
	return unary, stream
}

// NewClientConnection creates a new RPC connection with the provided options.
func NewClientConnection(endpoint string, options ...ClientOption) (*grpc.ClientConn, error) {
	c, err := NewClient(options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize client")
	}
	conn, err := c.GetConnection(endpoint)
	return conn, errors.Wrap(err, "failed to establish connection")
}

// MonitorClientConnection enable notifications on connection state change. If no
// interval is provided (i.e. 0) a default value of 2 seconds will be used.
func MonitorClientConnection(ctx context.Context, conn *grpc.ClientConn, ti time.Duration) <-chan connectivity.State {
	// Use a default value, if no internal is provided
	if ti == 0 {
		ti = 2 * time.Second
	}

	monitor := make(chan connectivity.State)
	go func() {
		s := conn.GetState()
		monitor <- s
		ticker := time.NewTicker(ti)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				newState := conn.GetState()
				if newState != s {
					s = newState
					monitor <- s
				}
			case <-ctx.Done():
				close(monitor)
				return
			}
		}
	}()
	return monitor
}
