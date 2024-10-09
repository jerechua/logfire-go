package main

import (
	"context"
	"log"
	"time"

	logfire "github.com/jerechua/logfire-go"
)

func main() {
	closer, err := logfire.Initialize(
		context.Background(),
		logfire.WithServiceName("test-my-service"), // This is optional.
	)
	if err != nil {
		log.Fatalf("Failed to initialize logfire: %v", err)
	}
	defer closer()

	logfire.Trace("This is a trace log!")
	logfire.Debug("This is a debug log!")
	logfire.Info("This is an info log!")
	logfire.Warn("This is a warn log!")
	logfire.Error("This is an error log!")
	logfire.Fatal("This is a fatal log!")

	outerLogger := logfire.NewSpanLogger(context.Background(), "span wrapper")
	defer outerLogger.Close()
	outerLogger.Info("something inside the span")
	time.Sleep(100 * time.Millisecond)

	innerLogger := logfire.NewSpanLogger(outerLogger.Context(), "inner span")
	defer innerLogger.Close()
	innerLogger.Fatal("something fatal inside!")
	time.Sleep(200 * time.Millisecond)

}
