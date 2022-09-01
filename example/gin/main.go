package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	if tracer := btrace.New(); tracer != nil {
		defer tracer.Shutdown()
	}
	router := gin.Default()
	router.Use(otelgin.Middleware(os.Getenv("BTEL_SERVICE_NAME")))
	router.GET("/user/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.String(http.StatusOK, "Hello %s", name)
	})
	router.Run(":8080")
}
