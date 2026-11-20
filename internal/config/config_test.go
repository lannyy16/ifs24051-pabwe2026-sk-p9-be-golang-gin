package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"APP_ENV", "APP_PORT", "DB_HOST", "OPENAI_MODEL"} {
		t.Setenv(k, "")
	}
	cfg := Load()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "nama-model", cfg.OpenAIModel)
	assert.Equal(t, ":8081", cfg.Addr())
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_PORT", "9999")
	t.Setenv("DB_DATABASE", "mydb")
	t.Setenv("DB_USERNAME", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_SSLMODE", "require")
	cfg := Load()
	assert.Equal(t, ":9999", cfg.Addr())
	assert.Equal(t,
		"host=db port=5433 user=u password=p dbname=mydb sslmode=require TimeZone=Asia/Jakarta",
		cfg.DSN())
}
