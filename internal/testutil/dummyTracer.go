package testutil

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var tracer = otel.Tracer("testutil")

func SetupMockTraceProvider() *tracetest.InMemoryExporter {

	spanChecker := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(spanChecker))
	otel.SetTracerProvider(provider)

	return spanChecker
}

func SetupTraceCtx() (context.Context, string) {
	ctx, span := tracer.Start(context.Background(), "testRoot")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()

	return ctx, traceID
}

func unMarshalAttrValue(val attribute.Value) string {
	switch val.Type() {
	case attribute.INVALID:
		return "INVALID"
	case attribute.BOOL:
		return fmt.Sprintf("%t", val.AsBool())
	case attribute.INT64:
		return fmt.Sprintf("%d", val.AsInt64())
	case attribute.FLOAT64:
		return fmt.Sprintf("%f", val.AsFloat64())
	case attribute.STRING:
		return val.AsString()
	case attribute.BOOLSLICE:
		return fmt.Sprintf("%v", val.AsBoolSlice())
	case attribute.INT64SLICE:
		return fmt.Sprintf("%v", val.AsInt64Slice())
	case attribute.FLOAT64SLICE:
		return fmt.Sprintf("%v", val.AsFloat64Slice())
	case attribute.STRINGSLICE:
		return fmt.Sprintf("%v", val.AsStringSlice())
	default:
		return "UNKNOWN"
	}
}

func PrintSpans(t *testing.T, spans tracetest.SpanStubs, traceID string) {
	t.Logf("---- %s ----\n", traceID)

	var found bool = false

	for _, span := range spans {
		if !(span.SpanContext.TraceID().String() == traceID) {
			continue
		}

		if span.Name == "testRoot" {
			continue
		}

		found = true

		t.Logf("Name: %s\n", span.Name)
		t.Logf("Attributes:\n")
		for _, attr := range span.Attributes {
			t.Logf("  %s: %s: %s\n", attr.Key, attr.Value.Type().String(), unMarshalAttrValue(attr.Value))
		}
		t.Logf("Events:\n")
		for _, event := range span.Events {
			t.Logf("  %s\n", event.Name)
			for _, attr := range event.Attributes {
				t.Logf("    %s: %s: %s\n", attr.Key, attr.Value.Type().String(), unMarshalAttrValue(attr.Value))
			}
		}
		t.Logf("--------------------------------\n")
	}

	if !found {
		t.Logf("Span not found. spans:\n")
		for _, span := range spans {
			t.Logf("%s(%s)\n", span.Name, span.SpanContext.TraceID().String())
		}
	}
}
