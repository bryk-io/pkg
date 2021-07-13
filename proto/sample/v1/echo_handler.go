package samplev1

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

// EchoHandler defines a sample implementation of the echo service provider.
type EchoHandler struct {
	UnimplementedEchoAPIServer
}

// Ping provides a sample reachability test method.
func (eh *EchoHandler) Ping(_ context.Context, _ *empty.Empty) (*Pong, error) {
	return &Pong{Ok: true}, nil
}

// Health provides a sample health check method.
func (eh *EchoHandler) Health(_ context.Context, _ *empty.Empty) (*HealthResponse, error) {
	return &HealthResponse{Alive: true}, nil
}

// Echo will process the incoming echo operation.
func (eh *EchoHandler) Echo(_ context.Context, req *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{
		Result: fmt.Sprintf("you said: %s", req.Value),
	}, nil
}

// Faulty will return an error about 20% of the time.
func (eh *EchoHandler) Faulty(_ context.Context, _ *empty.Empty) (*DummyResponse, error) {
	if x := ri(1, 9); x == 2 || x == 4 {
		return nil, status.Error(codes.Internal, "dummy error")
	}
	return &DummyResponse{Ok: true}, nil
}

// Slow will report a random latency between 10 to 200ms.
func (eh *EchoHandler) Slow(_ context.Context, _ *empty.Empty) (*DummyResponse, error) {
	time.Sleep(time.Duration(ri(10, 200)) * time.Millisecond)
	return &DummyResponse{Ok: true}, nil
}
