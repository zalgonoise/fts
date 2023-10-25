package fts

import (
	"context"
	"log/slog"
	"os"
)

type loggedIndexer[K SQLType, V SQLType] struct {
	indexer Indexer[K, V]
	logger  *slog.Logger
}

// Search implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Search method, registering log entries before the
// call and if it raises an error with a Warn-level event.
//
// This call will look for matches for the input value through the indexed terms, returning a collection of matching
// Attribute, which will contain both key and (full) value for that match.
//
// This call returns an error if the underlying SQL query fails, if scanning for the results fails, or an
// ErrNotFoundKeyword error if there are zero results from the query.
func (i loggedIndexer[K, V]) Search(ctx context.Context, searchTerm V) ([]Attribute[K, V], error) {
	i.logger.InfoContext(ctx, "finding matches for search term", slog.Any("search_term", searchTerm))

	res, err := i.indexer.Search(ctx, searchTerm)
	if err != nil {
		i.logger.WarnContext(ctx, "error when finding matches", slog.String("error", err.Error()))
	}

	return res, err
}

// Insert implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Insert method, registering log entries before the
// call and if it raises an error with a Warn-level event.
//
// This call indexes new attributes in the Indexer, via the input Attribute's key and value content.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input. This is especially useful for the initial load sequence.
func (i loggedIndexer[K, V]) Insert(ctx context.Context, attrs ...Attribute[K, V]) error {
	i.logger.InfoContext(ctx, "inserting attributes", slog.Int("num_attributes", len(attrs)))

	if err := i.indexer.Insert(ctx, attrs...); err != nil {
		i.logger.WarnContext(ctx, "failed to insert attributes", slog.String("error", err.Error()))

		return err
	}

	return nil
}

// Delete implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Delete method, registering log entries before the
// call and if it raises an error with a Warn-level event.
//
// This call removes attributes in the Indexer, which match input K-type keys.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input.
func (i loggedIndexer[K, V]) Delete(ctx context.Context, keys ...K) error {
	i.logger.InfoContext(ctx, "deleting keys", slog.Any("keys", keys))

	if err := i.indexer.Delete(ctx, keys...); err != nil {
		i.logger.WarnContext(ctx, "failed to delete indexed items", slog.String("error", err.Error()))

		return err
	}

	return nil
}

// Shutdown implements the Indexer interface.
//
// This implementation calls the underlying Indexer's Shutdown method, registering log entries before the
// call and if it raises an error with a Warn-level event.
//
// This call gracefully closes the Indexer.
func (i loggedIndexer[K, V]) Shutdown(ctx context.Context) error {
	i.logger.InfoContext(ctx, "shutting down Indexer")

	if err := i.indexer.Shutdown(ctx); err != nil {
		i.logger.WarnContext(ctx, "failed to gracefully shut down", slog.String("error", err.Error()))

		return err
	}

	return nil
}

// IndexerWithLogs decorates the input Indexer with a slog.Logger using the input slog.Handler.
//
// If the Indexer is nil, a no-op Indexer is returned. If the input slog.Handler is nil, a default
// text handler is created as a safe default. If the input Indexer is already a logged Indexer; then
// its logger's handler is replaced with this handler (input or default one).
//
// This Indexer will not add any new functionality besides decorating the Indexer with log events.
func IndexerWithLogs[K SQLType, V SQLType](indexer Indexer[K, V], handler slog.Handler) Indexer[K, V] {
	if indexer == nil {
		return NoOp[K, V]()
	}

	if handler == nil {
		handler = slog.NewTextHandler(os.Stderr, nil)
	}

	if withLogs, ok := (indexer).(loggedIndexer[K, V]); ok {
		withLogs.logger = slog.New(handler)

		return withLogs
	}

	return loggedIndexer[K, V]{
		indexer: indexer,
		logger:  slog.New(handler),
	}
}
