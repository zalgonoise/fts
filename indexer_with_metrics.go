package fts

import (
	"context"
	"errors"
	"time"

	"github.com/zalgonoise/fts/metrics"
)

type Metrics interface {
	IncSearchesTotal()
	IncSearchesFailed()
	ObserveSearchLatency(ctx context.Context, dur time.Duration)

	IncInsertsTotal()
	IncInsertsFailed()
	ObserveInsertLatency(ctx context.Context, dur time.Duration)

	IncDeletesTotal()
	IncDeletesFailed()
	ObserveDeleteLatency(ctx context.Context, dur time.Duration)
}

type metricsIndexer[K SQLType, V SQLType] struct {
	indexer Indexer[K, V]
	metrics Metrics
}

// Search implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Search method, registering counter and latency observation
// metrics about this call.
//
// This call will look for matches for the input value through the indexed terms, returning a collection of matching
// Attribute, which will contain both key and (full) value for that match.
//
// This call returns an error if the underlying SQL query fails, if scanning for the results fails, or an
// ErrNotFoundKeyword error if there are zero results from the query.
func (i metricsIndexer[K, V]) Search(ctx context.Context, searchTerm V) (res []Attribute[K, V], err error) {
	start := time.Now()
	i.metrics.IncSearchesTotal()

	res, err = i.indexer.Search(ctx, searchTerm)
	if err != nil {
		i.metrics.IncSearchesFailed()
	}

	i.metrics.ObserveSearchLatency(ctx, time.Since(start))

	return res, err
}

// Insert implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Insert method, registering counter and latency observation
// metrics about this call.
//
// This call indexes new attributes in the Indexer, via the input Attribute's key and value content.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input. This is especially useful for the initial load sequence.
func (i metricsIndexer[K, V]) Insert(ctx context.Context, attrs ...Attribute[K, V]) error {
	start := time.Now()
	i.metrics.IncInsertsTotal()

	err := i.indexer.Insert(ctx, attrs...)
	if err != nil {
		i.metrics.IncInsertsFailed()
	}

	i.metrics.ObserveInsertLatency(ctx, time.Since(start))

	return err
}

// Delete implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Delete method, registering counter and latency observation
// metrics about this call.
//
// This call removes attributes in the Indexer, which match input K-type keys.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input.
func (i metricsIndexer[K, V]) Delete(ctx context.Context, keys ...K) error {
	start := time.Now()
	i.metrics.IncDeletesTotal()

	err := i.indexer.Delete(ctx, keys...)
	if err != nil {
		i.metrics.IncDeletesFailed()
	}

	i.metrics.ObserveDeleteLatency(ctx, time.Since(start))

	return err
}

// Shutdown implements the Indexer interface.
//
// This implementation checks if the Metrics implementation contains either a Shutdown or a Close method, calling it if
// so, and then it calls the underlying Indexer's Shutdown method.
//
// This call gracefully closes the Indexer.
func (i metricsIndexer[K, V]) Shutdown(ctx context.Context) error {
	var err error

	switch v := i.metrics.(type) {
	case interface {
		Shutdown(ctx context.Context) error
	}:
		err = v.Shutdown(ctx)
	case interface {
		Close() error
	}:
		err = v.Close()
	}

	return errors.Join(i.indexer.Shutdown(ctx), err)
}

// IndexerWithMetrics decorates the input Indexer with a Metrics interface.
//
// If the Indexer is nil, a no-op Indexer is returned. If the input Metrics is nil, a default
// Prometheus metrics handler is created as a safe default, on port 8080. If the input Indexer is already an
// Indexer with Metrics; then its Metrics is replaced with this one (input or default one).
//
// This Indexer will not add any new functionality besides decorating the Indexer with metrics registry.
func IndexerWithMetrics[K SQLType, V SQLType](indexer Indexer[K, V], m Metrics) Indexer[K, V] {
	if indexer == nil {
		return NoOp[K, V]()
	}

	if m == nil {
		var err error
		m, err = metrics.New(8080)
		if err != nil {
			return indexer
		}
	}

	if withMetrics, ok := (indexer).(metricsIndexer[K, V]); ok {
		withMetrics.metrics = m

		return withMetrics
	}

	return metricsIndexer[K, V]{
		indexer: indexer,
		metrics: m,
	}
}
