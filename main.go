package main

import (
	"log"
	"os"

	"image-analysis-platform/config"
	"image-analysis-platform/database"
	"image-analysis-platform/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Ensure the upload directory exists before handlers start writing files.
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory: ", err)
	}

	database.Init(cfg.DBPath)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	routes.Setup(r)

	log.Printf("Starting server on :%s\n", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
