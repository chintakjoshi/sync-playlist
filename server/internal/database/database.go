package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Global DB instance
var DB *gorm.DB

type User struct {
	gorm.Model
	GoogleID  string `gorm:"uniqueIndex"`
	Email     string `gorm:"uniqueIndex"`
	Name      string
	AvatarURL string
}

type UserService struct {
	gorm.Model
	UserID          uint   `gorm:"not null"`
	ServiceType     string `gorm:"not null"` // "spotify", "youtube"
	AccessToken     string
	RefreshToken    string
	TokenExpiry     int64
	ServiceUserID   string // User ID from the service
	ServiceUserName string
}

func InitDB() error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto migrate tables
	err = db.AutoMigrate(&User{}, &UserService{})
	if err != nil {
		return err
	}

	DB = db
	log.Println("Database connection established and tables migrated")
	return nil
}
