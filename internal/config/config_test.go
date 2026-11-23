package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"APP_ENV", "PORT", "APP_PORT", "DB_HOST", "OPENAI_KEY", "OPENAI_API_KEY", "OPENAI_MODEL"} {
		t.Setenv(k, "")
	}
	cfg := Load()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "key", cfg.OpenAIKey)
	assert.Equal(t, "nama-model", cfg.OpenAIModel)
	assert.Equal(t, ":8081", cfg.Addr())

	// OPENAI_API_KEY dipakai bila OPENAI_KEY tidak diset.
	t.Setenv("OPENAI_API_KEY", "sk-fallback")
	assert.Equal(t, "sk-fallback", Load().OpenAIKey)
}

func TestPortOverridesAppPort(t *testing.T) {
	t.Setenv("APP_PORT", "9999")
	t.Setenv("PORT", "7777")
	assert.Equal(t, ":7777", Load().Addr())
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("PORT", "")
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
