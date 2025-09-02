package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/routes"
)

func main() {
	// Debug: In toàn bộ biến môi trường khi container start
	envs := os.Environ()
	sort.Strings(envs) // sắp xếp cho dễ đọc
	fmt.Println("====== ENVIRONMENT VARIABLES ======")
	for _, e := range envs {
		fmt.Println(e)
	}
	fmt.Println("===================================")

	// Kết nối DB + AutoMigrate
	config.ConnectDB()

	// Tạo instance router
	r := gin.Default()

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
