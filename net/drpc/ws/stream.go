package ws

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"storj.io/drpc"
	"storj.io/drpc/drpchttp"
)

type wrappedStream struct {
	ctx  context.Context // original request context
	conn *websocket.Conn // websocket connection
	json bool            // use JSON encoding?
	init bool            // already initialized on receiver end?
}

// Context obtained on the incoming HTTP request.
func (st *wrappedStream) Context() context.Context {
	return st.ctx
}

// CloseSend signals to the remote that we will no longer send any messages.
func (st *wrappedStream) CloseSend() error {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "no more messages")
	return st.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(1*time.Second))
}

// Close the request stream.
func (st *wrappedStream) Close() error {
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
	// FIXME: Improve initialization and support 'CloseAndRecv'
	// There's a problem with server side streaming only.
	// - The original RPC request triggers this method but blocks since no
	//   data is written by the client on the socket connection.
	// - A "solution" for now is simply to ignore the first original request.
	// - The following 'MsgRecv' invocations occur only when the client
	//   actual sends data, hence preventing blocking and/or corrupting the
	//   socket connection.
	if !st.init {
		st.init = true
		return nil
	}
	_, data, err := st.conn.ReadMessage()
	if err != nil {
		return err
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

func (st *wrappedStream) process() {
	// Automatically close connection when the request's
	// context completes.
	<-st.ctx.Done()
	_ = st.Close()
}
