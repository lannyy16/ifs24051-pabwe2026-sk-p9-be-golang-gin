package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app/docs"
	"app/internal/config"
	"app/internal/database"
)

func setup(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := database.ConnectSQLiteMemory()
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	cfg := &config.Config{AppKey: "router-secret", AdminUsername: "admin", AdminPassword: "admin"}
	require.NoError(t, database.Seed(db, cfg))
	return NewRouter(cfg, db)
}

func call(r *gin.Engine, method, path, token string, body any, hdr ...string) (int, map[string]any, *httptest.ResponseRecorder) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if len(hdr) == 2 {
		req.Header.Set(hdr[0], hdr[1])
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out, w
}

func login(t *testing.T, r *gin.Engine, user, pass string) string {
	t.Helper()
	code, out, _ := call(r, "POST", "/api/auth/login", "", gin.H{"username": user, "password": pass})
	require.Equal(t, 200, code)
	return out["data"].(map[string]any)["token"].(string)
}

func TestHealthRoot(t *testing.T) {
	r := setup(t)
	code, out, _ := call(r, "GET", "/", "", nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "Delcom Blog API berjalan", out["message"])
}

func TestNotFoundAndSwagger(t *testing.T) {
	r := setup(t)
	code, out, _ := call(r, "GET", "/tidak/ada", "", nil)
	assert.Equal(t, 404, code)
	assert.Equal(t, "Endpoint not found", out["message"])

	_, _, w := call(r, "GET", "/swagger/index.html", "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIJSONSchemeAndError(t *testing.T) {
	r := setup(t)
	_, _, w := call(r, "GET", "/api.json", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "http://example.com")

	_, _, w = call(r, "GET", "/api.json", "", nil, "X-Forwarded-Proto", "https")
	assert.Contains(t, w.Body.String(), "https://example.com")

	req := httptest.NewRequest("GET", "/api.json", nil)
	req.TLS = &tls.ConnectionState{}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "https://example.com")

	orig := docs.SpecJSON
	docs.SpecJSON = []byte("{")
	defer func() { docs.SpecJSON = orig }()
	code, out, _ := call(r, "GET", "/api.json", "", nil)
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", out["status"])
}

func TestEndToEndRoutes(t *testing.T) {
	r := setup(t)
	admin := login(t, r, "admin", "admin")
	member := login(t, r, "member", "member")

	code, _, _ := call(r, "GET", "/api/public/categories", "", nil)
	assert.Equal(t, 200, code)
	code, _, _ = call(r, "GET", "/api/admin/users", admin, nil)
	assert.Equal(t, 200, code)
	code, _, _ = call(r, "GET", "/api/admin/users", member, nil)
	assert.Equal(t, 403, code)
	code, _, _ = call(r, "GET", "/api/posts", "", nil)
	assert.Equal(t, 401, code)

	code, out, _ := call(r, "POST", "/api/posts", member, gin.H{"title": "Halo", "content": "dunia", "is_published": true})
	require.Equal(t, 201, code)
	assert.Equal(t, "halo", out["data"].(map[string]any)["slug"])
	code, _, _ = call(r, "GET", "/api/posts/stats", member, nil)
	assert.Equal(t, 200, code)
	code, _, _ = call(r, "GET", "/api/public/posts/halo", "", nil)
	assert.Equal(t, 200, code)
	code, _, _ = call(r, "GET", "/api/auth/profile", member, nil)
	assert.Equal(t, 200, code)
	code, _, _ = call(r, "POST", "/api/posts/ai-generate", member, gin.H{"topic": "x"})
	assert.Equal(t, 400, code)
}
