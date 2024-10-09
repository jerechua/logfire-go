# Logfire Go

Prototype/Proof of concept using Logfire with the OpenTelemetry SDK.

## Getting Started

### Env Vars

Ensure you have the `LOGFIRE_TOKEN` in your environment variables. This should
be a Logfire write token.

### Usage

In the simplest case, you need to initialize the logfire.

```go
closer, err := logfire.Initialize(context.Background())
if err != nil {
    log.Fatalf("Failed to initialize logfire: %v", err)
}
defer closer()
```

Then you can use the logger as you would with any other logger.

```go
logfire.Trace("This is a trace log!")
logfire.Debug("This is a debug log!")
logfire.Info("This is an info log!")
logfire.Warn("This is a warn log!")
logfire.Error("This is an error log!")
logfire.Fatal("This is a fatal log!")
```

### Span Usage

#### Simple Span

```go
logger := logfire.NewSpanLogger(context.Background(), "span wrapper")
defer logger.Close()
logger.Info("something inside the span")
```

#### Nested Spans

```go
outer := logfire.NewSpanLogger(context.Background(), "outer span")
defer outer.Close()

inner := logfire.NewSpanLogger(outer.Context(), "inner span")
defer inner.Close()

inner.Info("nested span")
```

#### Span from Context

Sometimes it's useful to create a span from an existing context that was passed in.  You can attach to the span using:

```go
func myFunc (ctxWithSpan context.Context) {
    logger := logfire.FromContext(ctx)
    defer logger.Close()
    logger.Info("attached logger to Span")
}
```

### Running the example

```shell
go run examples/simple_logfire/main.go
```

### Running the example with Gin

This uses otelgin's Middleware with some logfire hooks.

```shell
go run examples/gin/main.go
```
