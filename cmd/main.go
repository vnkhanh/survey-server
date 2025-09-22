package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/routes"
	"gorm.io/gorm"
)

func main() {
	// Kết nối DB
	config.ConnectDB()

	// Tạo bảng tự động
	if err := config.DB.AutoMigrate(&models.KhaoSat{}, &models.Answer{}); err != nil {
		log.Fatalf("AutoMigrate lỗi: %v", err)
	}

	// Seed khảo sát demo (nếu chưa có)
	var demoSurvey models.KhaoSat
	if err := config.DB.First(&demoSurvey, "tieu_de = ?", "Khảo sát demo").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			limit := 3 // giới hạn số lần trả lời
			demoSurvey = models.KhaoSat{
				TieuDe:    "Khảo sát demo",
				MoTa:      "Đây là khảo sát test",
				GioiHanTL: &limit,
			}
			config.DB.Create(&demoSurvey)
		}
	}
	// In ra ID của khảo sát demo
	log.Printf("Demo survey ID: %d\n", demoSurvey.ID)
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

	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}

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
