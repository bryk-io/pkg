// Code generated by protoc-gen-go-drpc. DO NOT EDIT.
// protoc-gen-go-drpc version: v0.0.32
// source: sample/v1/foo_api.proto

package samplev1

import (
	context "context"
	errors "errors"

	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	drpc "storj.io/drpc"
	drpcerr "storj.io/drpc/drpcerr"
)

type drpcEncoding_File_sample_v1_foo_api_proto struct{}

func (drpcEncoding_File_sample_v1_foo_api_proto) Marshal(msg drpc.Message) ([]byte, error) {
	return proto.Marshal(msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_foo_api_proto) MarshalAppend(buf []byte, msg drpc.Message) ([]byte, error) {
	return proto.MarshalOptions{}.MarshalAppend(buf, msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_foo_api_proto) Unmarshal(buf []byte, msg drpc.Message) error {
	return proto.Unmarshal(buf, msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_foo_api_proto) JSONMarshal(msg drpc.Message) ([]byte, error) {
	return protojson.Marshal(msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_foo_api_proto) JSONUnmarshal(buf []byte, msg drpc.Message) error {
	return protojson.Unmarshal(buf, msg.(proto.Message))
}

type DRPCFooAPIClient interface {
	DRPCConn() drpc.Conn

	Ping(ctx context.Context, in *emptypb.Empty) (*Pong, error)
	Health(ctx context.Context, in *emptypb.Empty) (*HealthResponse, error)
	Request(ctx context.Context, in *emptypb.Empty) (*Response, error)
	Faulty(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error)
	Slow(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error)
	OpenServerStream(ctx context.Context, in *emptypb.Empty) (DRPCFooAPI_OpenServerStreamClient, error)
	OpenClientStream(ctx context.Context) (DRPCFooAPI_OpenClientStreamClient, error)
}

type drpcFooAPIClient struct {
	cc drpc.Conn
}

func NewDRPCFooAPIClient(cc drpc.Conn) DRPCFooAPIClient {
	return &drpcFooAPIClient{cc}
}

func (c *drpcFooAPIClient) DRPCConn() drpc.Conn { return c.cc }

func (c *drpcFooAPIClient) Ping(ctx context.Context, in *emptypb.Empty) (*Pong, error) {
	out := new(Pong)
	err := c.cc.Invoke(ctx, "/sample.v1.FooAPI/Ping", drpcEncoding_File_sample_v1_foo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcFooAPIClient) Health(ctx context.Context, in *emptypb.Empty) (*HealthResponse, error) {
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.FooAPI/Health", drpcEncoding_File_sample_v1_foo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcFooAPIClient) Request(ctx context.Context, in *emptypb.Empty) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/sample.v1.FooAPI/Request", drpcEncoding_File_sample_v1_foo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcFooAPIClient) Faulty(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error) {
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.FooAPI/Faulty", drpcEncoding_File_sample_v1_foo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcFooAPIClient) Slow(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error) {
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.FooAPI/Slow", drpcEncoding_File_sample_v1_foo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcFooAPIClient) OpenServerStream(ctx context.Context, in *emptypb.Empty) (DRPCFooAPI_OpenServerStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, "/sample.v1.FooAPI/OpenServerStream", drpcEncoding_File_sample_v1_foo_api_proto{})
	if err != nil {
		return nil, err
	}
	x := &drpcFooAPI_OpenServerStreamClient{stream}
	if err := x.MsgSend(in, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return nil, err
	}
	if err := x.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type DRPCFooAPI_OpenServerStreamClient interface {
	drpc.Stream
	Recv() (*GenericStreamChunk, error)
}

type drpcFooAPI_OpenServerStreamClient struct {
	drpc.Stream
}

func (x *drpcFooAPI_OpenServerStreamClient) Recv() (*GenericStreamChunk, error) {
	m := new(GenericStreamChunk)
	if err := x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *drpcFooAPI_OpenServerStreamClient) RecvMsg(m *GenericStreamChunk) error {
	return x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{})
}

func (c *drpcFooAPIClient) OpenClientStream(ctx context.Context) (DRPCFooAPI_OpenClientStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, "/sample.v1.FooAPI/OpenClientStream", drpcEncoding_File_sample_v1_foo_api_proto{})
	if err != nil {
		return nil, err
	}
	x := &drpcFooAPI_OpenClientStreamClient{stream}
	return x, nil
}

type DRPCFooAPI_OpenClientStreamClient interface {
	drpc.Stream
	Send(*OpenClientStreamRequest) error
	CloseAndRecv() (*StreamResult, error)
}

type drpcFooAPI_OpenClientStreamClient struct {
	drpc.Stream
}

func (x *drpcFooAPI_OpenClientStreamClient) Send(m *OpenClientStreamRequest) error {
	return x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{})
}

func (x *drpcFooAPI_OpenClientStreamClient) CloseAndRecv() (*StreamResult, error) {
	if err := x.CloseSend(); err != nil {
		return nil, err
	}
	m := new(StreamResult)
	if err := x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *drpcFooAPI_OpenClientStreamClient) CloseAndRecvMsg(m *StreamResult) error {
	if err := x.CloseSend(); err != nil {
		return err
	}
	return x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{})
}

type DRPCFooAPIServer interface {
	Ping(context.Context, *emptypb.Empty) (*Pong, error)
	Health(context.Context, *emptypb.Empty) (*HealthResponse, error)
	Request(context.Context, *emptypb.Empty) (*Response, error)
	Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error)
	Slow(context.Context, *emptypb.Empty) (*DummyResponse, error)
	OpenServerStream(*emptypb.Empty, DRPCFooAPI_OpenServerStreamStream) error
	OpenClientStream(DRPCFooAPI_OpenClientStreamStream) error
}

type DRPCFooAPIUnimplementedServer struct{}

func (s *DRPCFooAPIUnimplementedServer) Ping(context.Context, *emptypb.Empty) (*Pong, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) Health(context.Context, *emptypb.Empty) (*HealthResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) Request(context.Context, *emptypb.Empty) (*Response, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) Slow(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) OpenServerStream(*emptypb.Empty, DRPCFooAPI_OpenServerStreamStream) error {
	return drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCFooAPIUnimplementedServer) OpenClientStream(DRPCFooAPI_OpenClientStreamStream) error {
	return drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

type DRPCFooAPIDescription struct{}

func (DRPCFooAPIDescription) NumMethods() int { return 7 }

func (DRPCFooAPIDescription) Method(n int) (string, drpc.Encoding, drpc.Receiver, interface{}, bool) {
	switch n {
	case 0:
		return "/sample.v1.FooAPI/Ping", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCFooAPIServer).
					Ping(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCFooAPIServer.Ping, true
	case 1:
		return "/sample.v1.FooAPI/Health", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCFooAPIServer).
					Health(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCFooAPIServer.Health, true
	case 2:
		return "/sample.v1.FooAPI/Request", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCFooAPIServer).
					Request(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCFooAPIServer.Request, true
	case 3:
		return "/sample.v1.FooAPI/Faulty", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCFooAPIServer).
					Faulty(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCFooAPIServer.Faulty, true
	case 4:
		return "/sample.v1.FooAPI/Slow", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCFooAPIServer).
					Slow(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCFooAPIServer.Slow, true
	case 5:
		return "/sample.v1.FooAPI/OpenServerStream", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return nil, srv.(DRPCFooAPIServer).
					OpenServerStream(
						in1.(*emptypb.Empty),
						&drpcFooAPI_OpenServerStreamStream{in2.(drpc.Stream)},
					)
			}, DRPCFooAPIServer.OpenServerStream, true
	case 6:
		return "/sample.v1.FooAPI/OpenClientStream", drpcEncoding_File_sample_v1_foo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return nil, srv.(DRPCFooAPIServer).
					OpenClientStream(
						&drpcFooAPI_OpenClientStreamStream{in1.(drpc.Stream)},
					)
			}, DRPCFooAPIServer.OpenClientStream, true
	default:
		return "", nil, nil, nil, false
	}
}

