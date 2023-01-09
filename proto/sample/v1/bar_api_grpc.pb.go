// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             buf-v1.11.0
// source: sample/v1/bar_api.proto

package samplev1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BarAPIClient is the client API for BarAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BarAPIClient interface {
	// Reachability test.
	Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Pong, error)
	// Health test.
	Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error)
	// Sample request.
	Request(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Response, error)
}

type barAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewBarAPIClient(cc grpc.ClientConnInterface) BarAPIClient {
	return &barAPIClient{cc}
}

func (c *barAPIClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Pong, error) {
	out := new(Pong)
	err := c.cc.Invoke(ctx, "/sample.v1.BarAPI/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *barAPIClient) Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error) {
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.BarAPI/Health", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *barAPIClient) Request(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/sample.v1.BarAPI/Request", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BarAPIServer is the server API for BarAPI service.
// All implementations must embed UnimplementedBarAPIServer
// for forward compatibility
type BarAPIServer interface {
	// Reachability test.
	Ping(context.Context, *emptypb.Empty) (*Pong, error)
	// Health test.
	Health(context.Context, *emptypb.Empty) (*HealthResponse, error)
	// Sample request.
	Request(context.Context, *emptypb.Empty) (*Response, error)
	mustEmbedUnimplementedBarAPIServer()
}

// UnimplementedBarAPIServer must be embedded to have forward compatible implementations.
type UnimplementedBarAPIServer struct {
}

func (UnimplementedBarAPIServer) Ping(context.Context, *emptypb.Empty) (*Pong, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedBarAPIServer) Health(context.Context, *emptypb.Empty) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedBarAPIServer) Request(context.Context, *emptypb.Empty) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Request not implemented")
}
func (UnimplementedBarAPIServer) mustEmbedUnimplementedBarAPIServer() {}

// UnsafeBarAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BarAPIServer will
// result in compilation errors.
type UnsafeBarAPIServer interface {
	mustEmbedUnimplementedBarAPIServer()
}

func RegisterBarAPIServer(s grpc.ServiceRegistrar, srv BarAPIServer) {
	s.RegisterService(&BarAPI_ServiceDesc, srv)
}

func _BarAPI_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarAPIServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sample.v1.BarAPI/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarAPIServer).Ping(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _BarAPI_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarAPIServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sample.v1.BarAPI/Health",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarAPIServer).Health(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _BarAPI_Request_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarAPIServer).Request(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sample.v1.BarAPI/Request",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarAPIServer).Request(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// BarAPI_ServiceDesc is the grpc.ServiceDesc for BarAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BarAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sample.v1.BarAPI",
	HandlerType: (*BarAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _BarAPI_Ping_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _BarAPI_Health_Handler,
		},
		{
			MethodName: "Request",
			Handler:    _BarAPI_Request_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "sample/v1/bar_api.proto",
}
