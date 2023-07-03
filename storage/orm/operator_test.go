package orm

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	tdd "github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type book struct {
	Title  string `json:"title"`
	Author string `json:"author_name"`
	Pages  uint8  `json:"pages"`
}

type accountStatus struct {
	Email         string    `json:"email"`
	LastUpdate    time.Time `json:"last_update"`
	BorrowedBooks []*book   `json:"borrowed_books"`
}

func (acc *accountStatus) addBook(b *book) {
	acc.BorrowedBooks = append(acc.BorrowedBooks, b)
	acc.LastUpdate = time.Now().UTC()
}

type sampleConstraint struct {
	IsCa           bool `json:"is_ca"`
	MaxPathLen     int  `json:"max_path_len"`
	MaxPathLenZero bool `json:"max_path_len_zero"`
}

type sampleStructure struct {
	Usages            []string          `json:"usages"`
	IssuerUrls        []string          `json:"issuer_urls"`
	OcspURL           string            `json:"ocsp_url"`
	CrlURL            string            `json:"crl_url"`
	OcspNoCheck       bool              `json:"ocsp_no_check"`
	Expiry            string            `json:"expiry"`
	AllowedExtensions []string          `json:"allowed_extensions"`
	CaConstraint      *sampleConstraint `json:"ca_constraint"`
}

// Returns a sample complex proto object without `bson` tag annotations.
func notAnnotatedStruct() *sampleStructure {
	return &sampleStructure{
		Usages:            []string{"ca"},
		IssuerUrls:        []string{"https://ca.acme.com"},
		OcspURL:           "https://ca.acme.com/ocsp",
		CrlURL:            "https://ca.acme.com/crl",
		OcspNoCheck:       true,
		Expiry:            "30d",
		AllowedExtensions: []string{uuid.New().String()},
		CaConstraint: &sampleConstraint{
			IsCa:           true,
			MaxPathLen:     2,
			MaxPathLenZero: false,
		},
	}
}

// Returns a sample struct with `bson` annotations.
func annotatedStruct() *book {
	return &book{
		Title:  uuid.New().String(),
		Pages:  uint8(rand.Intn(300) + 50),
		Author: "john doe",
	}
}

