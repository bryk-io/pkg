package amqp

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
)

func ExampleTopology() {
	// To simplify storage and sharing. The topology for an application
	// can be easily managed either in YAML or JSON format.
	var inYAML = `
exchanges:
- name: sample.tasks
  kind: direct
  durable: true
- name: sample.notifications
  kind: fanout
  durable: true
- name: sample.topic
  kind: topic
queues:
- name: hello
  durable: true
  auto_delete: false
  exclusive: false
- name: tasks
  durable: true
  auto_delete: false
  exclusive: false
- name: notifications
  durable: true
  auto_delete: false
  exclusive: false
- name: by_topic
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
`
	tp := Topology{}
	err := yaml.Unmarshal([]byte(inYAML), &tp)
	if err != nil {
		panic(err)
	}
}

func ExampleQueueOptions_AsArguments() {
	ttl, _ := time.ParseDuration("15s")
	exp, _ := time.ParseDuration("1h")
	opts := QueueOptions{
		MessageTTL:           &ttl,
		Expiration:           &exp,
		MaxLength:            500,
		MaxLengthBytes:       1024 * 100, // 100MB
		DLExchange:           "sample.dead",
		SingleActiveConsumer: true,
		MaxPriority:          4,
		LazyMode:             true,
		Overflow:             OverflowRejectDL,
	}
	fmt.Printf("%+v", opts.AsArguments())
}
