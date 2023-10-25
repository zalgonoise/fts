package fts

import "context"

// NoOp returns a no-op Indexer for the given key-value types K and V.
func NoOp[K SQLType, V SQLType]() Indexer[K, V] {
	return noOpIndexer[K, V]{}
}

type noOpIndexer[K SQLType, V SQLType] struct{}

// Search implements the Indexer interface.
//
// This is a no-op call and the returned values are always both nil.
func (i noOpIndexer[K, V]) Search(context.Context, V) ([]Attribute[K, V], error) { return nil, nil }

// Insert implements the Indexer interface.
//
// This is a no-op call and the returned error is always nil.
func (i noOpIndexer[K, V]) Insert(context.Context, ...Attribute[K, V]) error { return nil }

// Delete implements the Indexer interface.
//
// This is a no-op call and the returned error is always nil.
func (i noOpIndexer[K, V]) Delete(context.Context, ...K) error { return nil }

// Shutdown implements the Indexer interface.
//
// This is a no-op call and the returned error is always nil.
func (i noOpIndexer[K, V]) Shutdown(context.Context) error { return nil }
