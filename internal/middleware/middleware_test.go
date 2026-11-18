package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secret = "test-secret"

func token(t *testing.T, key string, method jwt.SigningMethod, exp time.Duration) string {
	t.Helper()
	claims := Claims{
		UserID: 7, Username: "budi", Role: "member",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp))},
	}
	var k any = []byte(key)
	if method == jwt.SigningMethodNone {
		k = jwt.UnsafeAllowNoneSignatureType
	}
	s, err := jwt.NewWithClaims(method, claims).SignedString(k)
	assert.NoError(t, err)
	return s
}

func do(r *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "10.0.0.1:1234"
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := do(r, http.MethodGet, "/x", "")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = do(r, http.MethodOptions, "/x", "")
	assert.Equal(t, 204, w.Code)
}

func TestAuthAndRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", Auth(secret), func(c *gin.Context) {
		u, ok := CurrentUser(c)
		assert.True(t, ok)
		c.JSON(200, gin.H{"id": u.ID, "role": u.Role})
	})
	r.GET("/admin", Auth(secret), RequireRole("admin"), func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/norole", RequireRole("admin"), func(c *gin.Context) { c.String(200, "ok") })

	good := token(t, secret, jwt.SigningMethodHS256, time.Hour)

	assert.Equal(t, 200, do(r, "GET", "/me", "Bearer "+good).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Basic abc").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer ").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer garbage").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+token(t, "other", jwt.SigningMethodHS256, time.Hour)).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+token(t, secret, jwt.SigningMethodHS256, -time.Hour)).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+token(t, secret, jwt.SigningMethodHS512, time.Hour)).Code)

	assert.Equal(t, 403, do(r, "GET", "/admin", "Bearer "+good).Code)
	assert.Equal(t, 401, do(r, "GET", "/norole", "").Code)
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) { c.Set(userKey, AuthUser{ID: 1, Role: "admin"}) },
		RequireRole("member", "admin"), func(c *gin.Context) { c.String(200, "ok") })
	assert.Equal(t, 200, do(r, "GET", "/ok", "").Code)
}

func TestCurrentUserWrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, ok := CurrentUser(c)
	assert.False(t, ok)
	c.Set(userKey, "bukan-AuthUser")
	_, ok = CurrentUser(c)
	assert.False(t, ok)
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimiter(2, 80*time.Millisecond))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 429, do(r, "GET", "/x", "").Code)

	time.Sleep(120 * time.Millisecond)
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
}
