package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type apiClient struct {
	baseURL          string
	client           *http.Client
	writeUploadFiles bool
}

type registerResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type imageUploadRequest struct {
	UserID           string `json:"user_id"`
	OriginalFilename string `json:"original_filename"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	FileSize         int64  `json:"file_size"`
	FileType         string `json:"file_type"`
}

type imageUpdateRequest struct {
	OriginalFilename string `json:"original_filename,omitempty"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
}

type imageResponse struct {
	ID               uint   `json:"id"`
	UserID           string `json:"user_id"`
	OriginalFilename string `json:"original_filename"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	FileSize         int64  `json:"file_size"`
	FileType         string `json:"file_type"`
	StoragePath      string `json:"storage_path"`
	AnalysisStatus   string `json:"analysis_status"`
}

type listImagesResponse struct {
	Images  []imageResponse `json:"images"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

type downloadResponse struct {
	ImageID     uint   `json:"image_id"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url"`
	FileSize    int64  `json:"file_size"`
	FileType    string `json:"file_type"`
}

func main() {
	baseURL := flag.String("base-url", defaultBaseURL(), "service base URL")
	seedUsers := flag.Int("seed-users", 3, "number of users to create for DB seeding")
	seedImages := flag.Int("seed-images", 4, "number of image metadata rows to create per seed user")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP client timeout")
	writeUploadFiles := flag.Bool("write-upload-files", true, "write local placeholder files for returned storage_path values")
	flag.Parse()

	if *seedUsers < 1 {
		fail("seed-users must be at least 1")
	}
	if *seedImages < 1 {
		fail("seed-images must be at least 1")
	}

	c := apiClient{
		baseURL:          strings.TrimRight(*baseURL, "/"),
		client:           &http.Client{Timeout: *timeout},
		writeUploadFiles: *writeUploadFiles,
	}
	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	password := "demo12345"

	fmt.Printf("Running API verification against %s\n", c.baseURL)

	var verifier registerResponse
	must("health check", func() error {
		var body map[string]string
		if err := c.do(http.MethodGet, "/health", "", nil, http.StatusOK, &body); err != nil {
			return err
		}
		if body["status"] != "ok" {
			return fmt.Errorf("expected health status ok, got %q", body["status"])
		}
		return nil
	})

	verifierUsername := "verify_" + runID
	must("register verifier user", func() error {
		err := c.do(http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"username": verifierUsername,
			"password": password,
		}, http.StatusCreated, &verifier)
		if err != nil {
			return err
		}
		if verifier.ID == 0 || verifier.Username != verifierUsername {
			return fmt.Errorf("unexpected register response: %+v", verifier)
		}
		return nil
	})

	var tokens tokenResponse
	must("login verifier user", func() error {
		err := c.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"username": verifierUsername,
			"password": password,
		}, http.StatusOK, &tokens)
		if err != nil {
			return err
		}
		return validateTokens(tokens)
	})

	must("refresh access token", func() error {
		var refreshed tokenResponse
		err := c.do(http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
			"refresh_token": tokens.RefreshToken,
		}, http.StatusOK, &refreshed)
		if err != nil {
			return err
		}
		if err := validateTokens(refreshed); err != nil {
			return err
		}
		tokens = refreshed
		return nil
	})

	must("reject duplicate registration", func() error {
		return c.do(http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"username": verifierUsername,
			"password": password,
		}, http.StatusConflict, nil)
	})

	must("reject invalid login", func() error {
		return c.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"username": verifierUsername,
			"password": "wrong-password",
		}, http.StatusUnauthorized, nil)
	})

	must("reject missing authorization", func() error {
		return c.do(http.MethodGet, "/api/v1/images", "", nil, http.StatusUnauthorized, nil)
	})

	must("reject invalid bearer token", func() error {
		return c.do(http.MethodGet, "/api/v1/images", "not-a-real-token", nil, http.StatusUnauthorized, nil)
	})

	accessToken := tokens.AccessToken
	verifyUserID := "verify-user-" + runID
	must("reject invalid image type", func() error {
		return c.do(http.MethodPost, "/api/v1/images", accessToken, imageUploadRequest{
			UserID:           verifyUserID,
			OriginalFilename: "not-an-image.txt",
			Width:            1,
			Height:           1,
			FileSize:         1,
			FileType:         "text/plain",
		}, http.StatusBadRequest, nil)
	})

	var throwaway imageResponse
	must("upload throwaway image metadata", func() error {
		err := c.do(http.MethodPost, "/api/v1/images", accessToken, imageUploadRequest{
			UserID:           verifyUserID,
			OriginalFilename: "throwaway.jpg",
			Width:            1920,
			Height:           1080,
			FileSize:         2_048_000,
			FileType:         "image/jpeg",
		}, http.StatusCreated, &throwaway)
		if err != nil {
			return err
		}
		if err := validateImage(throwaway, verifyUserID, "throwaway.jpg"); err != nil {
			return err
		}
		return c.ensureLocalUploadFile(throwaway)
	})

	must("list verifier images", func() error {
		var list listImagesResponse
		path := "/api/v1/images?" + url.Values{
			"user_id":  []string{verifyUserID},
			"page":     []string{"1"},
			"per_page": []string{"20"},
		}.Encode()
		if err := c.do(http.MethodGet, path, accessToken, nil, http.StatusOK, &list); err != nil {
			return err
		}
		if list.Total != 1 || len(list.Images) != 1 || list.Images[0].ID != throwaway.ID {
			return fmt.Errorf("expected one verifier image %d, got total=%d len=%d", throwaway.ID, list.Total, len(list.Images))
		}
		return nil
	})

	must("reject missing image user_id filter", func() error {
		return c.do(http.MethodGet, "/api/v1/images", accessToken, nil, http.StatusBadRequest, nil)
	})

	must("get image details", func() error {
		var got imageResponse
		if err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/images/%d", throwaway.ID), accessToken, nil, http.StatusOK, &got); err != nil {
			return err
		}
		if got.ID != throwaway.ID {
			return fmt.Errorf("expected image ID %d, got %d", throwaway.ID, got.ID)
		}
		return nil
	})

	must("update image metadata", func() error {
		var updated imageResponse
		err := c.do(http.MethodPut, fmt.Sprintf("/api/v1/images/%d", throwaway.ID), accessToken, imageUpdateRequest{
			OriginalFilename: "throwaway-updated.jpg",
			Width:            1280,
			Height:           720,
		}, http.StatusOK, &updated)
		if err != nil {
			return err
		}
		if updated.OriginalFilename != "throwaway-updated.jpg" || updated.Width != 1280 || updated.Height != 720 {
			return fmt.Errorf("unexpected updated image: %+v", updated)
		}
		throwaway = updated
		return nil
	})

	must("reject empty image update", func() error {
		return c.do(http.MethodPut, fmt.Sprintf("/api/v1/images/%d", throwaway.ID), accessToken, map[string]any{}, http.StatusBadRequest, nil)
	})

	must("get image download URL", func() error {
		var download downloadResponse
		if err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/images/%d/download", throwaway.ID), accessToken, nil, http.StatusOK, &download); err != nil {
			return err
		}
		if download.ImageID != throwaway.ID || download.DownloadURL == "" {
			return fmt.Errorf("unexpected download response: %+v", download)
		}
		return nil
	})

	must("delete throwaway image", func() error {
		if err := c.do(http.MethodDelete, fmt.Sprintf("/api/v1/images/%d", throwaway.ID), accessToken, nil, http.StatusOK, nil); err != nil {
			return err
		}
		return c.removeLocalUploadFile(throwaway)
	})

	must("return not found for deleted image", func() error {
		return c.do(http.MethodGet, fmt.Sprintf("/api/v1/images/%d", throwaway.ID), accessToken, nil, http.StatusNotFound, nil)
	})

	totalSeedImages := 0
	for userIndex := 1; userIndex <= *seedUsers; userIndex++ {
		seedUsername := fmt.Sprintf("seed_%s_%02d", runID, userIndex)
		seedUserID := fmt.Sprintf("seed-user-%s-%02d", runID, userIndex)
		seedToken, err := c.createAndLogin(seedUsername, password)
		if err != nil {
			fail("seed user %s: %v", seedUsername, err)
		}

		createdIDs := make([]uint, 0, *seedImages)
		for imageIndex := 1; imageIndex <= *seedImages; imageIndex++ {
			image, err := c.uploadSeedImage(seedToken, seedUserID, userIndex, imageIndex)
			if err != nil {
				fail("seed image user=%s index=%d: %v", seedUserID, imageIndex, err)
			}
			createdIDs = append(createdIDs, image.ID)
			totalSeedImages++
		}

		must(fmt.Sprintf("verify seeded images for %s", seedUserID), func() error {
			var list listImagesResponse
			path := "/api/v1/images?" + url.Values{
				"user_id":  []string{seedUserID},
				"page":     []string{"1"},
				"per_page": []string{"100"},
			}.Encode()
			if err := c.do(http.MethodGet, path, seedToken, nil, http.StatusOK, &list); err != nil {
				return err
			}
			if list.Total != int64(*seedImages) || len(list.Images) != *seedImages {
				return fmt.Errorf("expected %d seeded images, got total=%d len=%d", *seedImages, list.Total, len(list.Images))
			}
			return nil
		})

		fmt.Printf("  seeded user=%s image_ids=%v\n", seedUserID, createdIDs)
	}

	fmt.Println()
	fmt.Println("API verification and DB seeding completed successfully.")
	fmt.Printf("Verifier user: %s (id=%d)\n", verifier.Username, verifier.ID)
	fmt.Printf("Seeded users: %d\n", *seedUsers)
	fmt.Printf("Seeded image metadata rows kept in DB: %d\n", totalSeedImages)
}