func TestOperator(t *testing.T) {
	assert := tdd.New(t)

	// Connection settings
	conf := options.Client()
	conf.ApplyURI("mongodb://localhost:27017/?tls=false")
	conf.SetMinPoolSize(2)
	conf.SetAppName("testing-code")
	conf.SetDirect(true)
	conf.SetReadPreference(readpref.Primary())

	// Get operator
	op, err := NewOperator("testing", conf)
	assert.Nil(err, "new operator")

	// Ensure the MongoDB server is reachable
	if err := op.Ping(); err != nil {
		t.Skip("unavailable MongoDB server:", err.Error())
	}

	t.Run("NotAnnotated", func(t *testing.T) {
		// Get model
		mod := op.Model("protos")

		t.Run("Insert", func(t *testing.T) {
			p1 := notAnnotatedStruct()
			id, err := mod.Insert(p1)
			assert.Nil(err, "create")

			// Find by id
			p2 := &sampleStructure{}
			assert.Nil(mod.FindByID(id, p2), "find")
			assert.Equal(p1, p2, "not equal")
		})

		t.Run("Batch", func(t *testing.T) {
			list := make([]*sampleStructure, 10)
			for i := 0; i < 10; i++ {
				list[i] = notAnnotatedStruct()
			}
			n, err := mod.Batch(list)
			assert.Nil(err, "batch")
			assert.Equal(10, int(n), "bad count")
		})

		t.Run("Find", func(t *testing.T) {
			var target []*sampleStructure
			err := mod.Find(Filter(), &target)
			assert.Nil(err, "find")
			assert.Equal(11, len(target), "no results")
		})

		t.Run("Count", func(t *testing.T) {
			total, err := mod.Estimate()
			assert.Nil(err, "total")
			count, err := mod.Count(Filter())
			assert.Nil(err, "count")
			assert.Equal(total, count, "mismatch")
		})

		t.Run("Distinct", func(t *testing.T) {
			var list []string
			err := mod.Distinct("expiry", Filter(), &list)
			assert.Nil(err, "distinct")
			assert.Equal(1, len(list), "distinct count")
		})

		t.Run("First", func(t *testing.T) {
			filter := Filter()
			filter["crl_url"] = "https://ca.acme.com/crl"
			el := &sampleStructure{}
			assert.Nil(mod.First(filter, el), "find")
		})

		t.Run("Update", func(t *testing.T) {
			filter := Filter()
			filter["crl_url"] = "https://ca.acme.com/crl"
			el := notAnnotatedStruct()
			el.CaConstraint.MaxPathLenZero = true
			el.OcspURL = "https://ca.acme.com/new_ocsp_url"
			assert.Nil(mod.Update(filter, el, true), "update")
		})

		t.Run("UpdateAll", func(t *testing.T) {
			filter := map[string]interface{}{
				"ocsp_no_check": true,
			}
			patch := map[string]interface{}{
				"ocsp_no_check": false,
			}
			res, err := mod.UpdateAll(filter, patch)
			assert.Nil(err, "update all")
			assert.True(res > 0, "update count")
		})

		t.Run("Delete", func(t *testing.T) {
			filter := Filter()
			filter["ca_constraint.is_ca"] = true
			assert.Nil(mod.Delete(filter), "delete")
		})

		t.Run("DeleteAll", func(t *testing.T) {
			filter := Filter()
			filter["ca_constraint.is_ca"] = true
			_, err := mod.DeleteAll(filter)
			assert.Nil(err, "delete all")
		})
	})

	t.Run("Annotated", func(t *testing.T) {
		// Get model
		mod := op.Model("shelf")

		t.Run("Insert", func(t *testing.T) {
			p1 := annotatedStruct()
			id, err := mod.Insert(p1)
			assert.Nil(err, "create")

			// Find by id
			p2 := &book{}
			assert.Nil(mod.FindByID(id, p2), "find")
			assert.Equal(p1, p2, "not equal")
		})

		t.Run("Batch", func(t *testing.T) {
			list := make([]*book, 10)
			for i := 0; i < 10; i++ {
				list[i] = annotatedStruct()
			}
			n, err := mod.Batch(list)
			assert.Nil(err, "batch")
			assert.Equal(10, int(n), "bad count")
		})

		t.Run("Find", func(t *testing.T) {
			var target []*book
			err := mod.Find(Filter(), &target)
			assert.Nil(err, "find")
			assert.True(len(target) > 10, "no results")
		})

		t.Run("Count", func(t *testing.T) {
			total, err := mod.Estimate()
			assert.Nil(err, "total")
			count, err := mod.Count(Filter())
			assert.Nil(err, "count")
			assert.Equal(total, count, "mismatch")
		})

		t.Run("First", func(t *testing.T) {
			filter := Filter()
			filter["author_name"] = "john doe"
			el := &book{}
			assert.Nil(mod.First(filter, el), "find")
		})

		t.Run("Update", func(t *testing.T) {
			filter := Filter()
			filter["author_name"] = "john doe"
			patch := Filter()
			patch["author_name"] = "jane doe"
			assert.Nil(mod.Update(filter, patch, true), "update")
		})

		t.Run("UpdateAll", func(t *testing.T) {
			filter := Filter()
			filter["author_name"] = "john doe"
			patch := Filter()
			patch["author_name"] = "jane doe"
			res, err := mod.UpdateAll(filter, patch)
			assert.Nil(err, "update all")
			assert.True(res > 0, "update count")
		})

		t.Run("Delete", func(t *testing.T) {
			filter := Filter()
			filter["pages"] = 50
			assert.Nil(mod.Delete(filter), "delete")
		})

		t.Run("DeleteAll", func(t *testing.T) {
			filter := Filter()
			filter["author_name"] = "jane doe"
			_, err := mod.DeleteAll(filter)
			assert.Nil(err, "delete all")
		})
	})

	t.Run("Subscribe", func(t *testing.T) {
		// Get model
		mod := op.Model("accounts")

		// Create original record
		account := &accountStatus{
			Email:         "rick@c137.com",
			LastUpdate:    time.Now().UTC(),
			BorrowedBooks: nil,
		}
		id, err := mod.Insert(account)
		assert.Nil(err, "insert")
		oid, _ := ParseID(id)

		// Open subscription for updates on the specific document
		csOpts := options.ChangeStream()
		csOpts.SetFullDocument(options.UpdateLookup)
		sub, err := mod.Subscribe(PipelineUpdateDocument(oid), csOpts)
		if err != nil {
			// Testing server is not a replicaSet or shard
			t.Skip(err)
		}

		go func() {
			defer log.Println("close subscription")
			for {
				select {
				case <-sub.Done():
					return
				case e := <-sub.Event():
					ac := new(accountStatus)
					err := e.Decode(ac)
					assert.Nil(err, "decode change event")
				}
			}
		}()

		// Run updates
		selector := map[string]interface{}{"_id": oid}
		for i := 0; i < 3; i++ {
			account.addBook(annotatedStruct())
			assert.Nil(mod.Update(selector, account, false), "update error")
			<-time.After(1 * time.Second)
		}

		// End test
		rt, err := sub.Close()
		assert.Nil(err, "close stream")
		assert.Nil(mod.Delete(selector), "delete sample record")
		log.Printf("resume token: %s", rt)
	})

	t.Run("Transaction", func(t *testing.T) {
		mod := op.Model("shelf")

		// Set transaction options
		opts := options.Transaction()
		opts.SetReadConcern(readconcern.Snapshot())
		opts.SetWriteConcern(writeconcern.Majority())

		// Start transaction
		err := op.Tx(func(tx *Transaction) error {
			log.Printf("transaction started with id: %s", tx.ID())

			// Adjust the model to use the transaction
			if err := mod.WithTransaction(tx); err != nil {
				return tx.Abort()
			}

			// Run operations
			if _, err = mod.Insert(annotatedStruct()); err != nil {
				return tx.Abort()
			}
			if _, err = mod.Insert(annotatedStruct()); err != nil {
				return tx.Abort()
			}
			el := annotatedStruct()
			el.Author = "jane doe"
			err = mod.Update(map[string]interface{}{"author_name": "john doe"}, el, false)
			if err != nil {
				return tx.Abort()
			}
			err = mod.Delete(map[string]interface{}{"pages": 50})
			if err != nil {
				return tx.Abort()
			}

			// Commit transaction and return the result
			return tx.Commit()
		}, opts)
		assert.Nil(err, "transaction error")
	})

	// Disconnect
	assert.Nil(op.Close(context.Background()), "disconnect")
}

