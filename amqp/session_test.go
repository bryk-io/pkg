package amqp

import (
	"context"
	"math/rand"
	"net/http"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
	"go.uber.org/goleak"
	"gopkg.in/yaml.v3"
)

var sampleTopology = `
exchanges:
- name: sample.dead
  kind: direct
- name: sample.tasks
  kind: direct
  durable: true
- name: sample.notifications
  kind: fanout
  durable: true
- name: sample.topic
  kind: topic
- name: sample.headers
  kind: headers
- name: sample.rpc
  kind: direct
queues:
- name: hello
- name: rpc
- name: tasks
  arguments:
    x-message-ttl: 10000
    x-expires: 360000
    x-max-length: 100
    x-max-length-bytes: 102400
    x-overflow: "reject-publish-dlx"
    x-dead-letter-exchange: sample.dead
    x-max-priority: 4
    x-queue-mode: "lazy"
- name: notifications
- name: by_topic
- name: by_headers
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
- exchange: sample.headers
  queue: by_headers
  arguments:
    foo: bar
    dimension: c137
    x-match: any
- exchange: sample.rpc
  queue: rpc
`

var sampleProducer *Producer

func init() {
	sampleProducer = &Producer{
		MessageType: "test",
		ContentType: "text/plain",
		Encoding:    "txt",
		AppID:       "golang/testing",
		SetTime:     true,
		SetID:       true,
	}
}

// Generate a random message.
func randomMessage() Message {
	seed := make([]byte, 6)
	_, _ = rand.Read(seed)
	return sampleProducer.Message(seed)
}

// Handle a subscription channel.
func handleDeliveries(ch <-chan Delivery, ll xlog.Logger) {
	ll.Info("start processing deliveries")
	for msg := range ch {
		// process message in some way
		ll.WithFields(xlog.Fields{
			"id":       msg.MessageId,
			"consumer": msg.ConsumerTag,
		}).Debug("message received")

		// random fake latency
		<-time.After(time.Duration(rand.Intn(100)) * time.Millisecond)

		// acknowledge message to mark it as `handled`
		if err := msg.Ack(false); err != nil {
			ll.WithField("error", err.Error()).Warning("failed to ack a received message")
		}
	}
	ll.Warning("closing deliveries processing loop")
}

// Use a dispatcher channel to periodically send random messages.
func handleDispatcher(dp *Dispatcher) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-dp.Done():
			return
		case err := <-dp.Errors():
			if err != nil {
				dp.parent.log.WithField("error", err.Error()).Warning("dispatch error")
			}
		case <-ticker.C:
			dp.Publish() <- randomMessage()
		}
	}
}

// Handle consumer event processing.
func consumerEvents(cc *Consumer, workers int, opts SubscribeOptions) {
	for {
		select {
		case <-cc.ctx.Done():
			return
		case <-cc.Pause():
			cc.log.Debug("consumer became unavailable")
		case <-cc.Ready():
			cc.log.Debug("consumer is available")
			for i := 1; i <= workers; i++ {
				cc.log.Debug("opening worker process to handle deliveries")
				deliveries, id, err := cc.Subscribe(opts)
				if err != nil {
					cc.log.Warning("failed to open subscription")
				} else {
					cc.log.WithField("id", id).Info("subscription open")
					go handleDeliveries(deliveries, cc.log)
				}
			}
		}
	}
}

// Create a temporary queue and binding to receive messages
// from a fanout exchange.
func temporaryQueue(c *Consumer) error {
	// Declare a temporary queue with a random name and connect
	// it to the "fanout" exchange.
	qn, err := c.AddQueue(Queue{Exclusive: true})
	if err != nil {
		return errors.Wrap(err, "failed to add queue")
	}
	err = c.AddBinding(Binding{
		Queue:    qn,
		Exchange: "sample.notifications",
	})
	if err != nil {
		return errors.Wrap(err, "failed to add binding")
	}

	// Open a subscription in the new queue to receive message
	s1, _, err := c.Subscribe(SubscribeOptions{Queue: qn})
	if err != nil {
		return errors.Wrap(err, "failed to open subscription")
	}
	go func() {
		for msg := range s1 {
			c.log.WithFields(xlog.Fields{
				"id":       msg.MessageId,
				"consumer": msg.ConsumerTag,
			}).Debug("message received")
			if err := msg.Ack(false); err != nil {
				c.log.Warning("failed to ACK")
			}
		}
	}()
	return nil
}

