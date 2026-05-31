package models

import "time"

type Image struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	UserID           string    `json:"user_id" gorm:"index;not null"`
	OriginalFilename string    `json:"original_filename" gorm:"not null"`
	UploadDate       time.Time `json:"upload_date" gorm:"autoCreateTime"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	FileSize         int64     `json:"file_size"`
	FileType         string    `json:"file_type"`
	StoragePath      string    `json:"storage_path"`
	AnalysisStatus   string    `json:"analysis_status" gorm:"default:pending"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ImageUploadRequest struct {
	UserID           string `json:"user_id" binding:"required"`
	OriginalFilename string `json:"original_filename" binding:"required"`
	Width            int    `json:"width" binding:"required,gt=0"`
	Height           int    `json:"height" binding:"required,gt=0"`
	FileSize         int64  `json:"file_size" binding:"required,gt=0"`
	FileType         string `json:"file_type" binding:"required,oneof=image/jpeg image/png image/gif image/webp"`
}

type ImageUpdateRequest struct {
	OriginalFilename string `json:"original_filename"`
	Width            int    `json:"width" binding:"omitempty,gt=0"`
	Height           int    `json:"height" binding:"omitempty,gt=0"`
}
