package rpc

import (
	"context"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func pvUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	// nolint: lll
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		msg, ok := req.(protoreflect.ProtoMessage)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid message type")
		}
		if err := protovalidate.Validate(msg); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return handler(ctx, req)
	}
}

func pvStreamServerInterceptor() grpc.StreamServerInterceptor {
	// nolint: lll
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &recvWrapper{ServerStream: stream})
	}
}

type recvWrapper struct {
	grpc.ServerStream
}

func (s *recvWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if msg, ok := m.(protoreflect.ProtoMessage); ok {
		if err := protovalidate.Validate(msg); err != nil {
			return err
		}
	}
	return nil
}
