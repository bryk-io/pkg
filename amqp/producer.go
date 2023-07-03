package amqp

import (
	"time"

	"github.com/google/uuid"
)

// Producer instances simplify the process of generating multiple message
// wrappers with consistent properties and behavior.
type Producer struct {
	// Content encoding code.
	Encoding string

	// Content type code.
	ContentType string

	// Message kind identifier.
	MessageType string

	// Application contextual identifier.
	AppID string

	// Add the timestamp value when creating a new message.
	SetTime bool

	// Add a randomly-generated, unique ID to each message.
	SetID bool
}

// Message returns a message wrapper for the provided content based on the
// producer instance settings.
func (p *Producer) Message(content []byte) Message {
	msg := Message{
		AppId:           p.AppID,
		ContentType:     p.ContentType,
		ContentEncoding: p.Encoding,
		Body:            content,
		Type:            p.MessageType,
	}
	if p.SetID {
		msg.MessageId = uuid.New().String()
	}
	if p.SetTime {
		msg.Timestamp = time.Now().UTC()
	}
	return msg
}
