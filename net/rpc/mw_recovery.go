package rpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func rcvUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		panicked := true

		defer func() {
			if r := recover(); r != nil || panicked {
				err = status.Errorf(codes.Internal, "%v", r)
			}
		}()

		resp, err := handler(ctx, req)
		panicked = false
		return resp, err
	}
}

func rcvStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		panicked := true

		defer func() {
			if r := recover(); r != nil || panicked {
				err = status.Errorf(codes.Internal, "%v", r)
			}
		}()

		err = handler(srv, stream)
		panicked = false
		return err
	}
}
