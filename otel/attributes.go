package otel

import (
	"context"
	"strings"

	xlog "go.bryk.io/pkg/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
)

// Attributes provide an easy-to-use mechanism to handle span
// and message metadata.
type Attributes xlog.Fields

// Set a specific attribute, overrides any previously set value.
func (attrs Attributes) Set(key string, value interface{}) {
	if v, ok := value.(string); ok {
		value = strings.TrimSpace(v)
	}
	if strings.TrimSpace(key) != "" {
		attrs[key] = value
	}
}

// Get a previously set attribute value or nil.
func (attrs Attributes) Get(key string) interface{} {
	v, ok := attrs[key]
	if !ok {
		return nil
	}
	return v
}

// Expand allows converting from attributes to a key/value list.
func (attrs Attributes) Expand() []attribute.KeyValue {
	var list []attribute.KeyValue
	for k, v := range attrs {
		if strings.TrimSpace(k) != "" {
			list = append(list, kvAny(k, v))
		}
	}
	return list
}

// Context returns a context instance with the attributes properly set
// as baggage (or correlation) values.
func (attrs Attributes) Context() context.Context {
	bag, _ := baggage.New(attrs.members()...)
	return baggage.ContextWithBaggage(context.Background(), bag)
}

// Join will add any number of attribute sets into current instance.
func (attrs Attributes) Join(list ...Attributes) {
	for _, md := range list {
		for k, v := range md {
			attrs.Set(k, v)
		}
	}
}

// Convert from key/value list to an attributes instance.
func (attrs Attributes) load(list []attribute.KeyValue) {
	for _, el := range list {
		if el.Key.Defined() {
			attrs[string(el.Key)] = el.Value.AsInterface()
		}
	}
}

// Load attribute values from a source string. Entries are defined as k=v pairs
// separated by ",". Parsed values are registered as strings.
func (attrs Attributes) parse(src string) {
	for _, s := range strings.Split(src, ",") {
		el := strings.Split(s, "=")
		if len(el) == 2 {
			attrs.Set(el[0], el[1])
		}
	}
}

// Returns a list-member of a baggage-string as defined by the W3C Baggage
// specification.
func (attrs Attributes) members() []baggage.Member {
	var members []baggage.Member
	for _, el := range attrs.Expand() {
		if m, err := baggage.NewMember(string(el.Key), el.Value.AsString()); err == nil {
			members = append(members, m)
		}
	}
	return members
}

// Join any number of attribute sets into a single collection.
// Duplicated values are override int the order in which the sets
// containing those values are presented to Join.
func join(list ...Attributes) Attributes {
	out := Attributes{}
	for _, md := range list {
		for k, v := range md {
			if strings.TrimSpace(k) != "" {
				out[k] = v
			}
		}
	}
	return out
}
