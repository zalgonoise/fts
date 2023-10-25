package fts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
)

const (
	uriFormat = "file:%s?cache=shared"
	inMemory  = ":memory:"

	checkTableExists = `
SELECT EXISTS(SELECT 1 FROM sqlite_master 
	WHERE type='table' 
	AND name='fulltext_search');
`

	createTableQuery = `
CREATE VIRTUAL TABLE fulltext_search 
	USING FTS5(id, val);
`
)

func open(uri string) (*sql.DB, error) {
	switch uri {
	case inMemory:
	case "":
		uri = inMemory
	default:
		if err := validateURI(uri); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", fmt.Sprintf(uriFormat, uri))
	if err != nil {
		return nil, err
	}

	return db, nil
}

func validateURI(uri string) error {
	stat, err := os.Stat(uri)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f, err := os.Create(uri)
			if err != nil {
				return err
			}

			return f.Close()
		}

		return err
	}

	if stat.IsDir() {
		return fmt.Errorf("%s is a directory", uri)
	}

	return nil
}

func initDatabase(db *sql.DB) error {
	ctx := context.Background()
	r, err := db.QueryContext(ctx, checkTableExists)
	if err != nil {
		return err
	}

	defer r.Close()

	for r.Next() {
		var count int
		if err = r.Scan(&count); err != nil {
			return err
		}

		if count == 1 {
			return nil
		}
	}

	_, err = db.ExecContext(ctx, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}
