package trace

import (
    "context"
    "testing"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

// TestNewNoopTracerProvider tests that the provider returns no-op tracer and span.
func TestNewNoopTracerProvider(t *testing.T) {
    provider := NewNoopTracerProvider()

    // Get a tracer and start a span from an empty context.
    tracer := provider.Tracer("test")
    ctx, span := tracer.Start(context.Background(), "testSpan")

    // Since no-op span should always return false for IsRecording.
    if span.IsRecording() {
    t.Error("expected span.IsRecording() to be false")
    }

    // Check that the span context is empty.
    sc := span.SpanContext()
    // For an empty span context, both TraceID and SpanID are invalid.
    if sc.TraceID().IsValid() || sc.SpanID().IsValid() {
    t.Error("expected empty span context")
    }

    // Verify that the span's associated tracer provider is non-nil.
    if span.TracerProvider() == nil {
    t.Error("expected non-nil TracerProvider")
    }

    // Check that putting the span into the context and retrieving it works.
    ctx = ContextWithSpan(ctx, span)
    rSpan := SpanFromContext(ctx)
    if rSpan != span {
    t.Error("expected roundtrip span correctly retrieved from context")
    }
}

// TestNoopSpanMethods calls all noopSpan methods to ensure they do not panic.
func TestNoopSpanMethods(t *testing.T) {
    // Use the singleton instance directly.
    s := noopSpanInstance

    // Call all no-op methods.
    s.SetStatus(codes.Error, "error occurred")
    s.SetAttributes(attribute.String("key", "value"))
    s.End()
    s.RecordError(nil)
    s.AddEvent("testEvent")
    // For AddLink, we pass nil as a no-op.
    s.AddLink(Link{})
    s.SetName("newName")

    // Test that calling noopTracer.Start always returns noopSpanInstance.
    tr := noopTracer{}
    _, newSpan := tr.Start(context.Background(), "anotherTestSpan")
    if newSpan != noopSpanInstance {
    t.Error("expected the span returned by noopTracer.Start to be noopSpanInstance")
    }
}