// Handle publisher event processing.
func publisherEvents(ctx context.Context, pub *Publisher, opts MessageOptions) {
	for {
		select {
		case <-ctx.Done():
			return
		case mr, ok := <-pub.MessageReturns():
			if ok {
				pub.log.Warningf("message returned: %+v", mr)
			}
		case <-pub.Pause():
			pub.log.Warning("publisher is unavailable")
		case <-pub.Ready():
			pub.log.Debug("publisher is ready")
			go handleDispatcher(pub.GetDispatcher(ctx, true, opts))
		}
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestFlows(t *testing.T) {
	// Ensure AMQP server is available for testing
	res, err := http.Get("http://localhost:15672/api/auth")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Skip("no AMQP server available for testing")
	}
	_ = res.Body.Close()

	// Main assets
	assert := tdd.New(t)
	server := "amqp://guest:guest@localhost:5672"
	ll := xlog.WithZero(xlog.ZeroOptions{
		PrettyPrint: true,
		ErrorField:  "error",
	})
	st := Topology{}
	assert.Nil(yaml.Unmarshal([]byte(sampleTopology), &st), "decode topology")

	// Return settings array
	getOptions := func(name string, extras ...Option) []Option {
		base := []Option{
			WithName(name),
			WithTopology(st),
			WithLogger(ll.Sub(xlog.Fields{"id": name})),
			WithPrefetch(1, 0),
		}
		base = append(base, extras...)
		return base
	}

	t.Run("Session", func(t *testing.T) {
		// Bare session with no activity
		session, err := open(getName("sample"), server, getOptions("custom-name")...)
		assert.Nil(err, "failed to open session")

		// Monitor session
		go func() {
			for status := range session.status {
				if status {
					ll.Debug("session is ready. start/resume processing")
				} else {
					ll.Debug("session is not ready. stop processing")
				}
			}
			ll.Warning("closing session monitor")
		}()

		// Wait for a bit and close
		<-time.After(1 * time.Second)
		assert.Nil(session.close(), "session close error")
	})

	t.Run("Consumer", func(t *testing.T) {
		// Create consumer
		cc, err := NewConsumer(server, getOptions("consumer-1")...)
		assert.Nil(err, "failed to start consumer")

		// Start consumer processing
		go consumerEvents(cc, 1, SubscribeOptions{Queue: "hello"})

		// Wait a bit and exit
		<-time.After(1 * time.Second)
		assert.Nil(cc.Close(), "consumer close")
	})

	t.Run("Publisher", func(t *testing.T) {
		// Create publisher
		pub, err := NewPublisher(server, getOptions("publisher-1")...)
		assert.Nil(err, "failed to create publisher")

		// Start publisher processing
		ctx, halt := context.WithCancel(context.Background())
		pubOptions := MessageOptions{RoutingKey: "hello"}
		go publisherEvents(ctx, pub, pubOptions)

		// Wait for a bit and close publisher
		<-time.After(1 * time.Second)
		halt()
		assert.Nil(pub.Close(), "close publisher error")
	})

	// Tests based on the RabbitMQ "getting started" tutorials
	// https://www.rabbitmq.com/getstarted.html
	t.Run("Tutorials", func(t *testing.T) {
		// Single publisher, single consumer
		// https://www.rabbitmq.com/tutorials/tutorial-one-go.html
		t.Run("01", func(t *testing.T) {
			// Create consumer
			sub, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")

			// Start consumer processing
			go consumerEvents(sub, 1, SubscribeOptions{Queue: "hello"})

			// Create publisher
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")

			// Start publisher processing
			ctx, halt := context.WithCancel(context.Background())
			pubOptions := MessageOptions{RoutingKey: "hello"}
			go publisherEvents(ctx, pub, pubOptions)

			// Wait for a bit and stop
			<-time.After(5 * time.Second)
			halt()
			<-ctx.Done()
			assert.Nil(pub.Close(), "close publisher-1")
			assert.Nil(sub.Close(), "close consumer-1")
		})

		// Work queue. Messages are delivered to one of the subscribers on
		// a round-robin model.
		// https://www.rabbitmq.com/tutorials/tutorial-two-go.html
		t.Run("02", func(t *testing.T) {
			// Create consumer
			sub, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")

			// Start consumer processing
			// A single consumer instance can handle multiple subscriptions
			// to handle the load.
			go consumerEvents(sub, 2, SubscribeOptions{Queue: "hello"})

			// Create publisher that add messages to the queue
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")

			// Start publisher processing
			ctx, halt := context.WithCancel(context.Background())
			pubOptions := MessageOptions{RoutingKey: "hello"}
			go publisherEvents(ctx, pub, pubOptions)

			// Wait for a bit and stop
			<-time.After(5 * time.Second)
			halt()
			assert.Nil(pub.Close(), "close publisher-1")
			assert.Nil(sub.Close(), "close consumer-1")
		})

		// Publish/Subscribe
		// Messages are delivered to multiple subscribers.
		// https://www.rabbitmq.com/tutorials/tutorial-three-go.html
		t.Run("03", func(t *testing.T) {
			// Create consumer-1 and wait for its connection to be ready
			c1, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")
			<-c1.Ready()

			// Create consumer-2 and wait for its connection to be ready
			c2, err := NewConsumer(server, getOptions("consumer-2")...)
			assert.Nil(err, "failed to start consumer")
			<-c2.Ready()

			// Setup consumers
			assert.Nil(temporaryQueue(c1), "failed to setup consumer-1")
			assert.Nil(temporaryQueue(c2), "failed to setup consumer-2")

			// Create publisher that add message to the queue
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")

			// Start publisher processing
			// Messages are published directly to a fanout exchange, all queues
			// bind to it will receive the messages.
			ctx, halt := context.WithCancel(context.Background())
			pubOptions := MessageOptions{
				Exchange: "sample.notifications",
				TTL:      60,
				Priority: 2,
			}
			go publisherEvents(ctx, pub, pubOptions)

			// Wait for a bit and stop
			<-time.After(5 * time.Second)
			halt()
			assert.Nil(c1.Close(), "close consumer-1")
			assert.Nil(c2.Close(), "close consumer-2")
			assert.Nil(pub.Close(), "close publisher-1")
		})

		// Routing
		// Subscribe to subset of the messages published.
		// https://www.rabbitmq.com/tutorials/tutorial-four-go.html
		t.Run("04", func(t *testing.T) {
			// Create consumer-1 and wait for its connection to be ready
			c1, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")
			<-c1.Ready()

			// Start receiving messages
			deliveries, _, err := c1.Subscribe(SubscribeOptions{
				Queue:   "tasks",
				AutoAck: true,
			})
			assert.Nil(err, "failed to open subscription")
			go func() {
				for msg := range deliveries {
					c1.log.WithField("rk", msg.RoutingKey).Info("message received")
				}
			}()

			// Create publisher that add message to the queue
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")
			<-pub.Ready()

			// Publish messages with different routing keys
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.tasks",
				RoutingKey: "foo", // will be received by the consumer
			})
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.tasks",
				RoutingKey: "bar", // will be received by the consumer
			})
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.tasks",
				RoutingKey: "baz", // won't be received by the consumer
			})

			// Wait for a bit and stop
			<-time.After(1 * time.Second)
			assert.Nil(c1.Close(), "close consumer-1")
			assert.Nil(pub.Close(), "close publisher-1")
		})

		// Topics
		// Subscribe for messages based on a pattern
		// https://www.rabbitmq.com/tutorials/tutorial-five-go.html
		t.Run("05", func(t *testing.T) {
			// Create consumer-1 and wait for its connection to be ready
			c1, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")
			<-c1.Ready()

			// Start receiving messages
			deliveries, _, err := c1.Subscribe(SubscribeOptions{
				Queue:   "by_topic",
				AutoAck: true,
			})
			assert.Nil(err, "failed to open subscription")
			go func() {
				for msg := range deliveries {
					c1.log.WithField("rk", msg.RoutingKey).Info("message received")
				}
			}()

			// Create publisher that add message to the queue
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")
			<-pub.Ready()

			// Publish messages with different routing keys
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.topic",
				RoutingKey: "foo", // won't be received by the consumer
			})
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.topic",
				RoutingKey: "stock.mxn.ob", // won't be received by the consumer
			})
			_ = pub.UnsafePush(randomMessage(), MessageOptions{
				Exchange:   "sample.topic",
				RoutingKey: "stock.nyc.cvx", // will be received by the consumer
			})

			// Wait for a bit and stop
			<-time.After(1 * time.Second)
			assert.Nil(c1.Close(), "close consumer-1")
			assert.Nil(pub.Close(), "close publisher-1")
		})

		// RPC
		// https://www.rabbitmq.com/tutorials/tutorial-six-go.html
		t.Run("06", func(t *testing.T) {
			// Create consumer-1 and wait for its connection to be ready
			c1, err := NewConsumer(server, getOptions("consumer-1", WithRPC())...)
			assert.Nil(err, "failed to start consumer")
			<-c1.Ready()

			// Start receiving messages
			// Consumer is used as a worker instance waiting for
			// RPC requests to be handled.
			deliveries, _, err := c1.Subscribe(SubscribeOptions{
				Queue:   "rpc",
				AutoAck: true,
			})
			assert.Nil(err, "failed to open subscription")
			go func() {
				for msg := range deliveries {
					// Submit RPC response
					response := randomMessage()
					response.CorrelationId = msg.MessageId // MUST be set to properly route response
					assert.Nil(c1.RespondRPC(response, msg.ReplyTo), "respond RPC")
				}
			}()

			// Create publisher
			pub, err := NewPublisher(server, getOptions("publisher-1", WithRPC())...)
			assert.Nil(err, "failed to create publisher")
			<-pub.Ready()

			// Wait a bit for extra setup
			<-time.After(1 * time.Second)

			// Dispatch RPC request
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			response, err := pub.SubmitRPC(ctx, "sample.rpc", randomMessage())
			assert.Nil(err, "submit RPC error")

			// Wait for response or timeout
			select {
			case res := <-response:
				ll.Debugf("response received: %x", res.Body)
			case <-ctx.Done():
				assert.Fail("RPC timeout")
			}
			cancel()

			// Wait for a bit and stop
			<-time.After(2 * time.Second)
			assert.Nil(c1.Close(), "close consumer-1")
			assert.Nil(pub.Close(), "close publisher-1")
		})

		// Headers
		// Route messages using its headers example
		t.Run("07", func(t *testing.T) {
			// Create consumer-1 and wait for its connection to be ready
			c1, err := NewConsumer(server, getOptions("consumer-1")...)
			assert.Nil(err, "failed to start consumer")
			<-c1.Ready()

			// Start receiving messages
			deliveries, _, err := c1.Subscribe(SubscribeOptions{
				Queue:   "by_headers",
				AutoAck: true,
			})
			assert.Nil(err, "failed to open subscription")
			go func() {
				for msg := range deliveries {
					c1.log.WithField("headers", msg.Headers).Info("message received")
				}
			}()

			// Create publisher that add message to the queue
			pub, err := NewPublisher(server, getOptions("publisher-1")...)
			assert.Nil(err, "failed to create publisher")
			<-pub.Ready()

			// Publish messages with different routing keys
			msg := randomMessage()
			msg.Headers = map[string]interface{}{"foo": "bar"}
			_ = pub.UnsafePush(msg, MessageOptions{Exchange: "sample.headers"})             // Will be delivered
			_ = pub.UnsafePush(randomMessage(), MessageOptions{Exchange: "sample.headers"}) // Won't reach queue

			// Wait for a bit and stop
			<-time.After(2 * time.Second)
			assert.Nil(c1.Close(), "close consumer-1")
			assert.Nil(pub.Close(), "close publisher-1")
		})
	})
}
