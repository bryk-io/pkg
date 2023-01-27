package orm

import (
	"context"
	"reflect"

	"go.bryk.io/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ns struct {
	Database   string `bson:"db"`
	Collection string `bson:"coll"`
}

// ChangeEvent provides a notification for an operation that modified
// state on a MongoDB instance.
// https://docs.mongodb.com/manual/reference/change-events/
type ChangeEvent struct {
	// Metadata related to the operation. Acts as the resumeToken for
	// the resumeAfter parameter when resuming a change stream.
	ID bson.D `bson:"_id"`

	// The type of operation that occurred. Can be any of the following values:
	// insert, delete, replace, update, drop, rename, dropDatabase, invalidate
	Operation string `bson:"operationType"`

	// The namespace (database and or collection) affected by the event.
	Namespace ns `bson:"ns"`

	// The timestamp from the oplog entry associated with the event.
	ClusterTime primitive.Timestamp `bson:"clusterTime"`

	// A document that contains the _id of the document created or modified
	// by the insert, replace, delete, update operations (i.e. CRUD operations).
	// For sharded collections, also displays the full shard key for the
	// document.
	DocumentKey bson.D `bson:"documentKey"`

	// The document created or modified by the insert, replace, delete,
	// update operations (i.e. CRUD operations). For delete operations, this
	// field is omitted as the document no longer exists.
	//
	// For update operations, this field only appears if you configured the
	// change stream with fullDocument set to updateLookup.
	FullDocument bson.M `bson:"fullDocument"`

	// A document describing the fields that were updated or removed by the
	// update operation.
	UpdateDescription bson.M `bson:"updateDescription,omitempty"`
}

// Decode will load the 'fullDocument' contents of the event instance
// into the provided 'target' element.
func (ce *ChangeEvent) Decode(target interface{}) error {
	// Is there data to decode?
	if ce.FullDocument == nil {
		return errors.New("no 'fullDocument' available")
	}

	// Verify target type
	if err := checkType(target, reflect.Ptr, "pointer"); err != nil {
		return err
	}

	// Decode
	data, err := bson.Marshal(ce.FullDocument)
	if err != nil {
		return err
	}
	return bson.Unmarshal(data, target)
}

// Ensure the event instance has contents.
func (ce *ChangeEvent) empty() bool {
	return ce.ID == nil
}

// Stream provides a simple interface to listen for change events.
// More information:
//
//	https://docs.mongodb.com/manual/changeStreams/
type Stream struct {
	cs    *mongo.ChangeStream
	ctx   context.Context
	halt  context.CancelFunc
	done  chan struct{}
	event chan ChangeEvent
}

// Close the stream and free all its related resources, the
// method returns the stream's resume token (when available)
// and any error produced when closing the stream.
// The resume token can later be used when opening a new stream
// using the 'SetResumeAfter' configuration option.
//
//	conf.SetResumeAfter(rt)
func (s *Stream) Close() (bson.Raw, error) {
	s.halt()
	<-s.done
	close(s.event)
	err := s.cs.Close(context.Background())
	rt := s.cs.ResumeToken()
	return rt, err
}

// Event channel is used to deliver any changes detected for the
// stream parameters.
func (s *Stream) Event() <-chan ChangeEvent {
	return s.event
}

// Done is used to signal when the stream processing has terminated.
func (s *Stream) Done() <-chan struct{} {
	return s.done
}

// Manage change stream cursor.
func (s *Stream) loop() {
	var err error
	for s.cs.Next(s.ctx) {
		var ce ChangeEvent
		if err = s.cs.Decode(&ce); err == nil {
			if !ce.empty() {
				s.event <- ce
			}
		}
	}
	close(s.done)
}
