package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/jerechua/logfire-go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		otelgin.Middleware(logfire.ServiceName())(c)
		c.Next()
	}
}