func DRPCRegisterFooAPI(mux drpc.Mux, impl DRPCFooAPIServer) error {
	return mux.Register(impl, DRPCFooAPIDescription{})
}

type DRPCFooAPI_PingStream interface {
	drpc.Stream
	SendAndClose(*Pong) error
}

type drpcFooAPI_PingStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_PingStream) SendAndClose(m *Pong) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCFooAPI_HealthStream interface {
	drpc.Stream
	SendAndClose(*HealthResponse) error
}

type drpcFooAPI_HealthStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_HealthStream) SendAndClose(m *HealthResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCFooAPI_RequestStream interface {
	drpc.Stream
	SendAndClose(*Response) error
}

type drpcFooAPI_RequestStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_RequestStream) SendAndClose(m *Response) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCFooAPI_FaultyStream interface {
	drpc.Stream
	SendAndClose(*DummyResponse) error
}

type drpcFooAPI_FaultyStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_FaultyStream) SendAndClose(m *DummyResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCFooAPI_SlowStream interface {
	drpc.Stream
	SendAndClose(*DummyResponse) error
}

type drpcFooAPI_SlowStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_SlowStream) SendAndClose(m *DummyResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCFooAPI_OpenServerStreamStream interface {
	drpc.Stream
	Send(*GenericStreamChunk) error
}

type drpcFooAPI_OpenServerStreamStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_OpenServerStreamStream) Send(m *GenericStreamChunk) error {
	return x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{})
}

type DRPCFooAPI_OpenClientStreamStream interface {
	drpc.Stream
	SendAndClose(*StreamResult) error
	Recv() (*OpenClientStreamRequest, error)
}

type drpcFooAPI_OpenClientStreamStream struct {
	drpc.Stream
}

func (x *drpcFooAPI_OpenClientStreamStream) SendAndClose(m *StreamResult) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

func (x *drpcFooAPI_OpenClientStreamStream) Recv() (*OpenClientStreamRequest, error) {
	m := new(OpenClientStreamRequest)
	if err := x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{}); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *drpcFooAPI_OpenClientStreamStream) RecvMsg(m *OpenClientStreamRequest) error {
	return x.MsgRecv(m, drpcEncoding_File_sample_v1_foo_api_proto{})
}
