package amqp

import (
	"time"
)

// Topology allows publishers and consumers to specify the expected/required
// state on the message broker used.
type Topology struct {
	// Exchanges provide destinations where messages are sent.
	Exchanges []Exchange `json:"exchanges,omitempty" yaml:",omitempty"`

	// Queues store messages for consumption.
	Queues []Queue `json:"queues,omitempty" yaml:",omitempty"`

	// Bindings connect exchange to queues to route messages.
	Bindings []Binding `json:"bindings,omitempty" yaml:",omitempty"`
}

// Queue store messages that are consumed by applications.
type Queue struct {
	// Unique name for the queue, may be empty in which case a random and
	// unique name will be generated. This can be useful when creating
	// temporary queues.
	Name string `json:"name"`

	// Whether the queue should be restored on server restarts.
	Durable bool `json:"durable"`

	// Whether to automatically delete the queue when the last consumer
	// is closed.
	AutoDelete bool `json:"auto_delete" yaml:"auto_delete"`

	// Exclusive queues are only accessible by the connection that declares
	// them and will be deleted when the connection closes. Channels on other
	// connections will receive an error when attempting to declare, bind, consume,
	// purge or delete a queue with the same name.
	Exclusive bool `json:"exclusive"`

	// Additional arguments.
	// Some commonly used arguments include:
	// - x-message-ttl (milliseconds)
	//   How long a message published to a queue can live before it is discarded
	// - x-expires (milliseconds)
	//   How long a queue can be unused for before it is automatically deleted
	// - x-max-length
	//   How many (ready) messages a queue can contain before it starts to drop
	//   them from its head.
	// - x-max-length-bytes
	//   Total body size for ready messages a queue can contain before it
	//   starts to drop them from its head
	// - x-overflow ("drop-head" "reject-publish" "reject-publish-dlx")
	//   Determines what happens to a message when the maximum length of a queue
	//   is reached
	// - x-dead-letter-exchange
	//   Name of an exchange to which messages will be republished if they are
	//   rejected or expire
	// - x-dead-letter-routing-key
	//   Replacement routing key to use when a message is dead-lettered. If this
	//   is not set, the message's original routing key will be used
	// - x-single-active-consumer
	//   Makes sure only one consumer at a time consumes from the queue and fails
	//   over to another registered consumer in case the active one is canceled
	//   or dies
	// - x-max-priority (between 0 and 9)
	//   Maximum number of priority levels for the queue to support; if not set,
	//   the queue will NOT support message priorities
	// - x-queue-mode ("default" "lazy")
	//   Set the queue into lazy mode, keeping as many messages as possible on
	//   disk to reduce RAM usage; if not set, the queue will keep an in-memory
	//   cache to deliver messages as fast as possible
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:"arguments,omitempty"`
}

// Exchange is an AMQP entity where messages are sent. Exchanges take a message
// and route it into zero or more queues. The routing algorithm used depends on
// the exchange type and rules called bindings.
type Exchange struct {
	// Unique name for the exchange. Names can consist of a non-empty sequence of
	// letters, digits, hyphen, underscore, period, or colon.
	Name string `json:"name"`

	// Exchange type, must be supported by the server.
	// Usual values are:
	// - direct: delivers messages to queues based on the message routing key.
	//   A direct exchange is ideal for the unicast routing of messages.
	//   https://www.rabbitmq.com/tutorials/amqp-concepts#exchange-direct
	// - fanout: routes messages to all of the queues that are bound to it and the
	//   routing key is ignored.
	//   https://www.rabbitmq.com/tutorials/amqp-concepts#exchange-fanout
	// - topic: route messages to one or many queues based on matching between a
	//   message routing key and the pattern that was used to bind a queue to an
	//   exchange.
	//   https://www.rabbitmq.com/tutorials/amqp-concepts#exchange-topic
	// - headers: designed for routing on multiple attributes that are more easily
	//   expressed as message headers than a routing key.
	//   https://www.rabbitmq.com/tutorials/amqp-concepts#exchange-headers
	Kind string `json:"kind"`

	// Durable and Non-Auto-Deleted exchanges will survive server restarts and
	// remain declared when there are no remaining bindings.
	Durable bool `json:"durable"`

	// Non-Durable and Auto-Deleted exchanges will be deleted when there are no
	// remaining bindings and not restored on server restart.
	AutoDelete bool `json:"auto_delete" yaml:"auto_delete"`

	// Exchanges declared as `internal` do not accept published messages.
	// Internal exchanges are useful when you wish to implement inter-exchange topologies
	// that should not be exposed to users of the broker.
	Internal bool `json:"internal"`

	// Additional arguments.
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:",omitempty"`
}

