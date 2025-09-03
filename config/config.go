package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vnkhanh/survey-server/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB khởi tạo kết nối PostgreSQL, connection pool và migrate
func ConnectDB() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// DSN chuẩn cho PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		host, user, password, dbName, port,
	)

	// Kết nối DB
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // chỉ log cảnh báo/lỗi
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Connection Pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB from gorm: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)           // số kết nối idle
	sqlDB.SetMaxOpenConns(100)          // số kết nối tối đa
	sqlDB.SetConnMaxLifetime(time.Hour) // tuổi thọ connection

	// Auto migrate các bảng
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
