package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	serviceName    = "my-service"
	serviceVersion = "1.0.0"
	endpoint       = "https://logfire-api.pydantic.dev/v1"
)

var headers = map[string]string{
	"Authorization": fmt.Sprintf("Bearer %s", os.Getenv("LOGFIRE_TOKEN")),
}

func initTracer() func() {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(endpoint+"/traces"),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)

	otel.SetTracerProvider(provider)

	return func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func main() {
	// Ensure LOGFIRE_WRITE_TOKEN is set
	if os.Getenv("LOGFIRE_TOKEN") == "" {
		log.Fatal("LOGFIRE_TOKEN environment variable is not set")
	}

	shutdown := initTracer()
	defer shutdown()

	tracer := otel.Tracer("example-tracer")

	_, span := tracer.Start(context.Background(), "example-operation")
	defer span.End()

	// Add some attributes to the span
	span.SetAttributes(
		attribute.String("logfire.span_type", "log"),
		attribute.String("logfire.msg_template", "log message template"),
		attribute.String("logfire.msg", "log message sent from Go!"),
		attribute.Int("logfire.level_num", 13),
	)

	fmt.Println("Trace sent to Logfire via OpenTelemetry")
}
