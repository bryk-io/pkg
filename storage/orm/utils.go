package orm

import (
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ParseID returns a MongoDB objectID instance from its hex-encoded
// representation.
func ParseID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

// Filter provides a simple shortcut for a commonly used filter value
// when no fields are specified.
func Filter() map[string]interface{} {
	return map[string]interface{}{}
}

// Returns a properly encoded BSON document.
func doc(filter map[string]interface{}) (bson.D, error) {
	data, err := bson.Marshal(filter)
	if err != nil {
		return nil, err
	}
	f := bson.D{}
	if err = bson.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f, nil
}

// Verify the provided element is of the expected kind.
func checkType(el interface{}, expected reflect.Kind, desc string) error {
	rv := reflect.ValueOf(el)
	if rv.Kind() != expected {
		return fmt.Errorf("target must be a %s, but was a %s", desc, rv.Kind())
	}
	return nil
}
