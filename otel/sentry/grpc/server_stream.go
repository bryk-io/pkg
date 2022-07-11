package sentrygrpc

import (
	"context"

	apiErrors "go.bryk.io/pkg/otel/errors"
	"google.golang.org/grpc"
)

// serverStream wraps around the embedded grpc.ServerStream, and intercepts
// the RecvMsg and SendMsg method call.
type serverStream struct {
	op            apiErrors.Operation
	ctx           context.Context
	events        chan event
	close         chan struct{}
	recvMessageID int
	sentMessageID int
	grpc.ServerStream
}

func (ws *serverStream) Context() context.Context {
	return ws.ctx
}

func (ws *serverStream) RecvMsg(m interface{}) error {
	err := ws.ServerStream.RecvMsg(m)
	if err == nil {
		ws.recvMessageID++
		ws.events <- event{
			id:      ws.recvMessageID,
			desc:    msgRecv,
			payload: m,
		}
	}
	return err
}

func (ws *serverStream) SendMsg(m interface{}) error {
	err := ws.ServerStream.SendMsg(m)
	ws.sentMessageID++
	ws.events <- event{
		id:      ws.sentMessageID,
		desc:    msgSent,
		payload: m,
	}
	return err
}

func (ws *serverStream) handleEvents() {
	for {
		select {
		case <-ws.ctx.Done():
			// closed by context
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
