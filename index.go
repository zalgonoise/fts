package fts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zalgonoise/x/errs"
	_ "modernc.org/sqlite"
)

const (
	errDomain = errs.Domain("fts")

	ErrZero     = errs.Kind("zero")
	ErrNotFound = errs.Kind("not found")

	ErrAttributes = errs.Entity("attributes")
	ErrKeyword    = errs.Entity("keyword")
)

const (
	minAlloc = 64

	insertValueQuery = `
INSERT INTO fulltext_search (id, val) 
	VALUES (?, ?);
`

	searchQuery = `
SELECT id, val FROM fulltext_search(?);
`

	deleteQuery = `
DELETE FROM fulltext_search
	WHERE id MATCH ?;
`
)

var (
	ErrZeroAttributes  = errs.WithDomain(errDomain, ErrZero, ErrAttributes)
	ErrNotFoundKeyword = errs.WithDomain(errDomain, ErrNotFound, ErrKeyword)
)

// Index exposes fast full-text search by leveraging the SQLite FTS5 feature.
//
// ref: https://www.sqlite.org/fts5.html
//
// This implementation, using the modernc.org's pure-Go SQLite driver, allows having a very broadly-typed
// and yet lightweight full-text search implementation, with optional persistence.
//
// Effectively, the Index uses a SQLite database as a cache, where it is storing (indexed) data as key-value pairs,
// allowing callers to find these key-value pairs by using keywords and search expressions against this data set.
//
// The expressions, syntax and example phrases for these queries can be found in section 3. of the reference document
// above; providing means of performing more complex queries over indexed data.
type Index[K SQLType, V SQLType] struct {
	db *sql.DB
}

// Search will look for matches for the input value through the indexed terms, returning a collection of matching
// Attribute, which will contain both key and (full) value for that match.
//
// This call returns an error if the underlying SQL query fails, if scanning for the results fails, or an
// ErrNotFoundKeyword error if there are zero results from the query.
func (i *Index[K, V]) Search(ctx context.Context, searchTerm V) (res []Attribute[K, V], err error) {
	rows, err := i.db.QueryContext(ctx, searchQuery, searchTerm)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	res = make([]Attribute[K, V], 0, minAlloc)

	for rows.Next() {
		attr := new(Attribute[K, V])

		if err = rows.Scan(&attr.Key, &attr.Value); err != nil {
			return nil, err
		}

		res = append(res, *attr)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("%w: %v", ErrNotFoundKeyword, searchTerm)
	}

	return res, nil
}

// Insert indexes new attributes in the Index, via the input Attribute's key and value content.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input. This is especially useful for the initial load sequence.
func (i *Index[K, V]) Insert(ctx context.Context, attrs ...Attribute[K, V]) error {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for idx := range attrs {
		if _, err = tx.ExecContext(ctx, insertValueQuery, attrs[idx].Key, attrs[idx].Value); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return tx.Rollback()
	}

	return nil
}

// Delete removes attributes in the Index, which match input K-type keys.
//
// A database transaction is performed in order to ensure that the query is executed as quickly as possible; in case
// multiple items are provided as input.
func (i *Index[K, V]) Delete(ctx context.Context, keys ...K) error {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for idx := range keys {
		if _, err = tx.ExecContext(ctx, deleteQuery, keys[idx]); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return tx.Rollback()
	}

	return nil
}

// Shutdown gracefully closes the Index SQLite database, by calling its Close method
func (i *Index[K, V]) Shutdown(_ context.Context) error {
	return i.db.Close()
}

// Attribute describes an entry to be added or returned from the Index, supporting types that are compatible
// with the SQLite FTS feature and implementation.
type Attribute[K SQLType, V SQLType] struct {
	Key   K
	Value V
}

// NewIndex creates an Index using the provided URI and set of Attribute.
//
// If the provided URI is an empty string or ":memory:", the SQLite implementation will comply and run in-memory.
// Otherwise, the URI is treated as a database URI and validated as an OS path. The latter option allows persistence
// of the Index.
//
// An error is returned if the database fails when being open, initialized, and loaded with the input Attribute.
func NewIndex[K SQLType, V SQLType](uri string, attrs ...Attribute[K, V]) (*Index[K, V], error) {
	db, err := open(uri)
	if err != nil {
		return nil, err
	}

	if err = initDatabase(db); err != nil {
		return nil, err
	}

	index := &Index[K, V]{
		db: db,
	}

	if len(attrs) > 0 {
		if err = index.Insert(context.Background(), attrs...); err != nil {
			closeErr := index.db.Close()

			return nil, errors.Join(err, closeErr)
		}
	}

	return index, nil
}
