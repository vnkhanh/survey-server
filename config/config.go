package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/vnkhanh/survey-server/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	host := os.Getenv("DB_HOST")
	portStr := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Chuyển port sang int
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid DB_PORT: %v", err)
	}

	// DSN PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		host, user, password, dbName, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // log cảnh báo/lỗi
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB from gorm: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate
	if err := db.AutoMigrate(
		&models.NguoiDung{},
		&models.KhaoSat{},
		&models.CauHoi{},
		&models.CauTraLoi{},
		&models.LuaChon{},
		&models.PhanHoi{},
		&models.Room{},
		&models.RoomNguoiThamGia{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	DB = db
	log.Println("Connected to PostgreSQL & migrated successfully")
}
