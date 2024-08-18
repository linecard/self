package tracing

import (
	"context"

	"github.com/linecard/self/internal/util"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitOtel initializes OpenTelemetry tracing and returns the context, tracer provider, and shutdown function.
func InitOtel() (tp *sdktrace.TracerProvider, shutdown func()) {
	ctx := context.Background()
	tp = sdktrace.NewTracerProvider()
	shutdown = func() {}

	if util.OtelConfigPresent() {
		log.Info().Msg("initializing OpenTelemetry with OTLP exporter")

		client := otlptracegrpc.NewClient()

		exp, err := otlptrace.New(ctx, client)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create OTLP exporter")
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp))

		shutdown = func() {
			_ = tp.ForceFlush(ctx)
			_ = exp.Shutdown(ctx)
			_ = tp.Shutdown(ctx)
		}
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)

	return tp, shutdown
}
