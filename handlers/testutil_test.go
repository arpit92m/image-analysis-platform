package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"image-analysis-platform/database"
	"image-analysis-platform/middleware"
	"image-analysis-platform/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	t.Setenv("UPLOAD_DIR", t.TempDir())

	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}
	database.DB.AutoMigrate(&models.Image{}, &models.User{})
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", Register)
		auth.POST("/login", Login)
		auth.POST("/refresh", RefreshToken)
	}

	images := v1.Group("/images")
	images.Use(middleware.AuthRequired())
	{
		images.POST("", UploadImage)
		images.GET("", ListImages)
		images.GET("/:id", GetImage)
		images.PUT("/:id", UpdateImage)
		images.DELETE("/:id", DeleteImage)
		images.GET("/:id/download", DownloadImage)
	}

	return r
}

func performRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBytes)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func getAuthToken(r *gin.Engine) string {
	// register
	performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "testuser",
		"password": "testpass123",
	}, "")

	// login
	w := performRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "testuser",
		"password": "testpass123",
	}, "")

	var resp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.AccessToken
}

// helper to generate a token for a known user without HTTP calls
func generateTestToken(userID uint, username string) string {
	token, _ := middleware.GenerateAccessToken(userID, username)
	return token
}
