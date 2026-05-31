package handlers

import (
	"fmt"
	"net/http"
	"time"

	"image-analysis-platform/database"
	"image-analysis-platform/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadImage handles POST /api/v1/images
func UploadImage(c *gin.Context) {
	var req models.ImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	image := models.Image{
		UserID:           req.UserID,
		OriginalFilename: req.OriginalFilename,
		UploadDate:       time.Now(),
		Width:            req.Width,
		Height:           req.Height,
		FileSize:         req.FileSize,
		FileType:         req.FileType,
		StoragePath:      fmt.Sprintf("uploads/%s/%s", req.UserID, uuid.New().String()),
		AnalysisStatus:   "pending",
	}

	result := database.DB.Create(&image)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image metadata"})
		return
	}

	c.JSON(http.StatusCreated, image)
}

// ListImages handles GET /api/v1/images?user_id=xxx
func ListImages(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query parameter is required"})
		return
	}

	var images []models.Image
	result := database.DB.Where("user_id = ?", userID).Order("upload_date desc").Find(&images)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// GetImage handles GET /api/v1/images/:id
func GetImage(c *gin.Context) {
	id := c.Param("id")

	var image models.Image
	result := database.DB.First(&image, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
}

// UpdateImage handles PUT /api/v1/images/:id
func UpdateImage(c *gin.Context) {
	id := c.Param("id")

	var image models.Image
	if err := database.DB.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	var req models.ImageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.OriginalFilename != "" {
		updates["original_filename"] = req.OriginalFilename
	}
	if req.Width > 0 {
		updates["width"] = req.Width
	}
	if req.Height > 0 {
		updates["height"] = req.Height
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	database.DB.Model(&image).Updates(updates)

	// reload to get updated values
	database.DB.First(&image, id)

	c.JSON(http.StatusOK, image)
}

// DeleteImage handles DELETE /api/v1/images/:id
func DeleteImage(c *gin.Context) {
	id := c.Param("id")

	var image models.Image
	if err := database.DB.First(&image, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	database.DB.Delete(&image)

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}
