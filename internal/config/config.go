// Package config memuat konfigurasi aplikasi dari environment / berkas .env (gaya Laravel).
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config menampung seluruh konfigurasi runtime aplikasi.
type Config struct {
	AppEnv        string
	AppPort       string
	AppKey        string
	AdminUsername string
	AdminPassword string

	DBConnection string
	DBHost       string
	DBPort       string
	DBDatabase   string
	DBUsername   string
	DBPassword   string
	DBSSLMode    string

	SearchEngineEndpoint string
	OpenAIEndpoint       string
	OpenAIKey            string
	OpenAIModel          string
}

// Load membaca .env (jika ada) lalu variabel lingkungan dengan nilai default.
func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		AppEnv:        get("APP_ENV", "development"),
		AppPort:       get("APP_PORT", "8081"),
		AppKey:        get("APP_KEY", "delcom_blog_secret_key_super_secure_2026_random_string_32chars"),
		AdminUsername: get("ADMIN_USERNAME", "admin"),
		AdminPassword: get("ADMIN_PASSWORD", "admin"),

		DBConnection: get("DB_CONNECTION", "pgsql"),
		DBHost:       get("DB_HOST", "127.0.0.1"),
		DBPort:       get("DB_PORT", "5432"),
		DBDatabase:   get("DB_DATABASE", "pabwe_p9_blog"),
		DBUsername:   get("DB_USERNAME", "postgres"),
		DBPassword:   get("DB_PASSWORD", "postgres"),
		DBSSLMode:    get("DB_SSLMODE", "disable"),

		SearchEngineEndpoint: get("SEARCH_ENGINE_ENDPOINT", "https://delcom.org/api/v1/search"),
		OpenAIEndpoint:       get("OPENAI_ENDPOINT", "https://delcom.org/api/v1"),
		OpenAIKey:            get("OPENAI_KEY", "key"),
		OpenAIModel:          get("OPENAI_MODEL", "nama-model"),
	}
}

// DSN membangun connection string PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBPort, c.DBUsername, c.DBPassword, c.DBDatabase, c.DBSSLMode,
	)
}

// Addr mengembalikan alamat listen Gin, contoh ":8081".
func (c *Config) Addr() string {
	return ":" + c.AppPort
}

func get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
