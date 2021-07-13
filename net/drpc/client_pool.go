package drpc

import (
	"errors"
	"sync"
)

// Pool provides a very simple cache and re-use mechanism for
// DRPC connections used by the client. We don't currently need
// to keep a pointer to active connections. Is the client's
// responsibility to put back connections when no longer active.
type pool struct {
	new    func() (interface{}, error) // constructor method
	free   func(el interface{}) error  // dispose method
	idle   []interface{}               // items available for use
	active int                         // number of items in-use
	limit  int                         // max number of items
	mtx    sync.Mutex                  // concurrent access lock
}

// Get an idle element from the pool. If there's no available
// element a new one will be created. If the pool capacity is
// exceeded an error will be returned.
func (p *pool) Get() (interface{}, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// Verify pool capacity
	if p.limit > 0 && p.active == p.limit {
		return nil, errors.New("pool: max capacity exceeded")
	}

	// No available elements? create a new one
	if len(p.idle) == 0 {
		el, err := p.new()
		if err != nil {
			return nil, err
		}
		p.idle = append(p.idle, el)
	}

	// Retrieve first available element and return it
	el := p.idle[0]
	p.idle = p.idle[1:]
	p.active++
	return el, nil
}

// Put elements back in the pool.
func (p *pool) Put(el ...interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	for _, e := range el {
		p.idle = append(p.idle, e)
		p.active--
	}
}

// Stats return the number of idle and active elements currently
// in the pool.
func (p *pool) Stats() (idle, active int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return len(p.idle), p.active
}

// Drain will asynchronously free all idle elements in the pool and
// report back any errors.
func (p *pool) Drain() <-chan error {
	sink := make(chan error)
	go func() {
		p.mtx.Lock()
		defer p.mtx.Unlock()
		for i, el := range p.idle {
			if err := p.free(el); err != nil {
				sink <- err
			}
			p.idle[i] = nil
		}
		p.idle = []interface{}{}
		close(sink)
	}()
	return sink
}
