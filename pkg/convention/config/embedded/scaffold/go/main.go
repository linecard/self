package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	var listen string
	router := gin.Default()

	if value, exists := os.LookupEnv("AWS_LWA_PORT"); exists {
		listen = "0.0.0.0:" + value
	} else {
		listen = "0.0.0.0:8081"
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, c.Request.Header)
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})

	router.Run(listen)
}
