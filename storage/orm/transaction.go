package orm

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransactionBody serves as a wrapper to group all operations
// confirming a specific transaction.
type TransactionBody func(tx *Transaction) error

// Transaction instances allow to atomically run several operations.
// If an operation fails, the whole transaction is reverted.
type Transaction struct {
	ctx  mongo.SessionContext
	done chan struct{}
}

// Commit the transaction. This method will return an error if the
// transaction is no longer active or has been aborted.
func (tx *Transaction) Commit() error {
	defer close(tx.done)
	return tx.ctx.CommitTransaction(tx.ctx)
}

// Abort the transaction and rollback all changes. This method will
// return an error if transaction is no longer active, has been
// committed or aborted.
func (tx *Transaction) Abort() error {
	defer close(tx.done)
	return tx.ctx.AbortTransaction(tx.ctx)
}

// Done triggers a notification when the transaction is completed.
// Either by abort or commit.
func (tx *Transaction) Done() <-chan struct{} {
	return tx.done
}

// ID returns the session identifier for the transaction.
func (tx *Transaction) ID() bson.Raw {
	return tx.ctx.ID()
}
