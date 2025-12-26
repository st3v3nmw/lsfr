package attest

import "sync"

// syncMap is a thread-safe map implementation.
type syncMap[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

// newMap creates a new thread-safe map.
func newMap[K comparable, V any]() *syncMap[K, V] {
	return &syncMap[K, V]{
		m: make(map[K]V),
	}
}

// Set adds or updates a key-value pair in the map.
func (m *syncMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = value
}

// Get retrieves a value by key from the map.
func (m *syncMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.m[key]
	return val, ok
}

// Range iterates over all key-value pairs in the map.
// The iteration stops if the provided function returns false.
func (m *syncMap[K, V]) Range(fn func(K, V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.m {
		if !fn(k, v) {
			break
		}
	}
}
