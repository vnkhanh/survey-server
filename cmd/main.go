package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/routes"
)

func main() {
	// Kết nối DB + AutoMigrate
	config.ConnectDB()

	// Tạo instance router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Route test server
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Survey server is running")
	})

	// Setup routes khác
	routes.SetupRoutes(r)

	// Lấy PORT từ biến môi trường
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s\n", port)
	r.Run(":" + port)
}
