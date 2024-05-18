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
func InitOtel() (ctx context.Context, tp *sdktrace.TracerProvider, shutdown func()) {
	ctx = context.Background()
	tp = sdktrace.NewTracerProvider()
	shutdown = func() {}

	if util.OtelConfigPresent() {
		log.Println("initializing OpenTelemetry with OTLP exporter")

		client := otlptracegrpc.NewClient()

		exp, err := otlptrace.New(ctx, client)
		if err != nil {
			log.Fatalf("failed to initialize grpc exporter: %v", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp))

		shutdown = func() {
			_ = exp.Shutdown(ctx)
			_ = tp.Shutdown(ctx)
		}
	}

	otel.SetTracerProvider(tp)

	return ctx, tp, shutdown
}
