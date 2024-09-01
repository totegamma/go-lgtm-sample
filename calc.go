package main

import (
	"context"
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type CalcNode struct {
	Operation string `json:"op"`
	A         any    `json:"a"`
	B         any    `json:"b"`
}

func CalcHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "Handler.Calc")
	defer span.End()

	var node CalcNode
	if err := c.Bind(&node); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	result, err := Calc(ctx, node)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"result": result})
}

func Calc(ctx context.Context, node CalcNode) (any, error) {
	ctx, span := tracer.Start(ctx, "Calc")
	defer span.End()

	var err error

	lhs := node.A
	lhsNode, ok := lhs.(map[string]any)
	if ok {
		lhs, err = Calc(ctx, CalcNode{
			Operation: lhsNode["op"].(string),
			A:         lhsNode["a"],
			B:         lhsNode["b"],
		})
		if err != nil {
			return nil, err
		}
	}
	A, ok := lhs.(float64)
	if !ok {
		span.SetAttributes(attribute.String("typeOfA", reflect.TypeOf(lhs).String()))
		err = errors.New("failed to convert A to number")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	rhs := node.B
	rhsNode, ok := rhs.(map[string]any)
	if ok {
		rhs, err = Calc(ctx, CalcNode{
			Operation: rhsNode["op"].(string),
			A:         rhsNode["a"],
			B:         rhsNode["b"],
		})
		if err != nil {
			return nil, err
		}
	}
	B, ok := rhs.(float64)
	if !ok {
		span.SetAttributes(attribute.String("typeOfB", reflect.TypeOf(rhs).String()))
		err = errors.New("failed to convert B to number")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("operation", node.Operation),
		attribute.Float64("A", A),
		attribute.Float64("B", B),
	)

	switch node.Operation {
	case "+":
		return A + B, nil
	case "-":
		return A - B, nil
	case "*":
		return A * B, nil
	case "/":
		if B == 0 {
			err = errors.New("division by zero")
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return nil, err
		}
		return A / B, nil
	default:
		err = errors.New("unknown operation")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}
