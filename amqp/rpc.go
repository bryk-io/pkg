package amqp

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

// RPC interface used to abstract additional functionality
// required by either subscribers and publishers.
type rpc struct {
	consumer  *Consumer               // dedicated consumer connection
	publisher *Publisher              // dedicated publisher connection
	mode      string                  // instance mode based on its parent handler
	sink      string                  // exclusive queue to handle RPC responses
	resp      map[string]chan Message // response handlers
	ctx       context.Context         // parent publisher context
	incoming  <-chan Delivery         // subscription for response messages
	log       xlog.Logger             // internal logger
	mu        sync.RWMutex
}

// Return underlying connection status.
func (r *rpc) isReady() bool {
	switch r.mode {
	case "pub":
		if r.consumer == nil {
			return false
		}
		return r.consumer.session.isReady()
	case "sub":
		if r.publisher == nil {
			return false
		}
		return r.publisher.session.isReady()
	}
	return false
}

// Return the queue used by the RPC instance to wait
// for responses.
func (r *rpc) queue() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sink
}

// Close RPC channel.
func (r *rpc) close() error {
	var err error
	switch r.mode {
	case "pub":
		err = r.consumer.Close()
		<-r.consumer.ctx.Done()
	case "sub":
		err = r.publisher.Close()
		<-r.publisher.ctx.Done()
	}
	return err
}

// Main event processing.
func (r *rpc) eventLoop() {
	for {
		select {
		// Close loop when RPCs context is done
		case <-r.ctx.Done():
			return
		// Setup RPC queue when it's connection becomes available.
		// This will create a new ephemeral queue each time the
		// publisher re-connects after losing connectivity with the broker.
		case <-r.consumer.Ready():
			if err := r.setupQueue(); err != nil {
				r.consumer.log.WithField("error", err.Error()).Warning("failed to setup RPC queue")
			}
		}
	}
}

// Return a new response handler.
func (r *rpc) responseHandler(ctx context.Context, id string) <-chan Message {
	// Register handler
	handler := make(chan Message, 1)
	r.mu.Lock()
	r.resp[id] = handler
	r.mu.Unlock()

	// De-register handler.
	go func(ctx context.Context, id string, h chan Message) {
		select {
		// RPC was closed
		case <-r.ctx.Done():
			break
		// Request context is done
		case <-ctx.Done():
			break
		// Handler was closed
		case _, ok := <-h:
			if !ok {
				break
			}
		}
		r.mu.Lock()
		delete(r.resp, id)
		r.mu.Unlock()
	}(ctx, id, handler)
	return handler
}

// Publish a response to the "replyTo" queue using the default
// exchange for routing.
func (r *rpc) submitResponse(msg Message, replyTo string) error {
	if r.publisher == nil {
		return errors.New("RPC not enabled to submit responses")
	}
	status, err := r.publisher.Push(msg, MessageOptions{RoutingKey: replyTo})
	if err != nil {
		return err
	}
	if !status {
		return errors.New("failed to submit RPC response")
	}
	return nil
}

// Handle RPC responses received.
func (r *rpc) handleResponses() {
	for resp := range r.incoming {
		// Select response handler
		r.mu.Lock()
		handler, ok := r.resp[resp.CorrelationId]
		r.mu.Unlock()

		// Submit response or warning.
		// To simplify usage, the received message is "auto-ack" so unpack
		// the message out of the "Delivery" instance before submitting it
		// as response.
		if ok {
			handler <- deliveryToMessage(resp)
			close(handler)
			continue
		}
		r.log.WithField("request-id", resp.CorrelationId).Warning("unknown RPC request")
	}
}

// Setup RPC response queue.
func (r *rpc) setupQueue() error {
	r.log.Debug("setup RPC queue")
	name, err := r.consumer.AddQueue(Queue{
		Name:       getName(r.consumer.session.name),
		Durable:    false,
		Exclusive:  true,
		AutoDelete: true,
	})
	if err != nil {
		return err
	}

	// Open deliveries subscription
	deliveries, id, err := r.consumer.Subscribe(SubscribeOptions{
		Queue:     name,
		AutoAck:   true,
		Exclusive: true,
	})
	if err != nil {
		return err
	}

	// Update state
	r.mu.Lock()
	r.sink = name
	r.incoming = deliveries
	r.mu.Unlock()

	// Start processing responses
	go r.handleResponses()
	r.log.WithFields(xlog.Fields{
		"queue":    name,
		"consumer": id,
	}).Info("RPC queue ready")
	return nil
}

// Helper method to "unpack" a message instance out of its
// delivery wrapper.
func deliveryToMessage(d Delivery) Message {
	return Message{
		Headers:         d.Headers,
		ContentType:     d.ContentType,
		ContentEncoding: d.ContentEncoding,
		DeliveryMode:    d.DeliveryMode,
		Priority:        d.Priority,
		CorrelationId:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Expiration:      d.Expiration,
		MessageId:       d.MessageId,
		Timestamp:       d.Timestamp,
		Type:            d.Type,
		UserId:          d.UserId,
		AppId:           d.AppId,
		Body:            d.Body,
	}
}
