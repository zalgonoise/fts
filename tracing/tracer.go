package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

const ServiceName = "full-text-search"

// Tracer returns a trace.Tracer for this (full-text search) service.
func Tracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(ServiceName)
}

type ShutdownFunc func(err context.Context) error

// Init registers a new tracer for this service, that exports its spans to the input trace.SpanExporter.
//
// This call returns the TracerProvider's shutdown function, and an error if raised.
func Init(ctx context.Context, traceExporter sdktrace.SpanExporter) (ShutdownFunc, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(ServiceName)),
	)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}
