package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultTimeout = 15 * time.Second

// GRPCExporter creates a trace.SpanExporter that pushes spans to a tracing backend via gRPC.
//
// This call takes in a URI string to the tracing backend, and returns the trace.SpanExporter and an error if raised.
func GRPCExporter(ctx context.Context, uri string) (sdktrace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, uri,
		// in this scenario, we want to block; normally WithBlock is not recommended.
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	return traceExporter, nil
}
