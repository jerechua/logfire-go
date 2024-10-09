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

// Config is the config that is required to initialize the logfire logger.
type Config struct {
	// ServiceName refers to the service this logger is for.
	ServiceName string
	// APIToken is the Write API token for logfire.
	APIToken string
	// The endpoint to logfire.
	Endpoint string
}

// Option is a function type that modifies Config
type Option func(*Config)

// WithServiceName sets the service name in the Config
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithEndpoint sets the endpoint in the Config
func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.Endpoint = endpoint
	}
}

// WithAPIToken sets the API token in the Config
func WithAPIToken(token string) Option {
	return func(c *Config) {
		c.APIToken = token
	}
}

// newConfigWithDefaults creates a new Config with default values and applies the given options
func newConfigWithDefaults(options ...Option) *Config {
	config := &Config{
		APIToken: os.Getenv("LOGFIRE_TOKEN"),
		Endpoint: defaultLogfireEndpoint,
	}

	for _, option := range options {
		option(config)
	}

	return config
}

// Initialize initializes the logfire logger.  This must be called at the start of the program.
func Initialize(ctx context.Context, opts ...Option) (func(), error) {
	config := newConfigWithDefaults(opts...)

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

var globalTracer oteltrace.Tracer

func Trace(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityTrace)
}

func Debug(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityDebug)
}

func Info(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityInfo)
}

func Warn(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityWarn)
}

func Error(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityError)
}

func Fatal(msg string) {
	sendLog(context.Background(), msg, otellog.SeverityFatal)
}

type SpanLogger struct {
	spanCtx context.Context
	span    oteltrace.Span
}

func (s *SpanLogger) Trace(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityTrace)
}

func (s *SpanLogger) Debug(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityDebug)
}

func (s *SpanLogger) Info(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityInfo)
}

func (s *SpanLogger) Warn(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityWarn)
}

func (s *SpanLogger) Error(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityError)
}

func (s *SpanLogger) Fatal(msg string) {
	sendLog(s.spanCtx, msg, otellog.SeverityFatal)
}

func (s *SpanLogger) Context() context.Context {
	return s.spanCtx
}

func (s *SpanLogger) Close() {
	s.span.End()
}

func NewSpanLogger(ctx context.Context, spanName string) *SpanLogger {
	spanCtx, span := globalTracer.Start(ctx, spanName)
	return &SpanLogger{
		spanCtx: spanCtx,
		span:    span,
	}
}
