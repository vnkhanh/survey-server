package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var DB *gorm.DB
var PORT string
var JWT_SECRET string

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	PORT = os.Getenv("PORT")
	JWT_SECRET = os.Getenv("JWT_SECRET")
}
