package tracing

import (
	"context"
	"log"

	"github.com/linecard/self/internal/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitOtel initializes OpenTelemetry tracing and returns the context, tracer provider, and shutdown function.
func InitOtel() (context.Context, *sdktrace.TracerProvider, func()) {
	if util.OtelConfigPresent() {
		ctx := context.Background()
		client := otlptracegrpc.NewClient()

		exp, err := otlptrace.New(ctx, client)
		if err != nil {
			log.Fatalf("failed to initialize grpc exporter: %v", err)
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp))

		shutdown := func() {
			_ = exp.Shutdown(ctx)
			_ = tp.Shutdown(ctx)
		}

		otel.SetTracerProvider(tp)

		return ctx, tp, shutdown
	}

	noopProvider := &sdktrace.TracerProvider{}
	return context.Background(), noopProvider, func() {}
}
