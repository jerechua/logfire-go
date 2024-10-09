# Logfire Go

Prototype/Proof of concept using Logfire with the OpenTelemetry SDK.

## Env Vars

`LOGFIRE_TOKEN` - The Logfire write token.

### Running the example

```shell
go run examples/simple_logfire/main.go
```

### Running the example with Gin

This uses otelgin's Middleware with some logfire hooks.

```shell
go run examples/gin/main.go
```
