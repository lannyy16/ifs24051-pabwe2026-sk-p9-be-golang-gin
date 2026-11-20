package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/features/auth"
	"app/internal/middleware"
)

const secret = "post-test-secret"

var errBoom = errors.New("boom")

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&auth.User{}, &Post{}))
	return db
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, _ := db.DB()
	require.NoError(t, sqlDB.Close())
}

func mkUser(t *testing.T, db *gorm.DB, name, role string) (auth.User, string) {
	t.Helper()
	u := auth.User{Username: name, Email: name + "@x.id", Password: "x", Role: role}
	require.NoError(t, db.Create(&u).Error)
	claims := middleware.Claims{
		UserID: u.ID, Username: u.Username, Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return u, tok
}

func router(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/public/categories", h.Categories)
	r.GET("/public/posts", h.PublicList)
	r.GET("/public/posts/:slug", h.PublicDetail)
	g := r.Group("/posts", middleware.Auth(secret))
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/stats", h.Stats)
	g.GET("/:id", h.Detail)
	g.PUT("/:id", h.Update)
	g.PATCH("/:id/toggle-publish", h.TogglePublish)
	g.DELETE("/:id", h.Delete)
	a := r.Group("/admin", middleware.Auth(secret), middleware.RequireRole(auth.RoleAdmin))
	a.GET("/users", h.AdminUsers)
	a.GET("/posts", h.AdminPosts)
	a.GET("/users/:id/posts", h.AdminUserPosts)
	return r
}

func call(r *gin.Engine, method, path, token string, body any) (int, map[string]any) {
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func dataMap(out map[string]any) map[string]any { return out["data"].(map[string]any) }
func dataList(out map[string]any) []any         { return out["data"].([]any) }

func TestHelpers(t *testing.T) {
	assert.Equal(t, "halo-dunia-2026", Slugify("  Halo, Dunia!! 2026  "))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))

	c, err := NormalizeCategory("")
	assert.NoError(t, err)
	assert.Equal(t, "Umum", c)
	c, err = NormalizeCategory("teknologi")
	assert.NoError(t, err)
	assert.Equal(t, "Teknologi", c)
	_, err = NormalizeCategory("Olahraga")
	assert.ErrorIs(t, err, ErrInvalidCategory)

	assert.Equal(t, "Bisnis", ResolveCategory("", "Olahraga", "bisnis", "Opini"))
	assert.Equal(t, "Umum", ResolveCategory("", "Olahraga"))

	assert.Equal(t, "pendek", makeExcerpt("  pendek  "))
	long := makeExcerpt(strings.Repeat("a", 200))
	assert.True(t, strings.HasSuffix(long, "…"))
}

