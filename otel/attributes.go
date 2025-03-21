package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
)

// Attributes provide an easy-to-use mechanism to handle span
// and message metadata.
type Attributes map[string]any

// Set a specific attribute, overrides any previously set value.
func (attrs Attributes) Set(key string, value any) {
	if v, ok := value.(string); ok {
		value = strings.TrimSpace(v)
	}
	if strings.TrimSpace(key) != "" {
		attrs[key] = value
	}
}

// Get a previously set attribute value or nil.
func (attrs Attributes) Get(key string) any {
	v, ok := attrs[key]
	if !ok {
		return nil
	}
	return v
}

// Context returns a context instance with the attributes properly set
// as baggage (or correlation) values.
func (attrs Attributes) Context() context.Context {
	bag, _ := baggage.New(members(attrs)...)
	return baggage.ContextWithBaggage(context.Background(), bag)
}

// Join will add any number of attribute sets into the current instance.
func (attrs Attributes) Join(list ...Attributes) {
	for _, md := range list {
		for k, v := range md {
			attrs.Set(k, v)
		}
	}
}

// Load from key/value list to an attributes instance.
func (attrs Attributes) Load(list []attribute.KeyValue) {
	for _, el := range list {
		if el.Key.Defined() {
			attrs[string(el.Key)] = el.Value.AsInterface()
		}
	}
}

// Expand allows converting from attributes to a key/value list.
func (attrs Attributes) Expand() []attribute.KeyValue {
	return expand(attrs)
}

// Returns a list-member of a baggage-string as defined by the W3C Baggage
// specification.
func members(attrs Attributes) []baggage.Member {
	var members []baggage.Member
	for _, el := range expand(attrs) {
		if m, err := baggage.NewMember(string(el.Key), el.Value.AsString()); err == nil {
			members = append(members, m)
		}
	}
	return members
}

// Expand allows converting from attributes to a key/value list.
func expand(attrs Attributes) []attribute.KeyValue {
	var list []attribute.KeyValue
	for k, v := range attrs {
		if strings.TrimSpace(k) != "" {
			list = append(list, kvAny(k, v))
		}
	}
	return list
}

// Any creates a new key-value pair instance with a passed name and
// automatic type inference. This is slower, and not type-safe.
func kvAny(k string, value any) attribute.KeyValue {
	if value == nil {
		return attribute.String(k, "<nil>")
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return attribute.String(k, stringer.String())
	}

	rv := reflect.ValueOf(value)

	// nolint: forcetypeassert, errcheck
	switch rv.Kind() {
	case reflect.Array:
		rv = rv.Slice(0, rv.Len())
		fallthrough
	case reflect.Slice:
		switch reflect.TypeOf(value).Elem().Kind() {
		case reflect.Bool:
			return attribute.BoolSlice(k, rv.Interface().([]bool))
		case reflect.Int:
			return attribute.IntSlice(k, rv.Interface().([]int))
		case reflect.Int64:
			return attribute.Int64Slice(k, rv.Interface().([]int64))
		case reflect.Float64:
			return attribute.Float64Slice(k, rv.Interface().([]float64))
		case reflect.String:
			return attribute.StringSlice(k, rv.Interface().([]string))
		default:
			return attribute.String(k, "<nil>")
		}
	case reflect.Bool:
		return attribute.Bool(k, rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64(k, rv.Int())
	case reflect.Float64:
		return attribute.Float64(k, rv.Float())
	case reflect.String:
		return attribute.String(k, rv.String())
	default:
		if b, err := json.Marshal(value); b != nil && err == nil {
			return attribute.String(k, string(b))
		}
		return attribute.String(k, fmt.Sprint(value))
	}
}
