package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	logfire "github.com/jerechua/logfire-go"
	logfiregin "github.com/jerechua/logfire-go/gin"
)

func main() {
	closer, err := logfire.Initialize(context.Background(), logfire.WithServiceName("gin-service"))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logfire: %v", err))
	}
	defer closer()

	// Create a default gin router
	router := gin.Default()

	router.Use(logfiregin.Middleware())

	// Define a simple GET route
	router.GET("/hello", func(c *gin.Context) {
		logger := logfire.FromContext(c.Request.Context())
		logger.Info("hellooo logfire!!")

		spanLogger := logfire.NewSpanLogger(c.Request.Context(), "span logger")
		defer spanLogger.Close()
		spanLogger.Info("I am a child span!")

		c.JSON(http.StatusOK, gin.H{
			"message": "Hello, World!",
		})
	})

	// Run the server on port 8080
	router.Run(":8080")
}
