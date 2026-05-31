package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"image-analysis-platform/models"
)

func TestRegister(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	w := performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "newuser",
		"password": "password123",
	}, "")

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["username"] != "newuser" {
		t.Errorf("expected username 'newuser', got %v", resp["username"])
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	body := map[string]string{
		"username": "dupuser",
		"password": "password123",
	}

	performRequest(r, "POST", "/api/v1/auth/register", body, "")

	w := performRequest(r, "POST", "/api/v1/auth/register", body, "")
	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	// missing password
	w := performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "user1",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing password, got %d", w.Code)
	}

	// short username
	w = performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "ab",
		"password": "password123",
	}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for short username, got %d", w.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "loginuser",
		"password": "password123",
	}, "")

	w := performRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "loginuser",
		"password": "password123",
	}, "")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "user2",
		"password": "password123",
	}, "")

	w := performRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "user2",
		"password": "wrongpassword",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRefreshToken(t *testing.T) {
	setupTestDB()
	r := setupRouter()

	performRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "refreshuser",
		"password": "password123",
	}, "")

	w := performRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "refreshuser",
		"password": "password123",
	}, "")

	var loginResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &loginResp)

	w = performRequest(r, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": loginResp.RefreshToken,
	}, "")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var refreshResp models.TokenResponse
	json.Unmarshal(w.Body.Bytes(), &refreshResp)
	if refreshResp.AccessToken == "" {
		t.Error("expected new access token after refresh")
	}
}
