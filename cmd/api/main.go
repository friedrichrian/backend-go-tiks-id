package main

import (
	"fmt"
	"backend/internal/config"
	"backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	r := gin.Default()

	// Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/", handler.Hello)

	// Run server
	r.Run(fmt.Sprintf(":%s", cfg.AppPort))
}