// Binding declarations connects an exchange to a queue so that messages
// published to it will be routed to the queue when the publishing routing key
// matches the binding parameters.
type Binding struct {
	// Name of the exchange to bind.
	Exchange string `json:"exchange" yaml:"exchange"`

	// Name of the queue to bind.
	Queue string `json:"queue" yaml:"queue"`

	// Allows the broker to route the message based on the topology
	// and settings specified.
	// - "direct" exchanges send the message to queues with a direct
	//   match on the routing key.
	// - "topic" exchanges expect the key to be of a particular format
	//   with segments separated by "."; for example "stock.nyc.cvx"
	//   or a pattern value like "stock.nyc.*" or "stock.#"
	// - "fanout" exchanges ignore the routing key and send the messages
	//   to all bind queues.
	// - "headers" exchanges ignore the routing key and use the message
	//   headers instead.
	RoutingKey []string `json:"routing_key" yaml:"routing_key"`

	// Additional arguments.
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:",omitempty"`
}

// QueueOptions provide a helper mechanism to adjust commonly used
// per-queue configuration arguments.
type QueueOptions struct {
	// How long a message published to a queue can live before it is
	// discarded.
	MessageTTL *time.Duration

	// How long a queue can be unused for before it is automatically deleted.
	Expiration *time.Duration

	// How many (ready) messages a queue can contain before it starts to drop
	// them from its head.
	MaxLength uint

	// Total body size for ready messages a queue can contain before it
	// starts to drop them from its head.
	MaxLengthBytes uint

	// Name of an exchange to which messages will be republished if they are
	// rejected or expire.
	DLExchange string

	// Replacement routing key to use when a message is dead-lettered. If this
	// is not set, the message's original routing key will be used.
	DLRoutingKey string

	// Makes sure only one consumer at a time consumes from the queue and fails
	// over to another registered consumer in case the active one is canceled
	// or dies.
	SingleActiveConsumer bool

	// Maximum number of priority levels for the queue to support; if not set,
	// the queue will NOT support message priorities. Valid values are between
	// 0 and 9.
	MaxPriority uint8

	// Keep as many messages as possible on disk to reduce RAM usage; if not
	// set, the queue will keep an in-memory cache to deliver messages as fast
	// as possible.
	LazyMode bool

	// Determines what happens to a message when the maximum length of a queue
	// is reached.
	Overflow OverflowMode
}

// AsArguments returns the options as a properly encoded set of arguments.
func (qo *QueueOptions) AsArguments() map[string]interface{} {
	list := make(map[string]interface{})
	if qo.MessageTTL != nil {
		list["x-message-ttl"] = qo.MessageTTL.Milliseconds()
	}
	if qo.Expiration != nil {
		list["x-expires"] = qo.Expiration.Milliseconds()
	}
	if qo.MaxLength > 0 {
		list["x-max-length"] = qo.MaxLength
	}
	if qo.DLExchange != "" {
		list["x-dead-letter-exchange"] = qo.DLExchange
	}
	if qo.DLRoutingKey != "" {
		list["x-dead-letter-routing-key"] = qo.DLRoutingKey
	}
	if qo.SingleActiveConsumer {
		list["x-single-active-consumer"] = true
	}
	if qo.MaxPriority <= 9 {
		list["x-max-priority"] = qo.MaxPriority
	}
	if qo.LazyMode {
		list["x-queue-mode"] = "lazy"
	}
	if qo.Overflow != "" {
		list["x-overflow"] = qo.Overflow
	}
	return list
}

// OverflowMode adjust the behavior of a queue to handle rejected
// messages.
type OverflowMode string

const (
	// OverflowDropHead set the queue so that the oldest messages in the queue
	// are dropped. This is the default behavior.
	OverflowDropHead OverflowMode = "drop-head"

	// OverflowReject set the queue so that the most recently published messages
	// will be discarded.
	OverflowReject OverflowMode = "reject-publish"

	// OverflowRejectDL set the queue so that the most recently published messages
	// will be discarded and send to the dead letter exchange is provided.
	OverflowRejectDL OverflowMode = "reject-publish-dlx"
)
