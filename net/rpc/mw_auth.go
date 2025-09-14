package rpc

import (
	"context"

	"google.golang.org/grpc"
)

// If a service implements the AuthFuncOverride method, it takes precedence over the
// `AuthFunc` method, and will be called instead of AuthFunc for all method invocations
// within that service.
type authFuncOverride interface {
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

func authUnaryServerInterceptor(authFunc authFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var newCtx context.Context
		var err error
		if overrideSrv, ok := info.Server.(authFuncOverride); ok {
			newCtx, err = overrideSrv.AuthFuncOverride(ctx, info.FullMethod)
		} else {
			newCtx, err = authFunc(ctx)
		}
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

func authStreamServerInterceptor(authFunc authFunc) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error
		newCtx, err = authFunc(stream.Context())
		if overrideSrv, ok := srv.(authFuncOverride); ok {
			newCtx, err = overrideSrv.AuthFuncOverride(stream.Context(), info.FullMethod)
		}
		if err != nil {
			return err
		}
		wrapped := wrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}

// thin wrapper around `grpc.ServerStream` that allows modifying context.
type wrappedServerStream struct {
	grpc.ServerStream
	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext, overwriting the nested
// `grpc.ServerStream.Context()`.
func (w *wrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream returns a ServerStream that has the ability to overwrite context.
func wrapServerStream(stream grpc.ServerStream) *wrappedServerStream {
	if ws, ok := stream.(*wrappedServerStream); ok {
		return ws
	}
	return &wrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}
