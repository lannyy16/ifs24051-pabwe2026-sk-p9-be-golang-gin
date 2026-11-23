package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/database"

	"github.com/stretchr/testify/assert"
)

func reset(t *testing.T) {
	t.Helper()
	router = nil
	oc, oe := connect, ensure
	t.Cleanup(func() { router, connect, ensure = nil, oc, oe })
}

func TestHandlerServesAPIWithSQLite(t *testing.T) {
	reset(t)
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", "file:vercel_test?mode=memory&cache=shared")

	// Rewrite gaya Vercel: path asli dibawa lewat query __p.
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest(http.MethodGet, "/api/index?__p=/api/public/posts", nil))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Pengenalan Bahasa Pemrograman Golang Modern")

	// Panggilan kedua memakai router yang sudah dibuat.
	w = httptest.NewRecorder()
	Handler(w, httptest.NewRequest(http.MethodGet, "/api/index?__p=/", nil))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Delcom Blog API")
}

func TestHandlerInitFailures(t *testing.T) {
	reset(t)
	connect = func(*config.Config) (*gorm.DB, error) { return nil, errors.New("db down") }
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "gagal inisialisasi")

	reset(t)
	connect = database.Connect
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", ":memory:")
	ensure = func(*gorm.DB, *config.Config) error { return errors.New("migrate gagal") }
	w = httptest.NewRecorder()
	Handler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, 500, w.Code)
}

func TestDefaultSeams(t *testing.T) {
	assert.NotNil(t, database.Connect)
}
