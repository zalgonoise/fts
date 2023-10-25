## FTS

### _a SQLite-based full-text search engine_

_________

### Concept

FTS is a Go library that facilitates fast full-text search with an index that can live in-memory or persisted to a file 
in the OS.

The index is backed by a (pure-Go-driver) SQLite database, leveraging its 
[FTS5 feature](https://www.sqlite.org/fts5.html) to deliver the fastest, simplest, yet modular full-text search engine
that can live completely in-memory.

This strategy makes FTS an ideal solution for small, concise key-value datasets that perform queries on the client side
directly -- and even on bigger datasets where it promises an incredibly fast indexing speed. With it, the library tries 
to provide:
- A simple and clean API for querying, inserting and deleting attributes to a full-text search index, as key-value pairs
- Support for multiple data types, both for the keys and the values, leveraging Generics in Go and ensuring
statically-typed attributes.
- Support for complex search terms, as noted in the section 3. of the 
[FTS5 documentation](https://www.sqlite.org/fts5.html).
- Blazing-fast indexing when comparing to other solutions -- all inserts are done in a single database transaction.
- No sugar: plain SQL, leveraging a SQLite full-text search feature, no processing or manipulation.


________


### Motivation

While exploring the [Steam HTTP API](http://github.com/zalgonoise/x/tree/master/steam), I've decided to create an 
alerting system (with webhooks) for when a certain product was on sale, with a desired discount (-% off). 

In itself, that isn't much despite my complaints on handling the HTTP responses from Steam; however I was having an 
issue: since the Steam API only refers to products by their `app_id`, I'd still need to figure out what that was 
from a product's name. And it doesn't make much sense to _just head over to their website and getting it from there_. No.
That was not the point of a Steam CLI app.

Fortunately, they expose their entire listing through a dedicated endpoint, so you can have all the `app_id` values 
alongside the product name. But this (as of today) consists of 177,590 entries. It's not too big, but it's not small 
either. Also, I wanted this app as something an end-user could run (as a client), and not have to communicate to a 
server to use full-text search. Elastic is crossed out, basically.

Dabbling with a couple of pure-Go solutions, I wasn't happy at all with the indexing times. I tried 
[Bleve](https://github.com/blevesearch/bleve) for example, since it was the Go go-to (hah!) library for full-text search
and provided amazing features; but it was taking over 2 minutes just to index the app IDs.

I kept looking -- suddenly SQLite pops up as an option. _FTS5_ they called it. I didn't even know SQLite supported 
full-text search. But I love SQLite since day one and personally try to use it where I need **extremely fast** SQL in a
project. Please never underestimate in-memory SQLite. I was sold. And I didn't regret it.

At the end, I can get my search results in under two seconds (wait that is a lot!) -- but under two seconds including:
1. Reading a JSON document with the app ID and name entries, as previously downloaded separately (as the HTTP response body content).
2. Creating an in-memory SQLite instance and initializing it with the FTS5 virtual table.
3. Inserting all 177k items.
4. Performing a query using a certain search term as provided by the caller

```shell
$ time go run github.com/zalgonoise/x/steam/cmd/steam search -name 'assetto corsa' 
# (...results)
go run github.com/zalgonoise/x/steam/cmd/steam search -name 'assetto corsa'  1.81s user 0.44s system 102% cpu 2.197 total
```

But you can also persist this SQLite database in the filesystem. What if you point to a local instance of an index?

```shell 
$ time go run github.com/zalgonoise/x/steam/cmd/steam search -name 'assetto corsa' -db '/tmp/steam.db' 
# (...results)
go run github.com/zalgonoise/x/steam/cmd/steam search -name 'assetto corsa' -db '/tmp/steam.db' 0.57s user 0.40s system 142% cpu 0.681 total
```

That's about 600ms. It's still no ElasticSearch! Or Postgres if you configure it right! But it's something usable, and 
it's a tool to consider up to a certain size and complexity, and to consider having on the client instead of on the 
server! It was good enough for my use-case and I took it. Are there better ways of doing this? For sure. But I used SQLite. :)

______

### Usage

> FTS is served as a Go library. You need to import it in your project.

#### Getting FTS 

You can get the library as a Go module, by importing it and running `go mod tidy` on the top-level directory of your 
module, or where `go.mod` is placed.

Then, you're able to initialize it with or without persistence, with or without a set of attributes used in an initial load.
Also, the `Indexer` type also supports options to add observability to your full-text search engine, adding
logging, metrics and tracing as optional configuration.

#### Creating an index

You can create a full-text search index from its concrete-type constructor [`fts.NewIndex()`](./index.go#L141),
or its interface constructor [`fts.New()`](./indexer.go#L52); however, only the latter allows decorating the index with
a logger, metrics and / or tracing in one-go. Regardless, when successful, both are an 
[`*fts.Index[K fts.SQLType, V fts.SQLType]`](./index.go#L40) type.

##### Options

If you choose to create an `Indexer`, you're free to add some configuration options, as described below:

|                    Function                     |                                 Input type                                 |                                                  Description                                                  |
|:-----------------------------------------------:|:--------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------:|
|    [`fts.WithURI`](./indexer_config.go#L22)     |                                  `string`                                  | Sets a path URI when connecting to the SQLite database, as a means to persist the database in the filesystem. |
|   [`fts.WithLogger`](./indexer_config.go#L30)   |            [`*slog.Logger`](https://pkg.go.dev/log/slog#Logger)            |                               Decorates the Indexer with the input slog.Logger.                               |
| [`fts.WithLogHandler`](./indexer_config.go#L40) |           [`slog.Handler`](https://pkg.go.dev/log/slog#Handler)            |                    Decorates the Indexer with a slog.Logger, using the input slog.Handler.                    |
|  [`fts.WithMetrics`](./indexer_config.go#L49)   |               [`fts.Metrics`](./indexer_with_metrics.go#L11)               |                            Decorates the Indexer with the input Metrics instance.                             |
|   [`fts.WithTrace`](./indexer_config.go#L58)    | [`trace.Tracer`](https://pkg.go.dev/go.opentelemetry.io/otel/trace#Tracer) |                              Decorates the Indexer with the input trace.Tracer.                               |

Below is an example where an in-memory index with a logger is created with some attributes, and is also searched on:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/zalgonoise/fts"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	
	// prepare some attributes to index
	attrs := []fts.Attribute[int64, string]{
		{Key: 1, Value: "some entry"},
		{Key: 2, Value: "another entry"},
		{Key: 3, Value: "gold entry"},
		{Key: 4, Value: "different entry"},
		// ...
	}

	// create an in-memory index with a logger, using the created attributes
	indexer, err := fts.New(attrs, fts.WithLogger(logger))
	if err != nil {
		logger.ErrorContext(ctx, "failed to create indexer", slog.String("error", err.Error()))
		
		os.Exit(1)
	}
	
	// search for gold
	results, err := indexer.Search(ctx, "gold")
	if err != nil {
		logger.ErrorContext(ctx, "couldn't get results", slog.String("error", err.Error()))

		os.Exit(1)
	}
	
	// find gold with key 3 and value "gold entry"
	logger.InfoContext(ctx, "found matches", slog.Int("num_results", len(results)))
	for i := range results {
		logger.InfoContext(ctx, "match entry", 
			slog.Int("index", i),
			slog.Int64("key", results[i].Key),
			slog.String("value", results[i].Value),
        )
    }
}
```

But similarly, you can even create an empty index and work with it within your application:

```go
package index

import (
    "context"
    
    "github.com/zalgonoise/fts"
)

type MyIndex struct {
    index *fts.Index[int, string]
    // ...
}

func New(uri string) (*MyIndex, error) {
    index, err := fts.NewIndex[int, string](uri)
    if err != nil {
      return nil, err
    }
  		
    return &MyIndex{
      index: index,
    }, nil
}

func (i *MyIndex) Search(ctx context.Context, searchTerm string) ([]int, error) {
    // ...
}


func (i *MyIndex) Insert(ctx context.Context, attrs ...fts.Attribute[int, string]) error {
    // ...
}
```

#### Using the index

The `Index` type and `Indexer` interface offer a simple CRD set of operations and a graceful shutdown method:

```go
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
```

These methods can be called any time within the lifetime of an Index (before Shutdown is called or the application 
hypothetically crashes), meaning that callers are not limited to writing once and querying forever -- they can safely 
add new attributes to the index and remove attributes by their keys, too.

#### Performing complex queries

Complex queries with matcher expressions and globs are also supported, as noted in the SQLite FTS5 feature specification, 
which can be found in the section 3. of the [FTS5 documentation](https://www.sqlite.org/fts5.html).

This is by far the best resource to use reference for your complex search terms. The (value) data types that support 
these terms are character data types like `string`, `[]byte` and `[]rune`.  


### Disclaimer 

This is **not** considered to be an enterprise-grade solution! While you're free to use this library as you please, 
it is highly recommended to explore other, more performant options. Many of them have drawbacks just like this one, 
many of them are usable via a server only, unlike this one. Please research and explore your options thoroughly!