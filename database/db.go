package database

import (
	"log"

	"image-analysis-platform/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	err = DB.AutoMigrate(&models.Image{})
	if err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	log.Println("Database connected and migrated successfully")
}