func defaultBaseURL() string {
	if baseURL := os.Getenv("API_BASE_URL"); baseURL != "" {
		return baseURL
	}
	return "http://localhost:8081"
}

func (c apiClient) createAndLogin(username string, password string) (string, error) {
	var registered registerResponse
	if err := c.do(http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": username,
		"password": password,
	}, http.StatusCreated, &registered); err != nil {
		return "", fmt.Errorf("register: %w", err)
	}

	var tokens tokenResponse
	if err := c.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	}, http.StatusOK, &tokens); err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	if err := validateTokens(tokens); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

func (c apiClient) uploadSeedImage(token string, userID string, userIndex int, imageIndex int) (imageResponse, error) {
	fileTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	fileType := fileTypes[(imageIndex-1)%len(fileTypes)]
	extension := strings.TrimPrefix(strings.TrimPrefix(fileType, "image/"), "jpeg")
	if extension == "" {
		extension = "jpg"
	}

	req := imageUploadRequest{
		UserID:           userID,
		OriginalFilename: fmt.Sprintf("seed-%02d-%02d.%s", userIndex, imageIndex, extension),
		Width:            800 + userIndex*100 + imageIndex*10,
		Height:           600 + userIndex*50 + imageIndex*10,
		FileSize:         int64(100_000 + userIndex*10_000 + imageIndex*1_000),
		FileType:         fileType,
	}

	var image imageResponse
	if err := c.do(http.MethodPost, "/api/v1/images", token, req, http.StatusCreated, &image); err != nil {
		return imageResponse{}, err
	}
	if err := validateImage(image, userID, req.OriginalFilename); err != nil {
		return imageResponse{}, err
	}
	if err := c.ensureLocalUploadFile(image); err != nil {
		return imageResponse{}, err
	}
	return image, nil
}

