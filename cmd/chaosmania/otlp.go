package main

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.uber.org/zap"
)

func initOTLP(logger *zap.Logger) func() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(os.Getenv("DEPLOYMENT_NAME")),
			semconv.ServiceNamespaceKey.String(os.Getenv("NAMESPACE")),
			semconv.DeploymentEnvironmentKey.String(os.Getenv("DOMAIN")),
		),
		resource.WithProcess(),
	)

	if err != nil {
		logger.Error("failed to create resource", zap.Error(err))
		return func() {}
	}

	// Set up a trace exporter using OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_INSECURE, OTEL_EXPORTER_OTLP_HEADERS
	traceExporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		logger.Error("failed to create tracer", zap.Error(err))
		return func() {}
	}

	// Register the trace exporter with a TracerProvider,
	// using a batch span processor to aggregate spans before export.
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		err := tracerProvider.Shutdown(ctx)
		if err != nil {
			logger.Warn("failed to shutdown", zap.Error(err))
		}
	}
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func InitOTLPProvider(logger *zap.Logger) func() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if endpoint != "" {
		return initOTLP(logger)
	} else {
		return func() {}
	}
}
