package rpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// GatewayRegisterFunc provides a mechanism to set up an HTTP mux for a gRPC server.
type GatewayRegisterFunc = func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error

// ServiceProvider is an entity that provides functionality to be exposed
// through an RPC server.
type ServiceProvider interface {
	// ServerSetup should perform any initialization requirements for the
	// particular service and register it with the provided server instance.
	ServerSetup(server *grpc.Server)

	// GatewaySetup should return the HTTP setup method or 'nil' if the service
	// has no HTTP support.
	GatewaySetup() GatewayRegisterFunc
}
