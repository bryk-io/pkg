// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             buf-v1.51.0
// source: sample/v1/echo_api.proto

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
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	EchoAPI_Ping_FullMethodName   = "/sample.v1.EchoAPI/Ping"
	EchoAPI_Health_FullMethodName = "/sample.v1.EchoAPI/Health"
	EchoAPI_Echo_FullMethodName   = "/sample.v1.EchoAPI/Echo"
	EchoAPI_Faulty_FullMethodName = "/sample.v1.EchoAPI/Faulty"
	EchoAPI_Slow_FullMethodName   = "/sample.v1.EchoAPI/Slow"
)

// EchoAPIClient is the client API for EchoAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Echo service main interface.
type EchoAPIClient interface {
	// Reachability test.
	Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Pong, error)
	// Health test.
	Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error)
	// Process an incoming echo request.
	Echo(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error)
	// Returns an error roughly about 20% of the time.
	Faulty(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*DummyResponse, error)
	// Exhibit a random latency between 10 and 200ms.
	Slow(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*DummyResponse, error)
}

type echoAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewEchoAPIClient(cc grpc.ClientConnInterface) EchoAPIClient {
	return &echoAPIClient{cc}
}

func (c *echoAPIClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Pong, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Pong)
	err := c.cc.Invoke(ctx, EchoAPI_Ping_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoAPIClient) Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, EchoAPI_Health_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoAPIClient) Echo(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, EchoAPI_Echo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoAPIClient) Faulty(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*DummyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, EchoAPI_Faulty_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoAPIClient) Slow(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*DummyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, EchoAPI_Slow_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EchoAPIServer is the server API for EchoAPI service.
// All implementations must embed UnimplementedEchoAPIServer
// for forward compatibility.
//
// Echo service main interface.
type EchoAPIServer interface {
	// Reachability test.
	Ping(context.Context, *emptypb.Empty) (*Pong, error)
	// Health test.
	Health(context.Context, *emptypb.Empty) (*HealthResponse, error)
	// Process an incoming echo request.
	Echo(context.Context, *EchoRequest) (*EchoResponse, error)
	// Returns an error roughly about 20% of the time.
	Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error)
	// Exhibit a random latency between 10 and 200ms.
	Slow(context.Context, *emptypb.Empty) (*DummyResponse, error)
	mustEmbedUnimplementedEchoAPIServer()
}

// UnimplementedEchoAPIServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedEchoAPIServer struct{}

func (UnimplementedEchoAPIServer) Ping(context.Context, *emptypb.Empty) (*Pong, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedEchoAPIServer) Health(context.Context, *emptypb.Empty) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedEchoAPIServer) Echo(context.Context, *EchoRequest) (*EchoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Echo not implemented")
}
func (UnimplementedEchoAPIServer) Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Faulty not implemented")
}
func (UnimplementedEchoAPIServer) Slow(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Slow not implemented")
}
func (UnimplementedEchoAPIServer) mustEmbedUnimplementedEchoAPIServer() {}
func (UnimplementedEchoAPIServer) testEmbeddedByValue()                 {}

// UnsafeEchoAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EchoAPIServer will
// result in compilation errors.
type UnsafeEchoAPIServer interface {
	mustEmbedUnimplementedEchoAPIServer()
}

func RegisterEchoAPIServer(s grpc.ServiceRegistrar, srv EchoAPIServer) {
	// If the following call pancis, it indicates UnimplementedEchoAPIServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&EchoAPI_ServiceDesc, srv)
}

func _EchoAPI_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoAPIServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EchoAPI_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoAPIServer).Ping(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoAPI_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoAPIServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EchoAPI_Health_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoAPIServer).Health(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoAPI_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoAPIServer).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EchoAPI_Echo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoAPIServer).Echo(ctx, req.(*EchoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoAPI_Faulty_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoAPIServer).Faulty(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EchoAPI_Faulty_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoAPIServer).Faulty(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoAPI_Slow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoAPIServer).Slow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EchoAPI_Slow_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoAPIServer).Slow(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// EchoAPI_ServiceDesc is the grpc.ServiceDesc for EchoAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EchoAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sample.v1.EchoAPI",
	HandlerType: (*EchoAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _EchoAPI_Ping_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _EchoAPI_Health_Handler,
		},
		{
			MethodName: "Echo",
			Handler:    _EchoAPI_Echo_Handler,
		},
		{
			MethodName: "Faulty",
			Handler:    _EchoAPI_Faulty_Handler,
		},
		{
			MethodName: "Slow",
			Handler:    _EchoAPI_Slow_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "sample/v1/echo_api.proto",
}