func TestAuthorFlowAndPublicEndpoints(t *testing.T) {
	db := newDB(t)
	r := router(NewHandler(NewService(NewRepository(db))))
	_, tok := mkUser(t, db, "penulis", auth.RoleMember)
	_, other := mkUser(t, db, "lain", auth.RoleMember)

	// kategori
	code, out := call(r, "GET", "/public/categories", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 6)

	// create
	body := gin.H{"title": "Belajar Golang Gin", "content": "isi artikel golang", "category": "Teknologi", "is_published": true}
	code, out = call(r, "POST", "/posts", tok, body)
	assert.Equal(t, 201, code)
	first := dataMap(out)
	assert.Equal(t, "belajar-golang-gin", first["slug"])
	assert.Equal(t, "penulis", first["author"].(map[string]any)["username"])
	assert.NotNil(t, first["published_at"])
	assert.NotEmpty(t, first["excerpt"])

	// slug unik
	code, out = call(r, "POST", "/posts", tok, gin.H{"title": "Belajar Golang Gin", "content": "draf kedua"})
	assert.Equal(t, 201, code)
	second := dataMap(out)
	assert.Equal(t, "belajar-golang-gin-2", second["slug"])
	assert.Equal(t, "Umum", second["category"])
	assert.Nil(t, second["published_at"])

	// validasi create
	code, _ = call(r, "POST", "/posts", tok, gin.H{"title": "x", "content": "y", "category": "Olahraga"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "POST", "/posts", tok, "{")
	assert.Equal(t, 400, code)

	// list & stats
	code, out = call(r, "GET", "/posts", tok, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 2)
	code, out = call(r, "GET", "/posts/stats", tok, nil)
	assert.Equal(t, 200, code)
	assert.EqualValues(t, 2, dataMap(out)["total"])
	assert.EqualValues(t, 1, dataMap(out)["published"])
	assert.EqualValues(t, 1, dataMap(out)["draft"])
	code, out = call(r, "GET", "/posts/stats", other, nil)
	assert.Equal(t, 200, code)
	assert.EqualValues(t, 0, dataMap(out)["total"])

	// detail
	firstPath := fmt.Sprintf("/posts/%v", first["id"])
	secondPath := fmt.Sprintf("/posts/%v", second["id"])
	code, _ = call(r, "GET", firstPath, tok, nil)
	assert.Equal(t, 200, code)
	code, _ = call(r, "GET", firstPath, other, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "GET", "/posts/999", tok, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "GET", "/posts/abc", tok, nil)
	assert.Equal(t, 400, code)
	code, _ = call(r, "GET", "/posts/0", tok, nil)
	assert.Equal(t, 400, code)

	// update: judul berubah => slug baru
	code, out = call(r, "PUT", firstPath, tok, gin.H{"title": "Judul Baru", "content": "konten baru", "category": "Bisnis", "excerpt": "ringkas"})
	assert.Equal(t, 200, code)
	assert.Equal(t, "judul-baru", dataMap(out)["slug"])
	assert.Equal(t, "ringkas", dataMap(out)["excerpt"])
	// judul sama => slug tetap
	code, out = call(r, "PUT", firstPath, tok, gin.H{"title": "Judul Baru", "content": "konten lagi", "category": "Bisnis"})
	assert.Equal(t, 200, code)
	assert.Equal(t, "judul-baru", dataMap(out)["slug"])
	code, _ = call(r, "PUT", firstPath, tok, gin.H{"title": "J", "content": "k", "category": "Olahraga"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", firstPath, tok, "{")
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/posts/abc", tok, gin.H{"title": "J", "content": "k"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/posts/999", tok, gin.H{"title": "J", "content": "k"})
	assert.Equal(t, 404, code)

	// toggle: draf -> terbit -> draf
	code, out = call(r, "PATCH", secondPath+"/toggle-publish", tok, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, true, dataMap(out)["is_published"])
	code, out = call(r, "PATCH", secondPath+"/toggle-publish", tok, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, false, dataMap(out)["is_published"])
	code, _ = call(r, "PATCH", "/posts/abc/toggle-publish", tok, nil)
	assert.Equal(t, 400, code)
	code, _ = call(r, "PATCH", "/posts/999/toggle-publish", tok, nil)
	assert.Equal(t, 404, code)

	// publik
	code, out = call(r, "GET", "/public/posts", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 1)
	code, out = call(r, "GET", "/public/posts?q=KONTEN&category=Bisnis", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 1)
	code, out = call(r, "GET", "/public/posts?q=tidak-ada", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 0)
	code, out = call(r, "GET", "/public/posts/judul-baru", "", nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "Judul Baru", dataMap(out)["title"])
	code, _ = call(r, "GET", "/public/posts/belajar-golang-gin-2", "", nil)
	assert.Equal(t, 404, code)

	// hapus
	code, _ = call(r, "DELETE", firstPath, other, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "DELETE", firstPath, tok, nil)
	assert.Equal(t, 200, code)
	code, _ = call(r, "DELETE", "/posts/abc", tok, nil)
	assert.Equal(t, 400, code)
}

func TestAdminEndpoints(t *testing.T) {
	db := newDB(t)
	r := router(NewHandler(NewService(NewRepository(db))))
	_, admin := mkUser(t, db, "admin", auth.RoleAdmin)
	member, mtok := mkUser(t, db, "member", auth.RoleMember)
	call(r, "POST", "/posts", mtok, gin.H{"title": "A", "content": "isi", "is_published": true})
	call(r, "POST", "/posts", mtok, gin.H{"title": "B", "content": "isi"})

	code, _ := call(r, "GET", "/admin/users", mtok, nil)
	assert.Equal(t, 403, code)

	code, out := call(r, "GET", "/admin/users", admin, nil)
	assert.Equal(t, 200, code)
	users := dataList(out)
	require.Len(t, users, 2)
	row := users[1].(map[string]any)
	assert.Equal(t, "member", row["username"])
	assert.EqualValues(t, 2, row["total_posts"])
	assert.EqualValues(t, 1, row["total_published"])

	code, out = call(r, "GET", "/admin/posts", admin, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 2)

	code, out = call(r, "GET", fmt.Sprintf("/admin/users/%d/posts", member.ID), admin, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, dataList(out), 2)
	code, _ = call(r, "GET", "/admin/users/999/posts", admin, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "GET", "/admin/users/abc/posts", admin, nil)
	assert.Equal(t, 400, code)
}

func TestClosedDatabaseReturns500(t *testing.T) {
	db := newDB(t)
	r := router(NewHandler(NewService(NewRepository(db))))
	_, tok := mkUser(t, db, "admin", auth.RoleAdmin)
	closeDB(t, db)

	for _, c := range []struct{ method, path string }{
		{"GET", "/public/posts"}, {"GET", "/public/posts/x"},
		{"GET", "/posts"}, {"GET", "/posts/stats"}, {"GET", "/posts/1"},
		{"PATCH", "/posts/1/toggle-publish"}, {"DELETE", "/posts/1"},
		{"GET", "/admin/users"}, {"GET", "/admin/posts"}, {"GET", "/admin/users/1/posts"},
	} {
		code, _ := call(r, c.method, c.path, tok, nil)
		assert.Equal(t, 500, code, c.method+" "+c.path)
	}
	code, _ := call(r, "POST", "/posts", tok, gin.H{"title": "x", "content": "y"})
	assert.Equal(t, 500, code)
	code, _ = call(r, "PUT", "/posts/1", tok, gin.H{"title": "x", "content": "y"})
	assert.Equal(t, 500, code)
}

type fakeRepo struct {
	Repository
	failSlug, failCreate, failUpdate, failDelete, failExists, nilUsers bool
}

func (f *fakeRepo) SlugExists(s string, id uint) (bool, error) {
	if f.failSlug {
		return false, errBoom
	}
	return f.Repository.SlugExists(s, id)
}
func (f *fakeRepo) Create(p *Post) error {
	if f.failCreate {
		return errBoom
	}
	return f.Repository.Create(p)
}
func (f *fakeRepo) Update(p *Post) error {
	if f.failUpdate {
		return errBoom
	}
	return f.Repository.Update(p)
}
func (f *fakeRepo) Delete(p *Post) error {
	if f.failDelete {
		return errBoom
	}
	return f.Repository.Delete(p)
}
func (f *fakeRepo) UserExists(id uint) (bool, error) {
	if f.failExists {
		return false, errBoom
	}
	return f.Repository.UserExists(id)
}
func (f *fakeRepo) ListUsersWithCounts() ([]AdminUser, error) {
	if f.nilUsers {
		return nil, nil
	}
	return f.Repository.ListUsersWithCounts()
}

func TestServiceErrorBranches(t *testing.T) {
	db := newDB(t)
	real := NewRepository(db)
	u, _ := mkUser(t, db, "svc", auth.RoleMember)
	seed, err := NewService(real).Create(u.ID, PostRequest{Title: "Seed", Content: "isi"})
	require.NoError(t, err)

	svc := func(f *fakeRepo) *Service { f.Repository = real; return NewService(f) }
	req := PostRequest{Title: "Judul Lain", Content: "isi"}

	_, err = svc(&fakeRepo{failSlug: true}).Create(u.ID, req)
	assert.ErrorIs(t, err, errBoom)
	_, err = svc(&fakeRepo{failCreate: true}).Create(u.ID, req)
	assert.ErrorIs(t, err, errBoom)

	_, err = svc(&fakeRepo{failSlug: true}).Update(u.ID, seed.ID, req)
	assert.ErrorIs(t, err, errBoom)
	_, err = svc(&fakeRepo{failUpdate: true}).Update(u.ID, seed.ID, req)
	assert.ErrorIs(t, err, errBoom)
	_, err = svc(&fakeRepo{failUpdate: true}).TogglePublish(u.ID, seed.ID)
	assert.ErrorIs(t, err, errBoom)
	assert.ErrorIs(t, svc(&fakeRepo{failDelete: true}).Delete(u.ID, seed.ID), errBoom)

	_, err = svc(&fakeRepo{failExists: true}).AdminUserPosts(u.ID)
	assert.ErrorIs(t, err, errBoom)

	users, err := svc(&fakeRepo{nilUsers: true}).AdminUsers()
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Empty(t, users)
}

func TestWriteErrorDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	writeError(c, errBoom)
	assert.Equal(t, 500, w.Code)
}
