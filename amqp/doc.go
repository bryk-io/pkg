/*
Package amqp simplify working with an "Advanced Message Queue Protocol" based broker.

AMQP allows an application to support advanced asynchronous communication schemas,
which in turn greatly simplify complex functional models on areas like horizontal
scaling, load balancing, interoperability, among others. The base design principles
are simple: brokers receive messages from publishers (applications that publish
them, also known as producers) and route them to consumers (applications that process
them). Since it is a network protocol, the publishers, consumers and the broker can
all reside on different machines and locations.

More precisely, the AMQP model has the following view of the world: messages are
published to exchanges, which are often compared to post offices or mailboxes.
Exchanges then distribute message copies to queues using rules called bindings.
The broker then deliver messages to consumers subscribed to queues.

Networks are unreliable and applications may fail to process messages, therefore the
AMQP model has a notion of message acknowledgements: when a message is delivered
the consumer notifies the broker. When message acknowledgements are in use, a broker
will only completely remove a message from a queue when it receives a notification for
that message.

In certain situations, e.g., when a message cannot be routed, messages may be returned
to publishers, dropped, or handled in some other specified way.

For more information:
https://www.rabbitmq.com/tutorials/amqp-concepts.html

Topology

Exchanges, Queues and Bindings are entities that describe the architecture and
expected behavior of an AMQP environment. This is often refer to as the "Topology"
of the system. The topology definitions can be stored and shared on JSON or YAML
format.

	queues:
	  - name: tasks
	  - name: notifications
	  - name: by_topic
	exchanges:
	  - name: sample.notifications
	    kind: fanout
	  - name: sample.tasks
	    kind: direct
	  - name: sample.topic
	    kind: topic
	bindings:
	  - exchange: sample.notifications
	    queue: notifications
	  - exchange: sample.tasks
	    queue: tasks
	    routing_key:
	      - foo
	      - bar
	  - exchange: sample.topic
	    queue: by_topic
	    routing_key:
	      - stock.nyc.#

Publishers

Publishers are applications that send messages to an exchange in the broker server.
When creating a new publisher, the instance will automatically monitor and handle
its network connection with the broker server. Its interface expose methods to handle
events, dynamically add new exchanges, and publish/push messages.

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

A more convenient way to interact with a publisher, specially when expecting to send
a large number of messages, is through the use of "Dispatcher" instances.

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

Consumers

Consumers are applications that asynchronously receive messages published to
queues they are interested in. Each consumer instance will automatically monitor
and handle its network connection with the broker server. Its interface expose
methods to handle events and dynamically add new queues and bindings. To start
receiving events a user must open a "Subscription". Each subscription returns
its unique ID and a channel to receive "Delivery" messages from. Each message
must send back an "ACK" signal back to the broker to prevent requeue or resend.
The "ACK" signals can be sent either manually or automatically (if the subscription
is opened with the "AutoAck" option).

	// Create a new consumer instance
	consumer, err := NewConsumer("amqp://guest:guest@localhost:5672")
	if err != nil {
		panic(err)
	}

	// Wait for the consumer connection to be ready
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

*/
package amqp
