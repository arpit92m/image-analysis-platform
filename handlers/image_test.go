package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

func TestUploadImage(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	body := map[string]interface{}{
		"user_id":           "user-123",
		"original_filename": "photo.jpg",
		"width":             1920,
		"height":            1080,
		"file_size":         2048000,
		"file_type":         "image/jpeg",
	}

	w := performRequest(r, "POST", "/api/v1/images", body, token)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["original_filename"] != "photo.jpg" {
		t.Errorf("expected filename 'photo.jpg', got %v", resp["original_filename"])
	}
	if resp["analysis_status"] != "pending" {
		t.Errorf("expected analysis_status 'pending', got %v", resp["analysis_status"])
	}
	storagePath, ok := resp["storage_path"].(string)
	if !ok || storagePath == "" {
		t.Fatalf("expected non-empty storage_path, got %v", resp["storage_path"])
	}
	if _, err := os.Stat(storagePath); err != nil {
		t.Fatalf("expected upload file at %s: %v", storagePath, err)
	}
}

func TestUploadImageValidation(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	// missing required fields
	w := performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id": "user-123",
	}, token)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// invalid file type
	w = performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id":           "user-123",
		"original_filename": "doc.pdf",
		"width":             100,
		"height":            100,
		"file_size":         1024,
		"file_type":         "application/pdf",
	}, token)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid file type, got %d", w.Code)
	}
}

func TestUploadImageUnauthorized(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]interface{}{
		"user_id":           "user-123",
		"original_filename": "photo.jpg",
		"width":             1920,
		"height":            1080,
		"file_size":         2048000,
		"file_type":         "image/jpeg",
	}

	w := performRequest(r, "POST", "/api/v1/images", body, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestListImages(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	// upload two images
	for i := 0; i < 2; i++ {
		performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
			"user_id":           "user-list",
			"original_filename": fmt.Sprintf("img%d.png", i),
			"width":             800,
			"height":            600,
			"file_size":         1024,
			"file_type":         "image/png",
		}, token)
	}

	w := performRequest(r, "GET", "/api/v1/images?user_id=user-list", nil, token)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	total := resp["total"].(float64)
	if total != 2 {
		t.Errorf("expected 2 images, got %v", total)
	}
}

func TestListImagesMissingUserID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	w := performRequest(r, "GET", "/api/v1/images", nil, token)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetImage(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	// upload an image first
	w := performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id":           "user-get",
		"original_filename": "test.jpg",
		"width":             640,
		"height":            480,
		"file_size":         512,
		"file_type":         "image/jpeg",
	}, token)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := fmt.Sprintf("%.0f", created["id"].(float64))

	w = performRequest(r, "GET", "/api/v1/images/"+id, nil, token)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetImageNotFound(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	w := performRequest(r, "GET", "/api/v1/images/99999", nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateImage(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	w := performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id":           "user-update",
		"original_filename": "old_name.jpg",
		"width":             640,
		"height":            480,
		"file_size":         512,
		"file_type":         "image/jpeg",
	}, token)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := fmt.Sprintf("%.0f", created["id"].(float64))

	w = performRequest(r, "PUT", "/api/v1/images/"+id, map[string]interface{}{
		"original_filename": "new_name.jpg",
		"width":             1024,
	}, token)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &updated)
	if updated["original_filename"] != "new_name.jpg" {
		t.Errorf("expected filename 'new_name.jpg', got %v", updated["original_filename"])
	}
}

func TestDeleteImage(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	w := performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id":           "user-del",
		"original_filename": "todelete.png",
		"width":             100,
		"height":            100,
		"file_size":         256,
		"file_type":         "image/png",
	}, token)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := fmt.Sprintf("%.0f", created["id"].(float64))
	storagePath := created["storage_path"].(string)

	w = performRequest(r, "DELETE", "/api/v1/images/"+id, nil, token)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if _, err := os.Stat(storagePath); !os.IsNotExist(err) {
		t.Fatalf("expected deleted upload file at %s, got stat err %v", storagePath, err)
	}

	// verify it's gone
	w = performRequest(r, "GET", "/api/v1/images/"+id, nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", w.Code)
	}
}

func TestDownloadImage(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()
	token := getAuthToken(r)

	w := performRequest(r, "POST", "/api/v1/images", map[string]interface{}{
		"user_id":           "user-dl",
		"original_filename": "download.jpg",
		"width":             1920,
		"height":            1080,
		"file_size":         4096,
		"file_type":         "image/jpeg",
	}, token)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := fmt.Sprintf("%.0f", created["id"].(float64))

	w = performRequest(r, "GET", "/api/v1/images/"+id+"/download", nil, token)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["download_url"] == nil || resp["download_url"] == "" {
		t.Error("expected non-empty download_url")
	}
	if resp["filename"] != "download.jpg" {
		t.Errorf("expected filename 'download.jpg', got %v", resp["filename"])
	}
}
