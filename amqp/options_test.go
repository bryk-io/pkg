package amqp

import (
	xlog "go.bryk.io/pkg/log"
	"gopkg.in/yaml.v3"
)

func ExampleWithLogger() {
	// Set the logger instance to use
	WithLogger(xlog.WithZero(xlog.ZeroOptions{
		PrettyPrint: true,
		ErrorField:  "error",
	}))
}

func ExampleWithPrefetch() {
	// Allow 5 in-flight message and a maximum of 512 bytes
	// in server-client buffers.
	WithPrefetch(5, 512)
}

func ExampleWithName() {
	// If not set, publishers are automatically named as "publisher-*"
	// and consumers as "consumer-*"
	WithName("custom-application-name")
}

func ExampleWithTopology() {
	// Allows to load an existing topology declaration, for example
	// from YAML or JSON file, or received from a remote location
	var sampleTopology = `
exchanges:
- name: sample.tasks
  kind: direct
  durable: true
- name: sample.notifications
  kind: fanout
  durable: true
queues:
- name: tasks
  durable: true
  arguments:
    - x-message-ttl: 10000
- name: notifications
  durable: true
bindings:
- exchange: sample.notifications
  queue: notifications
- exchange: sample.tasks
  queue: tasks
  routing_key:
  - foo
  - bar
`
	tp := Topology{}
	_ = yaml.Unmarshal([]byte(sampleTopology), &tp)
	WithTopology(tp)
}
