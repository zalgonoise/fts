package fts

import "database/sql"

// Number is a type constraint that comprises all types that are integer or real numbers.
type Number interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

// Char is a type constraint that comprises all types that represent character sets.
type Char interface {
	string | []byte | []rune
}

// SQLNullable is a type constraint that comprises all supported sql.Null* types, in a full-text search context.
type SQLNullable interface {
	sql.NullBool |
		sql.NullInt16 | sql.NullInt32 | sql.NullInt64 |
		sql.NullFloat64 |
		sql.NullString
}

// SQLType is a type constraint that joins the Number, Char and SQLNullable type constraints.
type SQLType interface {
	Number | Char | SQLNullable
}
