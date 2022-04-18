package amqp

import (
	"log"
)

var consumer *Consumer

func doStuff(_ Delivery) {}

func ExampleNewConsumer() {
	// Create a new consumer instance
	consumer, err := NewConsumer("amqp://guest:guest@localhost:5672")
	if err != nil {
		panic(err)
	}

	// Wait for the consumer to be ready
	<-consumer.Ready()

	// Open a subscription and start working with events
	tasksToHandle, id, err := consumer.Subscribe(SubscribeOptions{Queue: "jobs"})
	if err != nil {
		panic(err)
	}
	log.Printf("subscription open: %s", id)

	// Handle all events received, sending an ACK message back to the
	// broker once the task has been successfully completed to prevent
	// requeue and resending.
	for msg := range tasksToHandle {
		doStuff(msg)
		if err := msg.Ack(false); err != nil {
			log.Printf("failed to process message: %s", err)
		}
	}

	// When no longer needed, close the consumer instance
	if err = consumer.Close(); err != nil {
		panic(err)
	}
}

func ExampleConsumer_AddBinding() {
	err := consumer.AddBinding(Binding{
		Exchange: "topic_exchange",
		Queue:    "my_tasks",
		RoutingKey: []string{
			"tasks.zone.us.#",
			"tasks.zone.eu.#",
		},
	})
	if err != nil {
		panic(err)
	}
}

func ExampleConsumer_AddQueue() {
	_, err := consumer.AddQueue(Queue{
		Name:       "my_temporal_and_exclusive_queue",
		AutoDelete: true,
		Exclusive:  true,
		Durable:    false,
	})
	if err != nil {
		panic(err)
	}
}

func ExampleConsumer_Subscribe() {
	// Open subscription
	deliveries, id, err := consumer.Subscribe(SubscribeOptions{
		Queue:   "my_tasks",
		AutoAck: true,
	})
	if err != nil {
		panic(err)
	}

	// Handle tasks, no need to manually send ACK because "AutoAck"
	// is set to "true"
	for task := range deliveries {
		doStuff(task)
	}

	// Close subscription when no longer need
	// but keep consumer connection
	err = consumer.CloseSubscription(id)
	if err != nil {
		panic(err)
	}
}
