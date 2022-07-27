package sentrygrpc

import (
	"context"
	"io"

	"go.bryk.io/pkg/errors"
	apiErrors "go.bryk.io/pkg/otel/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// clientStream  wraps around the embedded grpc.ClientStream, and intercepts
// the RecvMsg and SendMsg method call.
type clientStream struct {
	op            apiErrors.Operation
	ctx           context.Context
	events        chan event
	desc          *grpc.StreamDesc
	done          chan error
	close         chan struct{}
	recvMessageID int
	sentMessageID int
	grpc.ClientStream
}

func (ws *clientStream) RecvMsg(m interface{}) error {
	err := ws.ClientStream.RecvMsg(m)
	if err == nil && !ws.desc.ServerStreams {
		// stream is closed
		ws.events <- event{desc: "stream closed"}
		ws.done <- nil
	} else if errors.Is(err, io.EOF) {
		// stream is closed
		ws.events <- event{desc: "stream closed"}
		ws.done <- nil
	} else if err != nil {
		// report error
		ws.events <- event{err: err}
		ws.done <- err
	} else {
		// report message received
		ws.recvMessageID++
		ws.events <- event{
			id:      ws.recvMessageID,
			desc:    msgRecv,
			payload: m,
		}
	}
	return err
}

func (ws *clientStream) SendMsg(m interface{}) error {
	err := ws.ClientStream.SendMsg(m)
	ws.sentMessageID++
	ws.events <- event{
		id:      ws.sentMessageID,
		desc:    msgSent,
		payload: m,
	}
	if err != nil {
		// report message delivery error
		ws.events <- event{err: err}
		ws.done <- err
	}
	return err
}

func (ws *clientStream) Header() (metadata.MD, error) {
	md, err := ws.ClientStream.Header()
	if err != nil {
		// report error while receiving header metadata
		ws.events <- event{err: err}
		ws.done <- err
	}
	return md, err
}

func (ws *clientStream) CloseSend() error {
	err := ws.ClientStream.CloseSend()
	if err != nil {
		// report error closing the "send" direction of the stream
		ws.events <- event{err: err}
		ws.done <- err
	}
	return err
}

func (ws *clientStream) handleEvents() {
	for {
		select {
		case <-ws.ctx.Done():
			// closed by context
			ws.done <- ws.ctx.Err()
			return
		case <-ws.close:
			// manually closed
			return
		case ev := <-ws.events:
			// report event
			reportEvent(ws.op, ev)
		}
	}
}
