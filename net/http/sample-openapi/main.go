package main

import (
	"context"
	"errors"
	"math/rand"
	"sync"

	api "github.com/bryk-io/sample-openapi/petstore"
	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/net/http"
	mwGzip "go.bryk.io/pkg/net/middleware/gzip"
	mwLog "go.bryk.io/pkg/net/middleware/logging"
	mwRecovery "go.bryk.io/pkg/net/middleware/recovery"
)

func main() {
	ll := log.WithCharm(log.CharmOptions{})
	operator := newOperator()
	svc, err := api.NewServer(operator)
	if err != nil {
		panic(err)
	}
	srv, err := http.NewServer(
		http.WithPort(8080),
		http.WithHandler(svc),
		http.WithMiddleware(mwRecovery.Handler()),
		http.WithMiddleware(mwGzip.Handler(5)),
		http.WithMiddleware(mwLog.Handler(ll, nil)),
	)
	if err != nil {
		panic(err)
	}
	ll.Info("server ready")
	if err := srv.Start(); err != nil {
		panic(err)
	}
}

// ! sample service implementation

func newOperator() *operator {
	db := new(db)
	db.pets = make(map[int64]*api.Pet)
	return &operator{db}
}

type db struct {
	pets map[int64]*api.Pet
	mu   sync.Mutex
}

func (db *db) set(pet *api.Pet) {
	db.mu.Lock()
	if !pet.ID.Set {
		pet.ID.SetTo(int64(rand.Intn(999)))
	}
	db.pets[pet.ID.Value] = pet
	db.mu.Unlock()
}

func (db *db) get(id int64) (*api.Pet, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	p, ok := db.pets[id]
	if !ok {
		return nil, errors.New("invalid pet id")
	}
	return p, nil
}

func (db *db) delete(id int64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.pets, id)
}

type operator struct {
	db *db
}

func (o *operator) AddPet(ctx context.Context, req *api.Pet) (*api.Pet, error) {
	o.db.set(req)
	return req, nil
}

func (o *operator) DeletePet(ctx context.Context, params api.DeletePetParams) error {
	o.db.delete(params.PetId)
	return nil
}

func (o *operator) GetPetById(ctx context.Context, params api.GetPetByIdParams) (api.GetPetByIdRes, error) {
	pet, err := o.db.get(params.PetId)
	if err != nil {
		return &api.GetPetByIdNotFound{}, nil
	}
	return pet, nil
}

func (o *operator) UpdatePet(ctx context.Context, params api.UpdatePetParams) error {
	pet, err := o.db.get(params.PetId)
	if err != nil {
		return err
	}
	pet.Status = params.Status
	if val, ok := params.Name.Get(); ok {
		pet.Name = val
	}
	o.db.set(pet)
	return nil
}
