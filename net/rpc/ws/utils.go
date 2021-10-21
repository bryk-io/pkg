package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

type chuckWrapper struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// Remove the result wrapper added by the gateway to stream chunks.
// https://github.com/grpc-ecosystem/grpc-gateway/blob/master/runtime/handler.go#L189
func removeResultWrapper(chunk []byte) []byte {
	wp := &chuckWrapper{}
	if err := json.Unmarshal(chunk, wp); err != nil {
		return chunk
	}
	if wp.Result != nil {
		data, err := json.Marshal(wp.Result)
		if err == nil && data != nil {
			return data
		}
	}
	return chunk
}

// IE and Edge do not delimit Sec-WebSocket-Protocol strings with spaces.
func fixProtocolHeader(header string) string {
	tokens := strings.SplitN(header, "Bearer,", 2)
	if len(tokens) < 2 {
		return ""
	}
	return fmt.Sprintf("Bearer %v", strings.Trim(tokens[1], " "))
}

func isClosedConnError(err error) bool {
	if str := err.Error(); strings.Contains(str, "use of closed network connection") {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}
