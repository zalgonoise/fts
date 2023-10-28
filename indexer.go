package fts

import (
	"context"

	"github.com/zalgonoise/cfg"
)

// Indexer describes the actions that a full-text search index should expose. It is declared as an
// interface so that a no-op implementation and observability decorators can be used interchangeably
// through a single constructor.
//
// An Indexer exposes full-text by registering (and tokenizing) key-value pairs of data, that can be looked-up
// for matches with keywords that would be found in the value part of the data. The queries return sets of matching
// key-value pairs.
//
// The Indexer allows creating, reading and deleting entries from the index. This ensures that the index can
// perform an initial load on its own; be updated with more recent data; and also pruning certain keys if needed.
//
// Finally, it also exposes a Shutdown method allowing a graceful shutdown of the search engine.
//
// The underlying index in an Indexer created by this package is an Index type, which leverages the SQLite FTS5 feature
// allowing a fast full-text search engine out-of-the-box, either in-memory or persisted to a file.
type Indexer[K SQLType, V SQLType] interface {

	// Search will look for matches for the input value through the indexed terms, returning a collection of matching
	// Attribute, which will contain both key and (full) value for that match.
	//
	// This call returns an error if the underlying SQL query fails, if scanning for the results fails, or an
	// ErrNotFoundKeyword error if there are zero results from the query.
	Search(ctx context.Context, searchTerm V) (res []Attribute[K, V], err error)

	// Insert indexes new attributes in the Indexer, via the input Attribute's key and value content.
	//
	// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
	// multiple items are provided as input. This is especially useful for the initial load sequence.
	Insert(ctx context.Context, attrs ...Attribute[K, V]) error

	// Delete removes attributes in the Indexer, which match input K-type keys.
	//
	// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
	// multiple items are provided as input.
	Delete(ctx context.Context, keys ...K) error

	// Shutdown gracefully closes the Indexer.
	Shutdown(ctx context.Context) error
}

// New creates an Indexer with the input Attribute and configuration options.
//
// This function allows creating an Index that is intended to be decorated with a logger, metrics and / or tracing.
func New[K SQLType, V SQLType](attributes []Attribute[K, V], opts ...cfg.Option[Config]) (Indexer[K, V], error) {
	config := cfg.New[Config](opts...)

	var (
		indexer Indexer[K, V]
		err     error
	)

	indexer, err = NewIndex[K, V](config.uri, attributes...)
	if err != nil {
		return NoOp[K, V](), err
	}

	if config.logHandler != nil {
		indexer = IndexerWithLogs(indexer, config.logHandler)
	}

	if config.metrics != nil {
		indexer = IndexerWithMetrics(indexer, config.metrics)
	}

	if config.tracer != nil {
		indexer = IndexerWithTrace(indexer, config.tracer)
	}

	return indexer, nil
}
