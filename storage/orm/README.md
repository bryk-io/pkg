# Package `orm`

Simple "Object-Relational Mapping" style library for MongoDB.

## Operator

The main entrypoint to use the package is an 'Operator' instance. An operator
provides a handler for a MongoDB database and manage the required client
connection(s) to it.

```go
// MongoDB client settings
conf := options.Client()
conf.ApplyURI("mongodb://localhost:27017/?tls=false")
conf.SetDirect(true)
conf.SetMinPoolSize(2)
conf.SetReadPreference(readpref.Primary())
conf.SetAppName("super-cool-app")
conf.SetReplicaSet("rs1")

// Get a new operator instance
db, err := NewOperator("testing", conf)
if err != nil {
  panic(err)
}

// Use the operator instance. For example to check if the
// MongoDB server is reachable
if err = db.Ping(); err != nil {
  panic(err)
}

// Close the connection to the database when no longer needed
if err = db.Close(context.Background()); err != nil {
  panic(err)
}
```

## Model

The most common use of an operator instance is to create models. A model
instance serves as a "wrapper" to a MongoDB collection and provides an
easy-to-use API on top of it to greatly simplify common operations. The
most common usage of a model instance is to provide CRUD operations without
the need to use intermediary data structures.

The encoding/decoding rules used when storing or retrieving data using a
model, are the following.

1. Only exported fields on structs are included
2. Default BSON encoder rules are applied
3. If available, `bson` tags are honored
4. If available, `json` tags are honored

This allows to easily store complex types without the need to modify the code
to include `bson` tags, for example, when using Protobuf Messages.

More information: <https://pkg.go.dev/go.mongodb.org/mongo-driver/bson>

## Transactions

In MongoDB, an operation on a single document is atomic. Because you can
use embedded documents and arrays to capture relationships between data in
a single document structure instead of normalizing across multiple documents
and collections, this single-document atomicity obviates the need for
multi-document transactions for many practical use cases.

For situations that require atomicity of reads and writes to multiple documents
(in a single or multiple collections), MongoDB supports multi-document
transactions.

Using this package a transaction is executed using the `Tx` method on an
operator instance. The method takes a `TransactionBody` function that
atomically binds all operations performed inside of it, and returns the final
result of committing or aborting the transaction. The active transaction must
be set on any models used inside the transaction body.

```go
c1 := db.Model("shelf")
c2 := db.Model("protos")

// Complex multi-collection operation
complexOperation := func(tx *Transaction) error {
  // Set the active transaction on all models used
  if err := c1.WithTransaction(tx); err != nil {
    return tx.Abort()
  }
  if err := c2.WithTransaction(tx); err != nil {
    return tx.Abort()
  }

  // Run tasks
  if _, err := c1.Insert(annotatedStruct()); err != nil {
    return tx.Abort()
  }
  if_, err := c2.Insert(notAnnotatedStruct()); err != nil {
    return tx.Abort()
  }

  // Commit transaction and return final result
  return tx.Commit()
}

// Execute transaction
if err := db.Tx(complexOperation, options.Transaction()); err != nil {
  panic(err)
}
```

More information: <https://docs.mongodb.com/manual/core/transactions/>

## Change Streams

Change streams allow applications to access real-time data changes without
the complexity and risk of tailing the oplog. Applications can use change
streams to subscribe to relevant data changes and immediately react to them.

Change streams are provided at the collection level by using the `Subscribe`
method on a model instance.

```go
shelf := db.Model("shelf")

// Open a stream to detect all operations performed on the
// 'shelf' collection
sub, err := shelf.Subscribe(PipelineCollection(), options.ChangeStream())
if err != nil {
  panic(err)
}

// Handle subscription events
go func() {
  for {
    select {
    case <-sub.Done():
      return
    case e := <-sub.Event():
      fmt.Printf("event: %+v\n", e)
    }
  }
}()

// When no longer needed, close the subscription
rt, err := sub.Close()
if err != nil {
  panic(err)
}
fmt.Printf("you can resume the subscription with: %v\n", rt)
```

More information: <https://docs.mongodb.com/manual/changeStreams/>
