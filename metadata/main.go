package metadata

import "sync"

// Map is just an alias for the simple and not-thread-safe `map`.
type Map = map[string]interface{}

// MD provides a basic dataset that is safe to use concurrently.
type MD struct {
	data map[string]interface{}
	mu   *sync.RWMutex
}

// New returns an empty metadata set.
func New() MD {
	return MD{
		data: make(map[string]interface{}),
		mu:   new(sync.RWMutex),
	}
}

// FromMap creates a new metadata set and populates with the
// provided `src` data.
func FromMap(src map[string]interface{}) MD {
	md := New()
	md.Load(src)
	return md
}

// Copy the source metadata instance's contents into a new one.
func (m MD) Copy() MD {
	cp := New()
	cp.Load(m.Values())
	return cp
}

// Get the value of a single data entry, return nil if no value is set.
func (m MD) Get(key string) interface{} {
	m.mu.RLock()
	v, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	return v
}

// Set a single data entry, override any value previously set for the
// same key.
func (m MD) Set(key string, value interface{}) {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
}

// Delete "key"(s) from the fields set if exists, if it doesn't this is
// simply a no-op.
func (m MD) Delete(key ...string) {
	m.mu.Lock()
	for _, k := range key {
		delete(m.data, k)
	}
	m.mu.Unlock()
}

// Load the provided list of values; overriding any previously set entries.
func (m MD) Load(src map[string]interface{}) {
	m.mu.Lock()
	for k, v := range src {
		m.data[k] = v
	}
	m.mu.Unlock()
}

// Values returns all values currently registered in the fields handler.
func (m MD) Values() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

// IsEmpty returns `true` if there are currently no values set on the
// metadata instance.
func (m MD) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data) == 0
}

// Clear (i.e. remove) all values currently set.
func (m MD) Clear() {
	m.mu.Lock()
	for k := range m.data {
		delete(m.data, k)
	}
	m.mu.Unlock()
}

// Join all values set in `other` into the current instance.
func (m MD) Join(other ...MD) {
	for _, b := range other {
		for k, v := range b.data {
			m.Set(k, v)
		}
	}
}
