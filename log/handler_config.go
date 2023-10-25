package log

import (
	"log/slog"

	"github.com/zalgonoise/x/cfg"
)

// Config defines optional configuration settings for a SpanContextHandler.
type Config struct {
	withSpanID bool
	handler    slog.Handler
}

// WithSpanID enables adding an attribute with the span ID data; as an optional setting.
func WithSpanID() cfg.Option[Config] {
	return cfg.Register[Config](func(config Config) Config {
		config.withSpanID = true

		return config
	})
}

// WithHandler uses the input slog.Handler as a base when creating the SpanContextHandler.
func WithHandler(handler slog.Handler) cfg.Option[Config] {
	if handler == nil {
		handler = defaultHandler()
	}

	return cfg.Register[Config](func(config Config) Config {
		config.handler = handler

		return config
	})
}

// WithLogger uses the input slog.Logger's slog.Handler as a base when creating the SpanContextHandler.
func WithLogger(logger *slog.Logger) cfg.Option[Config] {
	if logger == nil {
		return WithHandler(defaultHandler())
	}

	return cfg.Register[Config](func(config Config) Config {
		config.handler = logger.Handler()

		return config
	})
}
