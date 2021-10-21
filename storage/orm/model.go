package orm

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Model instances serve as a "wrapper" to a MongoDB collection and
// provide an easy-to-use API on top of it to greatly simplify common
// tasks.
type Model struct {
	// MongoDB collection backing the model.
	Collection *mongo.Collection

	// Name of the model. Used also as collection name.
	name string

	// Transaction in-progress
	tx *Transaction

	// State lock
	mu sync.Mutex
}

// WithTransaction sets the active transaction for the model. All CRUD
// operations are bound to the active transaction when executed. The
// active transaction will be removed automatically once it has been
// aborted or committed. This method will return an error if another
// transaction is currently active.
func (m *Model) WithTransaction(tx *Transaction) error {
	m.mu.Lock()
	if m.tx != nil {
		m.mu.Unlock()
		return errors.New("transaction already in progress")
	}
	m.tx = tx
	m.mu.Unlock()
	go func() {
		<-m.tx.Done()
		m.mu.Lock()
		m.tx = nil
		m.mu.Unlock()
	}()
	return nil
}

// Estimate executes a count command and returns an estimate of
// the number of documents in the collection using available metadata.
// This operation trades-off accuracy for speed.
// For more information:
// https://docs.mongodb.com/manual/reference/method/db.collection.estimatedDocumentCount/
func (m *Model) Estimate() (int64, error) {
	return m.Collection.EstimatedDocumentCount(context.Background())
}

// Count returns the number of documents in the collection that satisfy
// the provided filter. An empty filter will count all the documents in
// the collection by performing a full scan. This operation trades-off
// speed for accuracy.
// For more information:
// https://docs.mongodb.com/manual/reference/method/db.collection.countDocuments/
func (m *Model) Count(filter map[string]interface{}) (int64, error) {
	f, err := doc(filter)
	if err != nil {
		return 0, err
	}
	return m.Collection.CountDocuments(context.Background(), f)
}

// Distinct allows to find the unique values for a specified field in the
// collection. If no 'filter' is specified a full scan of the collection
// is performed.
//    var list []string
//    err := mod.Distinct("user_type", Filter(), &list)
//
// For more information:
// https://docs.mongodb.com/manual/reference/command/distinct/
func (m *Model) Distinct(field string, filter map[string]interface{}, result interface{}) error {
	f, err := doc(filter)
	if err != nil {
		return err
	}
	val, err := m.Collection.Distinct(context.Background(), field, f)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(val)
	return json.Unmarshal(data, result)
}

// Insert the item in the model's underlying collection.
func (m *Model) Insert(item interface{}) (string, error) {
	res, err := m.Collection.InsertOne(m.ctx(), item)
	if err != nil {
		return "", err
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("invalid id")
	}
	return id.Hex(), nil
}

// Batch executes an insert command to save multiple documents into the
// collection and return the number of successfully stored items. The
// provided item must be a slice.
func (m *Model) Batch(item interface{}, opts ...*options.InsertManyOptions) (int64, error) {
	// Verify input type
	if err := checkType(item, reflect.Slice, "slice"); err != nil {
		return 0, err
	}

	// Get source slice of items
	src := reflect.ValueOf(item)
	documents := make([]interface{}, src.Len())
	for i := 0; i < len(documents); i++ {
		documents[i] = src.Index(i).Interface()
	}

	// Insert in batch
	res, err := m.Collection.InsertMany(m.ctx(), documents, opts...)
	if err != nil {
		return 0, err
	}
	return int64(len(res.InsertedIDs)), nil
}

// Update will look for the first document that satisfies the 'filter' value
// and apply the 'patch' to it. If no such document currently exists, it will
// be automatically generated if 'upsert' is set to true.
func (m *Model) Update(filter map[string]interface{}, patch interface{}, upsert bool) error {
	// Get filter
	f, err := doc(filter)
	if err != nil {
		return err
	}

	// Run update operation
	_, err = m.Collection.UpdateOne(m.ctx(), f, bson.M{"$set": patch}, &options.UpdateOptions{
		Upsert: &upsert,
	})
	return err
}

