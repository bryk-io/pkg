package ws

import (
	"context"
	"io"
	"time"

	"github.com/gorilla/websocket"
	"go.bryk.io/pkg/errors"
	"storj.io/drpc"
	"storj.io/drpc/drpchttp"
)

type wrappedStream struct {
	ctx  context.Context // original request context
	conn *websocket.Conn // websocket connection
	done bool            // already closed
	json bool            // use JSON encoding?
	init bool            // already initialized on receiver end?
}

// Context obtained on the incoming HTTP request.
func (st *wrappedStream) Context() context.Context {
	return st.ctx
}

// CloseSend signals to the remote that we will no longer send any messages.
func (st *wrappedStream) CloseSend() error {
	if st.done {
		return nil
	}
	st.done = true
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "no more messages")
	return st.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(1*time.Second))
}

// Close the request stream.
func (st *wrappedStream) Close() error {
	if st.done {
		return nil
	}
	st.done = true
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
	_ = st.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(1*time.Second))
	return st.conn.Close()
}

// MsgSend sends the message to the remote.
func (st *wrappedStream) MsgSend(msg drpc.Message, enc drpc.Encoding) error {
	data, err := st.encode(msg, enc)
	if err != nil {
		return err
	}
	return st.conn.WriteMessage(st.messageType(), data)
}

// MsgRecv receives a message from the remote.
func (st *wrappedStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) error {
	// Initialize stream and handle empty messages
	if !st.init {
		st.init = true
		if size, _ := enc.Marshal(msg); len(size) == 0 {
			return nil // no data is expected, continue
		}
	}

	// Receive data from client
	_, data, err := st.conn.ReadMessage()
	if err != nil {
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.CloseNormalClosure {
			if st.done {
				st.done = true
			}
			return io.EOF
		}
	}
	return st.decode(msg, enc, data)
}

func (st *wrappedStream) encode(msg drpc.Message, enc drpc.Encoding) ([]byte, error) {
	if st.json {
		return drpchttp.JSONMarshal(msg, enc)
	}
	return enc.Marshal(msg)
}

func (st *wrappedStream) decode(msg drpc.Message, enc drpc.Encoding, data []byte) error {
	if st.json {
		return drpchttp.JSONUnmarshal(data, msg, enc)
	}
	return enc.Unmarshal(data, msg)
}

func (st *wrappedStream) messageType() int {
	if st.json {
		return websocket.TextMessage
	}
	return websocket.BinaryMessage
}
