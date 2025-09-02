package config

import (
	"fmt"
	"log"
	"os"

	"github.com/vnkhanh/survey-server/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB khởi tạo kết nối PostgreSQL và migrate bảng
func ConnectDB() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Render sẽ cấp PORT app riêng, ta không dùng ở đây
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		host, user, password, dbName, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto migrate bảng
	err = db.AutoMigrate(
		&models.NguoiDung{},
		&models.KhaoSat{},
		&models.CauHoi{},
		&models.CauTraLoi{},
		&models.LuaChon{},
		&models.PhanHoi{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	DB = db
	log.Println("Connected to PostgreSQL & migrated successfully")
}
