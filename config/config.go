package config

import "os"

type Config struct {
	Port      string
	DBPath    string
	UploadDir string
}

func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "8081"),
		DBPath:    getEnv("DB_PATH", "images.db"),
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
