package fts

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tracedIndexer[K SQLType, V SQLType] struct {
	indexer Indexer[K, V]
	tracer  trace.Tracer
}

// Search implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Search method, registering spans that last for this call's
// lifetime.
//
// This call will look for matches for the input value through the indexed terms, returning a collection of matching
// Attribute, which will contain both key and (full) value for that match.
//
// This call returns an error if the underlying SQL query fails, if scanning for the results fails, or an
// ErrNotFoundKeyword error if there are zero results from the query.
func (i tracedIndexer[K, V]) Search(ctx context.Context, searchTerm V) ([]Attribute[K, V], error) {
	ctx, span := i.tracer.Start(ctx, "search",
		trace.WithAttributes(attribute.String("search_term", fmt.Sprintf("%v", searchTerm))),
	)

	defer span.End()

	res, err := i.indexer.Search(ctx, searchTerm)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		return res, err
	}

	span.SetAttributes(attribute.Int("num_results", len(res)))

	return res, err
}

// Insert implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Insert method, registering spans that last for this call's
// lifetime.
//
// This call indexes new attributes in the Indexer, via the input Attribute's key and value content.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input. This is especially useful for the initial load sequence.
func (i tracedIndexer[K, V]) Insert(ctx context.Context, attrs ...Attribute[K, V]) error {
	ctx, span := i.tracer.Start(ctx, "insert",
		trace.WithAttributes(attribute.Int("num_attributes", len(attrs))),
	)

	defer span.End()

	err := i.indexer.Insert(ctx, attrs...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	return err
}

// Delete implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Delete method, registering spans that last for this call's
// lifetime.
//
// This call removes attributes in the Indexer, which match input K-type keys.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input.
func (i tracedIndexer[K, V]) Delete(ctx context.Context, keys ...K) error {
	ctx, span := i.tracer.Start(ctx, "delete",
		trace.WithAttributes(attribute.Int("num_keys", len(keys))),
	)

	defer span.End()

	err := i.indexer.Delete(ctx, keys...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	return err
}

// Shutdown implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Shutdown method.
//
// This call gracefully closes the Indexer.
func (i tracedIndexer[K, V]) Shutdown(ctx context.Context) error {
	return i.indexer.Shutdown(ctx)
}

// IndexerWithTrace decorates the input Indexer with a trace.Tracer interface.
//
// If the Indexer is nil, a no-op Indexer is returned. If the input Metrics is nil, a default
// Prometheus metrics handler is created as a safe default. If the input Indexer is already an Indexer with Metrics;
// then its Metrics is replaced with this one (input or default one).
//
// This Indexer will not add any new functionality besides decorating the Indexer with metrics registry.
func IndexerWithTrace[K SQLType, V SQLType](indexer Indexer[K, V], tracer trace.Tracer) Indexer[K, V] {
	if indexer == nil {
		return NoOp[K, V]()
	}

	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("indexer")
	}

	if withLogs, ok := (indexer).(tracedIndexer[K, V]); ok {
		withLogs.tracer = tracer

		return withLogs
	}

	return tracedIndexer[K, V]{
		indexer: indexer,
		tracer:  tracer,
	}
}
