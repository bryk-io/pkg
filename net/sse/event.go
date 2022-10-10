package sse

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.bryk.io/pkg/errors"
)

// Event instances are the minimal communication unit between
// the server/publisher and any clients/subscribers.
// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#fields
type Event struct {
	// Event identifier.
	id int

	// A string identifying the type of event described. If this is specified, an
	// event will be dispatched on the browser to the listener for the specified
	// event name; the website source code should use `addEventListener()` to listen
	// for named events. The `onmessage` handler is called if no event name is
	// specified for a message.
	name string

	// Payload/contents for the event/message.
	data interface{}

	// Reconnection time. If the connection to the server is lost, the browser
	// will wait for the specified time (in milliseconds) before attempting to
	// reconnect.
	retry uint
}

// ID return the event's unique identifier.
func (e Event) ID() int {
	return e.id
}

// Name returns the event's type identifier, if any. Events with no type
// value are considered messages by the spec.
func (e Event) Name() string {
	return e.name
}

// Data returns the event's payload.
func (e Event) Data() interface{} {
	return e.data
}

// Encode the event in the proper HTTP transmission format.
func (e Event) Encode() ([]byte, error) {
	if e.name == "" && e.data == nil {
		return nil, errors.New("invalid event")
	}
	buf := bytes.NewBuffer(nil)
	_, _ = buf.Write([]byte("id: " + fmt.Sprintf("%d\n", e.id)))
	if e.data != nil {
		js, err := json.Marshal(e.data)
		if err != nil {
			return nil, err
		}
		_, _ = buf.Write([]byte("data: " + fmt.Sprintf("%s\n", js)))
	}
	if e.retry != 0 {
		_, _ = buf.Write([]byte("retry: " + fmt.Sprintf("%d\n", e.retry)))
	}
	if e.name != "" {
		_, _ = buf.Write([]byte("event: " + e.name + "\n"))
	}
	_, _ = buf.Write([]byte("\n\n"))
	return buf.Bytes(), nil
}

// Decode the event data into the provided `target` element. If `target`
// is `nil` or not a pointer this method returns `json.InvalidUnmarshalError`.
func (e Event) Decode(target interface{}) error {
	js, err := json.Marshal(e.data)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, target)
}
