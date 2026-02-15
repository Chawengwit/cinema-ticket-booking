package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// ใน dev ยังไม่มี reverse proxy จริง ๆ ให้ปิดไปก่อน
	_ = r.SetTrustedProxies(nil)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"env":  os.Getenv("APP_ENV"),
			"port": os.Getenv("PORT"),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_ = r.Run(":" + port)
}
