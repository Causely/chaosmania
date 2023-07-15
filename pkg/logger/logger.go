package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ctxKey struct{}

func NewContext(parent context.Context, z *zap.Logger) context.Context {
	return context.WithValue(parent, ctxKey{}, z)
}

func FromContext(ctx context.Context) *zap.Logger {
	childLogger, _ := ctx.Value(ctxKey{}).(*zap.Logger)

	if traceID := trace.SpanFromContext(ctx).SpanContext().TraceID(); traceID.IsValid() {
		childLogger = childLogger.With(zap.String("trace-id", traceID.String()))
	}

	if spanID := trace.SpanFromContext(ctx).SpanContext().SpanID(); spanID.IsValid() {
		childLogger = childLogger.With(zap.String("span-id", spanID.String()))
	}

	return childLogger
}
