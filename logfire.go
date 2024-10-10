// Package logfire uses OpenTelemetry to send logs to the Logfire API.
//
// This package is not affiliated with Pydantic. See https://pydantic.dev/ to learn more.
package logfire

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"

	otellog "go.opentelemetry.io/otel/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	serviceVersion         = "0.0.1"
	defaultLogfireEndpoint = "https://logfire-api.pydantic.dev/v1"
	logfireTracerName      = "logfire"
)

var (
	globalTracer      oteltrace.Tracer
	globalServiceName string
	globalLogger      *SpanLogger
)

// config is the config that is required to initialize the logfire logger.
type config struct {
	// ServiceName refers to the service this logger is for.
	ServiceName string
	// APIToken is the Write API token for logfire.
	APIToken string
	// The endpoint to logfire.
	Endpoint string
}

// Option is a function type that modifies Config.
type Option func(*config)

// WithServiceName sets the service name in the Config.
func WithServiceName(name string) Option {
	return func(c *config) {
		c.ServiceName = name
	}
}

// WithEndpoint sets the endpoint in the Config.
func WithEndpoint(endpoint string) Option {
	return func(c *config) {
		c.Endpoint = endpoint
	}
}

// WithAPIToken sets the API token in the Config.
func WithAPIToken(token string) Option {
	return func(c *config) {
		c.APIToken = token
	}
}

// newConfigWithDefaults creates a new Config with default values and applies the given options.
func newConfigWithDefaults(options ...Option) *config {
	config := &config{
		APIToken: os.Getenv("LOGFIRE_TOKEN"),
		Endpoint: defaultLogfireEndpoint,
	}

	for _, option := range options {
		option(config)
	}

	return config
}

// Returns the logfire service name.
func ServiceName() string {
	return globalServiceName
}

// Initialize initializes the logfire logger.  This must be called at the start of the program.
func Initialize(ctx context.Context, opts ...Option) (func(), error) {
	config := newConfigWithDefaults(opts...)

	globalServiceName = config.ServiceName

	if config.APIToken == "" {
		return nil, errors.New("config.APIToken is required")
	}

	var headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", config.APIToken),
	}

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(config.Endpoint+"/traces"),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	resources, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	provider := sdktrace.NewTracerProvider(
		// TODO: This doesn't seem to send live log events?
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(1*time.Second)),
		sdktrace.WithResource(resources),
	)

	otel.SetTracerProvider(provider)

	globalTracer = otel.Tracer(logfireTracerName)
	globalLogger = &SpanLogger{
		spanCtx: context.Background(),
		// This is unused for the global logger.  You should not
		// attempt to close the global logger, or it will panic!
		span: nil,
	}

	return func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

func sendLog(ctx context.Context, msg string, severity otellog.Severity) {
	_, span := globalTracer.Start(ctx, msg)
	defer span.End()

	// Add some attributes to the span
	span.SetAttributes(
		attribute.String("logfire.span_type", "log"),
		attribute.String("logfire.msg_template", "log message template"),
		attribute.String("logfire.msg", msg),
		attribute.Int("logfire.level_num", int(severity)),
	)
}

// Tracer returns an OpenTelemetry Tracer that can be used to hook into other
// OpenTelemetry integrations.  Integrations using this tracer will send logs directly
// to Logfire.
func Tracer() oteltrace.Tracer {
	if globalTracer == nil {
		panic("did you forget to call Initialize()?")
	}
	return globalTracer
}

// Trace logs a message to Logfire with severity Trace.
func Trace(msg string) {
	globalLogger.Trace(msg)
}

// Debug logs a message to Logfire with severity Debug.
func Debug(msg string) {
	globalLogger.Debug(msg)
}

// Info logs a message to Logfire with severity Info.
func Info(msg string) {
	globalLogger.Info(msg)
}

// Warn logs a message to Logfire with severity Warn.
func Warn(msg string) {
	globalLogger.Warn(msg)
}

// Error logs a message to Logfire with severity Error.
func Error(msg string) {
	globalLogger.Error(msg)
}

// Fatal logs a message to Logfire with severity Fatal.
func Fatal(msg string) {
	globalLogger.Fatal(msg)
}

// SpanLogger creates a span for the current context.  The SpanLogger is also aware of
// the context in which the span was created, and can be used to create child spans.
type SpanLogger struct {
	spanCtx context.Context
	span    oteltrace.Span
}

// Trace logs a message in the current span context to Logfire with severity Trace.
func (s *SpanLogger) Trace(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityTrace)
}

// Debug logs a message in the current span context to Logfire with severity Debug.
func (s *SpanLogger) Debug(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityDebug)
}

// Info logs a message in the current span context to Logfire with severity Info.
func (s *SpanLogger) Info(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityInfo)
}

// Warn logs a message in the current span context to Logfire with severity Warn.
func (s *SpanLogger) Warn(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityWarn)
}

// Error logs a message in the current span context to Logfire with severity Error.
func (s *SpanLogger) Error(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityError)
}

// Fatal logs a message in the current span context to Logfire with severity Fatal.
func (s *SpanLogger) Fatal(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityFatal)
}

// Context returns the context of the current span.
func (s *SpanLogger) Context() context.Context {
	return s.spanCtx
}

// Close ends the current span.
func (s *SpanLogger) Close() {
	s.span.End()
}

// NewSpanLogger creates a new child SpanLogger from the given context.
// Use this if you want to create or "nest" a new Span.
func NewSpanLogger(ctx context.Context, spanName string) *SpanLogger {
	spanCtx, span := globalTracer.Start(ctx, spanName)
	return &SpanLogger{
		spanCtx: spanCtx,
		span:    span,
	}
}

// FromContext creates a new SpanLogger from the given context.
// Use this if you want to use the same Span as the context you're in.
func FromContext(ctx context.Context) *SpanLogger {
	span := oteltrace.SpanFromContext(ctx)
	return &SpanLogger{
		spanCtx: ctx,
		span:    span,
	}
}
