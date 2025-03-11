package trace

import (
    "context"
    "testing"

    "go.opentelemetry.io/otel/codes"
)

// TestNewNoopTracerProvider tests that the no-op tracer provider and tracer behave as expected.
func TestNewNoopTracerProvider(t *testing.T) {
    provider := NewNoopTracerProvider()
    tracer := provider.Tracer("test")

    // Start a span with an empty context.
    ctx := context.Background()
    ctx, span := tracer.Start(ctx, "operation")

    // Verify that the span's context is empty.
    sc := span.SpanContext()
    // Expect empty TraceID and SpanID.
    if sc.TraceID().IsValid() || sc.SpanID().IsValid() {
    t.Error("expected empty span context")
    }

    // Verify that the span is not recording.
    if span.IsRecording() {
    t.Error("expected span not recording")
    }

    // Verify TracerProvider returned by span is a no-op provider.
    tp := span.TracerProvider()
    if tp == nil {
    t.Error("expected non-nil tracer provider")
    }
    // Check that calling Tracer on the no-op provider returns a no-op tracer.
    tracer2 := tp.Tracer("another")
    _, span2 := tracer2.Start(ctx, "another operation")
    if span2.IsRecording() {
    t.Error("expected span2 not recording")
    }
}

// TestNoopSpanMethods tests that the various no-op span methods execute without errors.
func TestNoopSpanMethods(t *testing.T) {
    // Use the global noopSpanInstance directly.
    span := noopSpanInstance

    // These calls should do nothing and must not panic.
    span.SetStatus(codes.Error, "error occurred")
    span.End()
    span.RecordError(nil)
    span.AddEvent("event")
}

// TestStartWithPreexistingNoopSpan tests that if the context already contains a no-op span, Start reuses it.
func TestStartWithPreexistingNoopSpan(t *testing.T) {
    provider := NewNoopTracerProvider()
    tracer := provider.Tracer("test")

    // Create a context with an existing no-op span.
    ctx := ContextWithSpan(context.Background(), noopSpanInstance)

    // Start a new span using the existing context.
    newCtx, span := tracer.Start(ctx, "operation")

    // Since the context already had a non-recording span, it should not change.
    existingSpan := SpanFromContext(ctx)
    if span != existingSpan {
    t.Error("expected to reuse the existing no-op span")
    }

    // Verify that the new context still returns the same span.
    reusedSpan := SpanFromContext(newCtx)
    if reusedSpan != span {
    t.Error("new context does not contain the expected no-op span")
    }
}