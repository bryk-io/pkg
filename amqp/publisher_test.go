package amqp

import (
	"context"
	"log"
	"time"
)

var publisher *Publisher

func ExampleNewPublisher() {
	// Create a new publisher instance
	publisher, err := NewPublisher("amqp://guest:guest@localhost:5672")
	if err != nil {
		panic(err)
	}

	// Wait for the publisher to be ready
	<-publisher.Ready()

	// Send a sample message
	msg := Message{
		Body:        []byte("hello world"),
		ContentType: "text/plain",
	}
	err = publisher.UnsafePush(msg, MessageOptions{Exchange: "my-exchange"})
	if err != nil {
		log.Printf("push error: %s", err)
	}

	// When no longer needed, close the publisher
	if err = publisher.Close(); err != nil {
		panic(err)
	}
}

func ExamplePublisher_AddExchange() {
	// Create and add definition for the new exchange
	newExchange := Exchange{
		Name:       "custom_notifications",
		Kind:       "fanout",
		Durable:    true,
		AutoDelete: false,
	}
	if err := publisher.AddExchange(newExchange); err != nil {
		panic(err)
	}
}

func ExamplePublisher_GetDispatcher() {
	// All messages send using the dispatcher instance will use
	// the options provided.
	opts := MessageOptions{
		Exchange:   "jobs",
		RoutingKey: "zone=us-west",
		Persistent: true,
		Immediate:  true,
		Mandatory:  true,
	}

	// A context instance allows to manually close the dispatcher
	// when no longer needed
	ctx, cancel := context.WithCancel(context.Background())

	// Create new dispatcher
	importantJobs := publisher.GetDispatcher(ctx, true, opts)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				importantJobs.Publish() <- Message{Body: []byte(time.Now().String())}
			case err := <-importantJobs.Errors():
				log.Printf("error: %s", err)
			case <-importantJobs.Done():
				log.Printf("dispatcher is closed")
				return
			}
		}
	}()

	// Wait for a bit
	<-time.After(10 * time.Second)
	cancel()
}
