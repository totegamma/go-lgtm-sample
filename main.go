package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/goark/mt/v2/mt19937"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("main")

func main() {

	// slogのセットアップ
	handler := &CustomHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)}
	slogger := slog.New(handler)
	slog.SetDefault(slogger)

	e := echo.New()
	e.Use(middleware.Recover())

	// トレースのセットアップ
	cleanup, err := setupTraceProvider("tempo:4318", "app", "v0.0.0")
	if err != nil {
		panic(err)
	}
	defer cleanup()
	skipper := otelecho.WithSkipper(
		func(c echo.Context) bool {
			return c.Path() == "/metrics" || c.Path() == "/health"
		},
	)
	e.Use(otelecho.Middleware("app", skipper))

	// アクセスログにtraceID/spanIDを追加
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/metrics" || c.Path() == "/health"
		},
		Format: `{"time":"${time_rfc3339_nano}",${custom},"remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","status":${status},` +
			`"error":"${error}","latency":${latency},"latency_human":"${latency_human}",` +
			`"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
		CustomTagFunc: func(c echo.Context, buf *bytes.Buffer) (int, error) {
			span := trace.SpanFromContext(c.Request().Context())
			buf.WriteString(fmt.Sprintf("\"%s\":\"%s\"", "traceID", span.SpanContext().TraceID().String()))
			buf.WriteString(fmt.Sprintf(",\"%s\":\"%s\"", "spanID", span.SpanContext().SpanID().String()))
			return 0, nil
		},
	}))

	// echoのメトリクスを収集
	e.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Namespace: "app",
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/metrics" || c.Path() == "/health"
		},
	}))

	// traceIDをレスポンスヘッダに追加
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			span := trace.SpanFromContext(c.Request().Context())
			c.Response().Header().Set("trace-id", span.SpanContext().TraceID().String())
			return next(c)
		}
	})

	e.GET("/metrics", echoprometheus.NewHandler())

	// ランダムな時間待機するAPI たまにエラーを返す
	rnd := rand.New(mt19937.New(time.Now().UnixNano()))
	e.GET("/wait", func(c echo.Context) error {
		waitTime := time.Duration(rnd.NormFloat64()*1000+500.0) * time.Millisecond
		if waitTime < 0 || waitTime > time.Second {
			return c.String(http.StatusInternalServerError, "Invalid wait time")
		}
		time.Sleep(waitTime)
		return c.String(http.StatusOK, fmt.Sprintf("Waited for %v ms", waitTime.Milliseconds()))
	})

	// 計算を行うAPI
	e.POST("/calc", CalcHandler)

	e.Logger.Fatal(e.Start(":8000"))
}

func setupTraceProvider(endpoint string, serviceName string, serviceVersion string) (func(), error) {

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(serviceVersion),
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tracerProvider)

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	cleanup := func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer provider: %v", err)
		}
	}
	return cleanup, nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type CustomHandler struct {
	slog.Handler
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {

	r.AddAttrs(slog.String("type", "app"))

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(slog.String("traceID", span.SpanContext().TraceID().String()))
		r.AddAttrs(slog.String("spanID", span.SpanContext().SpanID().String()))
	}

	return h.Handler.Handle(ctx, r)
}
