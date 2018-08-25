package storage

import (
	"sync"
)

// GoMap is a StorageClient implementation for a simple Go sync.Map.
type GoMap struct {
	m *sync.Map
}

// WasUsed checks if the preimage was used for a previous payment already.
func (m GoMap) WasUsed(preimage string) (bool, error) {
	// We don't need the value. "ok" contains whether a value was found.
	_, ok := m.m.Load(preimage)
	return ok, nil
}

// SetUsed stores the information that a preimage has been used for a payment.
func (m GoMap) SetUsed(preimage string) error {
	m.m.Store(preimage, true)
	return nil
}

// NewGoMap creates a new GoMap.
func NewGoMap() GoMap {
	return GoMap{
		m: &sync.Map{},
	}
}
