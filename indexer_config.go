package fts

import (
	"log/slog"

	"github.com/zalgonoise/x/cfg"
	"go.opentelemetry.io/otel/trace"
)

// Config defines optional settings in an Indexer
type Config struct {
	uri string

	logHandler slog.Handler
	metrics    Metrics
	tracer     trace.Tracer
}

// WithURI sets a path URI when connecting to the SQLite database, as a means to persist the database in the filesystem.
//
// This option is not mandatory and thus set in the configuration if desired.
func WithURI(uri string) cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.uri = uri

		return config
	})
}

// WithLogger decorates the Indexer with the input slog.Logger.
func WithLogger(logger *slog.Logger) cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.logHandler = logger.Handler()

		return config
	})
}

// WithLogHandler decorates the Indexer with a slog.Logger, using the input slog.Handler.
func WithLogHandler(handler slog.Handler) cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.logHandler = handler

		return config
	})
}

// WithMetrics decorates the Index with the input Metrics instance.
func WithMetrics(metrics Metrics) cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.metrics = metrics

		return config
	})
}

// WithTrace decorates the Index with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.tracer = tracer

		return config
	})
}
