package orm

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Operator instances receive MongoDB client options and manage an
// underlying network connection. An operator can be used to instantiate
// any number of models.
type Operator struct {
	// Database used
	db string

	// MongoDB client instance
	client *mongo.Client

	// MongoDB client options
	opts *options.ClientOptions
}

// NewOperator returns a new ORM operator instance for the specified
// MongoDB database. The operator will create and manage the required
// client connection(s) based on the provided configuration options.
// Remember to call the 'Close' method to free resources when the
// operator is no longer needed.
func NewOperator(db string, opts *options.ClientOptions) (*Operator, error) {
	// Use custom registry by default
	if opts.Registry == nil {
		opts.Registry = bsonRegistry()
	}

	// Validate client options
	var err error
	if err = opts.Validate(); err != nil {
		return nil, err
	}

	// Get client
	op := &Operator{
		opts: opts,
		db:   db,
	}
	op.client, err = mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	return op, nil
}

// Close any existing MongoDB client connection.
func (op *Operator) Close(ctx context.Context) error {
	return op.client.Disconnect(ctx)
}

// Ping performs a reachability test to the MongoDB server used by
// the operator instance. A default timeout of 5 seconds is used.
func (op *Operator) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return op.client.Ping(ctx, readpref.Primary())
}

// Tx can be used to enable causal consistency for a group of operations
// or to execute operations in an ACID transaction.
func (op *Operator) Tx(body TransactionBody, opts *options.TransactionOptions) error {
	return op.client.UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(opts)
		if err != nil {
			return nil
		}
		tx := &Transaction{
			ctx:  sctx,
			done: make(chan struct{}),
		}
		return body(tx)
	})
}

// Model returns a new model instance. The 'name' provided will also
// be used as the model's underlying MongoDB collection.
// The encoding rules for items stored and retrieved using the model are:
// 1. Only exported fields on structs are included
// 2. Default BSON encoder rules
// 3. Use `bson` tags, if available
// 4. Use `json` tags, if available
// https://pkg.go.dev/go.mongodb.org/mongo-driver/bson
func (op *Operator) Model(name string) *Model {
	return &Model{
		name:       name,
		Collection: op.client.Database(op.db).Collection(name),
	}
}
