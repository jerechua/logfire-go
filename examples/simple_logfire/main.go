package main

import (
	"context"
	"log"

	logfire "github.com/jerechua/logfire-go"
)

func main() {
	closer, err := logfire.Initialize(context.Background(), logfire.WithServiceName("test-my-service"))
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

	// ctx, outerSpanCloser := logfire.NewSpan(context.Background(), "span wrapper")
	// logfire.InfoCtx(ctx, "something inside the span")
	// time.Sleep(time.Second * 5)
	// defer outerSpanCloser()

	logger := logfire.NewSpanLogger(context.Background(), "span wrapper")
	defer logger.Close()
	logger.Info("something inside the span")

}