// UpdateAll will try to apply the provided 'patch' to all the documents
// satisfying the specified 'filter' and return the number of documents modified
// by the operation.
func (m *Model) UpdateAll(filter map[string]interface{}, patch interface{}) (int64, error) {
	// Get filter
	f, err := doc(filter)
	if err != nil {
		return 0, err
	}

	// Run update operation
	res, err := m.Collection.UpdateMany(m.ctx(), f, bson.M{"$set": patch})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, err
}

// Delete will look for the first document that satisfies the provided 'filter'
// value and permanently remove it. The operation does not fail if no document
// satisfy the filter value.
func (m *Model) Delete(filter map[string]interface{}) error {
	f, err := doc(filter)
	if err != nil {
		return err
	}
	_, err = m.Collection.DeleteOne(m.ctx(), f)
	return err
}

// DeleteAll will remove any document that satisfies the provided 'filter'
// value and return the number of documents deleted by the operation.
func (m *Model) DeleteAll(filter map[string]interface{}) (int64, error) {
	f, err := doc(filter)
	if err != nil {
		return 0, err
	}
	res, err := m.Collection.DeleteMany(m.ctx(), f)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, err
}

// FindByID looks for a given document based on its '_id' field.
// The provided 'id' value must be a MongoDB objectID hex string.
// The returned document is automatically decoded into 'result',
// that must be a pointer to a given struct.
func (m *Model) FindByID(id string, result interface{}) error {
	// Verify target type
	if err := checkType(result, reflect.Ptr, "pointer"); err != nil {
		return err
	}

	// Parse id
	oid, err := ParseID(id)
	if err != nil {
		return err
	}

	// Run query
	sr := m.Collection.FindOne(m.ctx(), bson.M{"_id": oid})
	if err := sr.Err(); err != nil {
		return err
	}

	// Decode result
	return sr.Decode(result)
}

// First looks for the first document in the collection that satisfies the
// specified 'filter'. The returned document is automatically decoded into
// 'result', that must be a pointer to a given struct.
func (m *Model) First(filter map[string]interface{}, result interface{}, opts ...*options.FindOneOptions) error {
	// Verify target type
	if err := checkType(result, reflect.Ptr, "pointer"); err != nil {
		return err
	}

	// Get filter
	f, err := doc(filter)
	if err != nil {
		return err
	}

	// Run query
	sr := m.Collection.FindOne(m.ctx(), f, opts...)
	if err := sr.Err(); err != nil {
		return err
	}

	// Decode result
	return sr.Decode(result)
}

// Find all documents in the collection that satisfy the provided 'filter'.
// The returned documents will be automatically decoded into 'result', that
// must be a pointer to a slice.
func (m *Model) Find(filter map[string]interface{}, result interface{}, opts ...*options.FindOptions) error {
	// Verify target type
	if err := checkType(result, reflect.Ptr, "pointer to a slice"); err != nil {
		return err
	}

	// Get filter
	f, err := doc(filter)
	if err != nil {
		return err
	}

	// Get cursor
	mc, err := m.Collection.Find(m.ctx(), f, opts...)
	if err != nil {
		return err
	}

	// Decode directly into the target slice
	return mc.All(m.ctx(), result)
}

// Subscribe will setup and return an stream instance that can used to
// receive change events based on the parameters provided.
func (m *Model) Subscribe(pipeline mongo.Pipeline, opts *options.ChangeStreamOptions) (*Stream, error) {
	cs, err := m.Collection.Watch(context.Background(), pipeline, opts)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	sub := &Stream{
		cs:    cs,
		ctx:   ctx,
		halt:  cancel,
		done:  make(chan struct{}),
		event: make(chan ChangeEvent),
	}
	go sub.loop()
	return sub, nil
}

func (m *Model) ctx() context.Context {
	if m.tx != nil {
		return m.tx.ctx
	}
	return context.Background()
}
