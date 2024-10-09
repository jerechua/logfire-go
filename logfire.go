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
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	serviceVersion         = "1.0.0"
	defaultLogfireEndpoint = "https://logfire-api.pydantic.dev/v1" // Default OTLP HTTP endpoint
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
func Initialize(opts ...Option) (func(), error) {
	config := newConfigWithDefaults(opts...)

	if config.APIToken == "" {
		return nil, errors.New("config.APIToken is required")
	}

	var headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", config.APIToken),
	}

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(config.Endpoint+"/traces"),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter, trace.WithBatchTimeout(1*time.Second)),
		trace.WithResource(resources),
	)

	otel.SetTracerProvider(provider)

	globalLogger = &logger{
		tracer: otel.Tracer(logfireTracerName),
	}

	return func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

type logger struct {
	tracer oteltrace.Tracer
}

func (l *logger) Log(ctx context.Context, spanName, msg string, severity otellog.Severity) {
	_, span := l.tracer.Start(ctx, spanName)
	defer span.End()

	// Add some attributes to the span
	span.SetAttributes(
		attribute.String("logfire.span_type", "log"),
		attribute.String("logfire.msg_template", "log message template"),
		attribute.String("logfire.msg", msg),
		attribute.Int("logfire.level_num", int(severity)),
	)
}

var globalLogger *logger

func Trace(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityTrace)
}

func Debug(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityDebug)
}

func Info(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityInfo)
}

func Warn(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityWarn)
}

func Error(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityError)
}

func Fatal(msg string) {
	globalLogger.Log(context.Background(), "logger", msg, otellog.SeverityFatal)
}

type SpanLogger struct {
	parentCtx context.Context
	spanCtx   context.Context
	span      oteltrace.Span
}

func (s *SpanLogger) Trace(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityTrace)
}

func (s *SpanLogger) Debug(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityDebug)
}

func (s *SpanLogger) Info(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityInfo)
}

func (s *SpanLogger) Warn(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityWarn)
}

func (s *SpanLogger) Error(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityError)
}

func (s *SpanLogger) Fatal(msg string) {
	globalLogger.Log(s.spanCtx, "logger", msg, otellog.SeverityFatal)
}

func (s *SpanLogger) Close() {
	s.span.End()
}

func NewSpanLogger(ctx context.Context, spanName string) *SpanLogger {
	spanCtx, span := globalLogger.tracer.Start(ctx, spanName)
	return &SpanLogger{
		parentCtx: ctx,
		spanCtx:   spanCtx,
		span:      span,
	}

}
