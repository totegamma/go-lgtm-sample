package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"app/internal/testutil"
)

var checker *tracetest.InMemoryExporter

func TestMain(m *testing.M) {

	checker = testutil.SetupMockTraceProvider()

	m.Run()
}

func TestCalcBasic(t *testing.T) {

	ctx, id := testutil.SetupTraceCtx()

	expr := CalcNode{
		Operation: "+",
		A:         1.0,
		B:         2.0,
	}

	result, err := Calc(ctx, expr)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, result)

	testutil.PrintSpans(t, checker.GetSpans(), id)
}

func TestCalcInvalidInput(t *testing.T) {

	ctx, id := testutil.SetupTraceCtx()

	expr := CalcNode{
		Operation: "/",
		A:         1.0,
		B: map[string]any{
			"op": "-",
			"a":  1.0,
			"b":  1.0,
		},
	}

	result, err := Calc(ctx, expr)
	if assert.NoError(t, err) {
		assert.Equal(t, 3.0, result)
	}

	testutil.PrintSpans(t, checker.GetSpans(), id)
}
