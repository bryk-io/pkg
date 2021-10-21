package rpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// GatewayRegister provides a mechanism to setup an HTTP mux for a gRPC server.
type GatewayRegister func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// ServiceProvider is an entity that provides functionality to be exposed
// through an RPC server.
type ServiceProvider interface {
	// ServerSetup should perform any initialization requirements for the
	// particular service and register it with the provided server instance.
	ServerSetup(server *grpc.Server)

	// GatewaySetup should return the HTTP setup method or 'nil' if the service
	// has no HTTP support.
	GatewaySetup() GatewayRegister
}

// Service represents a given application being accessed through an RPC server.
type Service struct {
	// The setup method should perform any initialization requirements for the
	// particular service and register it with the provided server instance.
	ServerSetup func(server *grpc.Server)

	// The gateway setup method is the HTTP handler registry function for the
	// service.
	GatewaySetup GatewayRegister
}
