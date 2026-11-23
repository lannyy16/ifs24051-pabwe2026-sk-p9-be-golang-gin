// Package middleware berisi CORS, autentikasi JWT, pembatasan role, dan rate limiter.
package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"app/internal/response"
)

const userKey = "user"

// AuthUser adalah identitas pengguna yang disuntikkan ke Gin context.
type AuthUser struct {
	ID       uint
	Username string
	Role     string
}

// Claims adalah isi token JWT aplikasi.
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// CORS mengizinkan akses lintas-origin untuk frontend React.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Auth memverifikasi Bearer token JWT dan menyimpan AuthUser pada context.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.AbortFail(c, 401, "Token autentikasi tidak ditemukan")
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(strings.TrimSpace(parts[1]), claims,
			func(_ *jwt.Token) (any, error) { return []byte(secret), nil },
			jwt.WithValidMethods([]string{"HS256"}),
		)
		if err != nil || !token.Valid {
			response.AbortFail(c, 401, "Token tidak valid atau sudah kedaluwarsa")
			return
		}
		c.Set(userKey, AuthUser{ID: claims.UserID, Username: claims.Username, Role: claims.Role})
		c.Next()
	}
}

// CurrentUser mengambil AuthUser dari context.
func CurrentUser(c *gin.Context) (AuthUser, bool) {
	v, exists := c.Get(userKey)
	if !exists {
		return AuthUser{}, false
	}
	u, ok := v.(AuthUser)
	return u, ok
}

// RequireRole hanya meloloskan pengguna dengan salah satu role yang diizinkan.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := CurrentUser(c)
		if !ok {
			response.AbortFail(c, 401, "Autentikasi diperlukan")
			return
		}
		for _, r := range roles {
			if strings.EqualFold(strings.TrimSpace(u.Role), r) {
				c.Next()
				return
			}
		}
		response.AbortFail(c, 403, "Anda tidak memiliki hak akses")
	}
}

type visitor struct {
	count int
	reset time.Time
}

// RateLimiter membatasi jumlah request per IP dalam satu jendela waktu.
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := map[string]*visitor{}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		mu.Lock()
		v, ok := visitors[ip]
		if !ok || now.After(v.reset) {
			v = &visitor{reset: now.Add(window)}
			visitors[ip] = v
		}
		v.count++
		blocked := v.count > limit
		mu.Unlock()
		if blocked {
			response.AbortFail(c, 429, "Terlalu banyak permintaan, coba lagi nanti")
			return
		}
		c.Next()
	}
}
