package routes

import (
	"image-analysis-platform/handlers"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/images", handlers.UploadImage)
		v1.GET("/images", handlers.ListImages)
		v1.GET("/images/:id", handlers.GetImage)
		v1.PUT("/images/:id", handlers.UpdateImage)
		v1.DELETE("/images/:id", handlers.DeleteImage)
	}
}
