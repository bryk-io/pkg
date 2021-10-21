/*
Package rpc provides a utilities and components to easily deploy a robust RPC network service.

RPC stands for Remote Procedure Call and its an architecture style for distributed systems in
which a system causes a procedure (subroutine) to execute in a different address space (commonly
on another computer on a network). This package use gRPC as its underlying high-performance
framework.

Deploying a gRPC service in production involves several different aspects and considerations.
Things like: logging, authentication, secure communication channels, request tracing, manage
server resources, support regular HTTP (REST) access, etc. This package greatly simplifies
the process of properly configuring and running a production grade RPC service.

More information:
https://grpc.io/

Server

For the server implementation the main component is using a 'Server' instance. The server is
configured, using functional style parameters, by providing a list of options to the 'NewServer'
and/or 'Setup' methods.

For example, let's create and start a server using some common configuration options.

	// Server configuration options
	settings := []ServerOption{
		WithLogger(nil),
		WithPanicRecovery(),
		WithServiceProvider(yourServiceHandler),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	// Create new server
	server, _ := NewServer(settings...)

	// Start the server instance and wait for it to be ready
	ready := make(chan bool)
	go server.Start(ready)
	<-ready

	// Server is ready now

Services

The most important configuration setting for a server instance are the "Services" it
supports. You can provide services either by providing a service definition using the
"WithService" option, or by implementing the "ServiceProvider" interface in your
application and passing it along using the "WithServiceProvider" option.

	// Echo service provider (i.e., implementing the "ServiceProvider" interface.)
	type echoProvider struct{}

	func (ep *echoProvider) ServerSetup(server *grpc.Server) {
		samplev1.RegisterEchoAPIServer(server, &samplev1.EchoHandler{})
	}

	func (ep *echoProvider) GatewaySetup() GatewayRegister {
		return samplev1.RegisterEchoAPIHandlerFromEndpoint
	}

	// Base server configuration options
	serverOpts := []ServerOption{
		WithPanicRecovery(),
		WithServiceProvider(&echoProvider{}),
	}

Client

In order to interact with an RPC server and access the provided functionality you need
to setup and establish a client connection. A client connection should be be closed when
no longer needed to free the used resources. A connection can also be monitored to detect
any changes in its current state.

A connection could be obtained from a client instance. The benefit of this approach is that
a single client instance can be used to generate multiple connections to different servers.

	// client options
	options := []ClientOption{
		WaitForReady(),
		WithTimeout(1 * time.Second),
	}
	client, err := NewClient(options...)
	if err != nil {
		panic(err)
	}

	// Use client to get a connection
	conn, err := client.GetConnection("server.com:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

For simpler use cases a connection can be directly obtained using the 'NewClientConnection'
method.

	// client options
	options := []ClientOption{
		WaitForReady(),
		WithTimeout(1 * time.Second),
	}

	// Get connection
	conn, err := NewClientConnection("server.com:9090", options...)
	if err != nil {
		panic(err)
	}

	// Use connection

	// Close it when not needed anymore
	defer conn.Close()

Regardless of how a connection is created you can setup a monitor for it using the
'MonitorClientConnection' method. The monitor instance can be properly terminated using
the provided context.

	// Get a monitor instance with a 5 second check interval
	ctx, close := context.WithCancel(context.TODO())
	defer close()
	monitor := MonitorClientConnection(ctx, conn, 5*time.Second)

	// Close the monitor in the background after 15 seconds
	go func() {
		<-time.After(15*time.Second)
		close()
	}

	// Catch changes in the connection state
	for state := range monitor {
		fmt.Printf("connection state: %s", state)
	}

For more information about functional style configuration options check the original article
by Dave Cheney: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis.

*/
package rpc