// Dummy operator reference to be used on examples.
var db *Operator

func ExampleNewOperator() {
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

	// Use the operator instance to create models. And the models
	// to manage data on the database.
	// ..

	// Close the connection to the database when no longer needed
	err = db.Close(context.Background())
	if err != nil {
		panic(err)
	}
}

func ExampleOperator_Tx() {
	c1 := db.Model("shelf")
	c2 := db.Model("protos")

	// Set transaction options
	opts := options.Transaction()
	opts.SetReadConcern(readconcern.Snapshot())
	opts.SetWriteConcern(writeconcern.Majority())

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
		if _, err := c2.Insert(notAnnotatedStruct()); err != nil {
			return tx.Abort()
		}

		// Commit transaction and return final result
		return tx.Commit()
	}

	// Execute complex atomic operation
	if err := db.Tx(complexOperation, opts); err != nil {
		panic(err)
	}
}

func ExampleModel_Insert() {
	// Create a model for the 'shelf' collection. The model
	// will encode/decode objects using the `bson` annotations
	// on the structs.
	shelf := db.Model("shelf")

	// Create an object to store on the database
	sample := &book{
		Title:  "Sample book",
		Author: "John Dow",
		Pages:  137,
	}

	// Store the object
	id, err := shelf.Insert(sample)
	if err != nil {
		panic(err)
	}
	fmt.Printf("book stored with id: %s", id)
}

func ExampleModel_Batch() {
	shelf := db.Model("shelf")

	// Get a list (slice) of items to store no the database.
	var list []*book

	// Batch will store all the items on a single operation. The
	// input provided must be a slice.
	n, err := shelf.Batch(list)
	if err != nil {
		panic(err)
	}
	fmt.Printf("documents saved: %d", n)
}

func ExampleModel_FindByID() {
	shelf := db.Model("shelf")

	// ID must be a valid hex-encoded MongoDB ObjectID
	id := "...hex id string..."

	// The result will be automatically decoded to
	// this instance
	record := book{}

	// Get the result from the collection using the model.
	// Notice the result holder is passed by reference
	// (i.e., is a pointer).
	err := shelf.FindByID(id, &record)
	if err != nil {
		panic(err)
	}
}

func ExampleModel_UpdateAll() {
	shelf := db.Model("shelf")

	// Filter and patch are based on the encoded records as
	// they appear on the database.
	filter := map[string]interface{}{
		"author_name": "John Dow",
	}
	patch := map[string]interface{}{
		"author_name": "Jane Dow",
	}

	// UpdateAll will apply the patch to all documents
	// satisfying the filter.
	total, err := shelf.UpdateAll(filter, patch)
	if err != nil {
		panic(err)
	}
	fmt.Printf("documents updated: %d", total)
}

func ExampleModel_DeleteAll() {
	mod := db.Model("authorities")

	// Filter is based on the encoded records as they appear on
	// the database.
	filter := Filter()
	filter["ca_constraint.is_ca"] = false

	// DeleteAll will remove all documents satisfying the filter.
	total, err := mod.DeleteAll(filter)
	if err != nil {
		panic(err)
	}
	fmt.Printf("documents deleted: %d", total)
}

func ExampleModel_Subscribe() {
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
}
