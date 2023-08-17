// Code generated by protoc-gen-go-drpc. DO NOT EDIT.
// protoc-gen-go-drpc version: v0.0.33
// source: sample/v1/echo_api.proto

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

type drpcEncoding_File_sample_v1_echo_api_proto struct{}

func (drpcEncoding_File_sample_v1_echo_api_proto) Marshal(msg drpc.Message) ([]byte, error) {
	return proto.Marshal(msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_echo_api_proto) MarshalAppend(buf []byte, msg drpc.Message) ([]byte, error) {
	return proto.MarshalOptions{}.MarshalAppend(buf, msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_echo_api_proto) Unmarshal(buf []byte, msg drpc.Message) error {
	return proto.Unmarshal(buf, msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_echo_api_proto) JSONMarshal(msg drpc.Message) ([]byte, error) {
	return protojson.Marshal(msg.(proto.Message))
}

func (drpcEncoding_File_sample_v1_echo_api_proto) JSONUnmarshal(buf []byte, msg drpc.Message) error {
	return protojson.Unmarshal(buf, msg.(proto.Message))
}

type DRPCEchoAPIClient interface {
	DRPCConn() drpc.Conn

	Ping(ctx context.Context, in *emptypb.Empty) (*Pong, error)
	Health(ctx context.Context, in *emptypb.Empty) (*HealthResponse, error)
	Echo(ctx context.Context, in *EchoRequest) (*EchoResponse, error)
	Faulty(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error)
	Slow(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error)
}

type drpcEchoAPIClient struct {
	cc drpc.Conn
}

func NewDRPCEchoAPIClient(cc drpc.Conn) DRPCEchoAPIClient {
	return &drpcEchoAPIClient{cc}
}

func (c *drpcEchoAPIClient) DRPCConn() drpc.Conn { return c.cc }

func (c *drpcEchoAPIClient) Ping(ctx context.Context, in *emptypb.Empty) (*Pong, error) {
	out := new(Pong)
	err := c.cc.Invoke(ctx, "/sample.v1.EchoAPI/Ping", drpcEncoding_File_sample_v1_echo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcEchoAPIClient) Health(ctx context.Context, in *emptypb.Empty) (*HealthResponse, error) {
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.EchoAPI/Health", drpcEncoding_File_sample_v1_echo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcEchoAPIClient) Echo(ctx context.Context, in *EchoRequest) (*EchoResponse, error) {
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.EchoAPI/Echo", drpcEncoding_File_sample_v1_echo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcEchoAPIClient) Faulty(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error) {
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.EchoAPI/Faulty", drpcEncoding_File_sample_v1_echo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *drpcEchoAPIClient) Slow(ctx context.Context, in *emptypb.Empty) (*DummyResponse, error) {
	out := new(DummyResponse)
	err := c.cc.Invoke(ctx, "/sample.v1.EchoAPI/Slow", drpcEncoding_File_sample_v1_echo_api_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DRPCEchoAPIServer interface {
	Ping(context.Context, *emptypb.Empty) (*Pong, error)
	Health(context.Context, *emptypb.Empty) (*HealthResponse, error)
	Echo(context.Context, *EchoRequest) (*EchoResponse, error)
	Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error)
	Slow(context.Context, *emptypb.Empty) (*DummyResponse, error)
}

type DRPCEchoAPIUnimplementedServer struct{}

func (s *DRPCEchoAPIUnimplementedServer) Ping(context.Context, *emptypb.Empty) (*Pong, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCEchoAPIUnimplementedServer) Health(context.Context, *emptypb.Empty) (*HealthResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCEchoAPIUnimplementedServer) Echo(context.Context, *EchoRequest) (*EchoResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCEchoAPIUnimplementedServer) Faulty(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

func (s *DRPCEchoAPIUnimplementedServer) Slow(context.Context, *emptypb.Empty) (*DummyResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

type DRPCEchoAPIDescription struct{}

func (DRPCEchoAPIDescription) NumMethods() int { return 5 }

func (DRPCEchoAPIDescription) Method(n int) (string, drpc.Encoding, drpc.Receiver, interface{}, bool) {
	switch n {
	case 0:
		return "/sample.v1.EchoAPI/Ping", drpcEncoding_File_sample_v1_echo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCEchoAPIServer).
					Ping(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCEchoAPIServer.Ping, true
	case 1:
		return "/sample.v1.EchoAPI/Health", drpcEncoding_File_sample_v1_echo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCEchoAPIServer).
					Health(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCEchoAPIServer.Health, true
	case 2:
		return "/sample.v1.EchoAPI/Echo", drpcEncoding_File_sample_v1_echo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCEchoAPIServer).
					Echo(
						ctx,
						in1.(*EchoRequest),
					)
			}, DRPCEchoAPIServer.Echo, true
	case 3:
		return "/sample.v1.EchoAPI/Faulty", drpcEncoding_File_sample_v1_echo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCEchoAPIServer).
					Faulty(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCEchoAPIServer.Faulty, true
	case 4:
		return "/sample.v1.EchoAPI/Slow", drpcEncoding_File_sample_v1_echo_api_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCEchoAPIServer).
					Slow(
						ctx,
						in1.(*emptypb.Empty),
					)
			}, DRPCEchoAPIServer.Slow, true
	default:
		return "", nil, nil, nil, false
	}
}

func DRPCRegisterEchoAPI(mux drpc.Mux, impl DRPCEchoAPIServer) error {
	return mux.Register(impl, DRPCEchoAPIDescription{})
}

type DRPCEchoAPI_PingStream interface {
	drpc.Stream
	SendAndClose(*Pong) error
}

type drpcEchoAPI_PingStream struct {
	drpc.Stream
}

func (x *drpcEchoAPI_PingStream) SendAndClose(m *Pong) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_echo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCEchoAPI_HealthStream interface {
	drpc.Stream
	SendAndClose(*HealthResponse) error
}

type drpcEchoAPI_HealthStream struct {
	drpc.Stream
}

func (x *drpcEchoAPI_HealthStream) SendAndClose(m *HealthResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_echo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCEchoAPI_EchoStream interface {
	drpc.Stream
	SendAndClose(*EchoResponse) error
}

type drpcEchoAPI_EchoStream struct {
	drpc.Stream
}

func (x *drpcEchoAPI_EchoStream) SendAndClose(m *EchoResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_echo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCEchoAPI_FaultyStream interface {
	drpc.Stream
	SendAndClose(*DummyResponse) error
}

type drpcEchoAPI_FaultyStream struct {
	drpc.Stream
}

func (x *drpcEchoAPI_FaultyStream) SendAndClose(m *DummyResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_echo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}

type DRPCEchoAPI_SlowStream interface {
	drpc.Stream
	SendAndClose(*DummyResponse) error
}

type drpcEchoAPI_SlowStream struct {
	drpc.Stream
}

func (x *drpcEchoAPI_SlowStream) SendAndClose(m *DummyResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_sample_v1_echo_api_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}