func (c apiClient) ensureLocalUploadFile(image imageResponse) error {
	if !c.writeUploadFiles {
		return nil
	}
	if image.StoragePath == "" {
		return fmt.Errorf("storage_path is empty")
	}

	localPath := filepath.Clean(filepath.FromSlash(image.StoragePath))
	if _, err := os.Stat(localPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return writePlaceholderImage(localPath, image.FileType)
}

func (c apiClient) removeLocalUploadFile(image imageResponse) error {
	if !c.writeUploadFiles || image.StoragePath == "" {
		return nil
	}
	err := os.Remove(filepath.Clean(filepath.FromSlash(image.StoragePath)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writePlaceholderImage(localPath string, fileType string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 66, G: 135, B: 245, A: 255})

	switch fileType {
	case "image/jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case "image/png":
		return png.Encode(file, img)
	case "image/gif":
		return gif.Encode(file, img, nil)
	case "image/webp":
		_, err := file.Write(webpPlaceholderBytes())
		return err
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func webpPlaceholderBytes() []byte {
	data, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		return []byte("RIFF\x1a\x00\x00\x00WEBPVP8 \x0e\x00\x00\x00")
	}
	return data
}

func (c apiClient) do(method string, path string, token string, body any, wantStatus int, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("%s %s: expected status %d, got %d body=%s", method, path, wantStatus, resp.StatusCode, string(respBody))
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response body %s: %w", string(respBody), err)
	}
	return nil
}

func validateTokens(tokens tokenResponse) error {
	if tokens.AccessToken == "" {
		return fmt.Errorf("access_token is empty")
	}
	if tokens.RefreshToken == "" {
		return fmt.Errorf("refresh_token is empty")
	}
	if tokens.ExpiresIn <= 0 {
		return fmt.Errorf("expires_in should be positive, got %d", tokens.ExpiresIn)
	}
	return nil
}

func validateImage(image imageResponse, userID string, filename string) error {
	if image.ID == 0 {
		return fmt.Errorf("image ID is empty")
	}
	if image.UserID != userID {
		return fmt.Errorf("expected user_id %q, got %q", userID, image.UserID)
	}
	if image.OriginalFilename != filename {
		return fmt.Errorf("expected filename %q, got %q", filename, image.OriginalFilename)
	}
	if image.AnalysisStatus != "pending" {
		return fmt.Errorf("expected analysis_status pending, got %q", image.AnalysisStatus)
	}
	if image.StoragePath == "" {
		return fmt.Errorf("storage_path is empty")
	}
	return nil
}

func must(name string, fn func() error) {
	if err := fn(); err != nil {
		fail("%s failed: %v", name, err)
	}
	fmt.Printf("[OK] %s\n", name)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
