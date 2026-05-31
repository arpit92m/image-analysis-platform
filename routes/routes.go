package routes

import (
	"image-analysis-platform/handlers"
	"image-analysis-platform/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// public auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/refresh", handlers.RefreshToken)
	}

	// protected image routes
	images := v1.Group("/images")
	images.Use(middleware.AuthRequired())
	{
		images.POST("", handlers.UploadImage)
		images.GET("", handlers.ListImages)
		images.GET("/:id", handlers.GetImage)
		images.PUT("/:id", handlers.UpdateImage)
		images.DELETE("/:id", handlers.DeleteImage)
		images.GET("/:id/download", handlers.DownloadImage)
	}
}
