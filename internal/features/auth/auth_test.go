package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/middleware"
)

const secret = "unit-test-secret"

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&User{}))
	return db
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, _ := db.DB()
	require.NoError(t, sqlDB.Close())
}

func router(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(db), secret))
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	g := r.Group("/", middleware.Auth(secret))
	g.POST("/logout", h.Logout)
	g.GET("/profile", h.Profile)
	g.PUT("/profile", h.UpdateProfile)
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

func tokenOf(out map[string]any) string {
	return out["data"].(map[string]any)["token"].(string)
}

func TestToResponse(t *testing.T) {
	res := ToResponse(&User{ID: 3, Username: "a", Email: "a@x.id", Role: RoleMember})
	assert.Equal(t, uint(3), res.ID)
	assert.Equal(t, "a@x.id", res.Email)
}

func TestRegisterLoginProfileFlow(t *testing.T) {
	r := router(newDB(t))

	code, out := call(r, "POST", "/register", "", gin.H{"username": "budi", "email": "Budi@Mail.com", "password": "rahasia123", "role": "admin"})
	assert.Equal(t, 201, code)
	assert.Equal(t, RoleMember, out["data"].(map[string]any)["user"].(map[string]any)["role"])
	token := tokenOf(out)

	// duplikat
	code, _ = call(r, "POST", "/register", "", gin.H{"username": "budi", "email": "lain@mail.com", "password": "rahasia123"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "POST", "/register", "", gin.H{"username": "budi2", "email": "budi@mail.com", "password": "rahasia123"})
	assert.Equal(t, 400, code)
	// validasi
	code, _ = call(r, "POST", "/register", "", gin.H{"username": "b"})
	assert.Equal(t, 400, code)

	// login dengan username & email
	code, out = call(r, "POST", "/login", "", gin.H{"username": "budi", "password": "rahasia123"})
	assert.Equal(t, 200, code)
	assert.NotEmpty(t, tokenOf(out))
	code, _ = call(r, "POST", "/login", "", gin.H{"username": "budi@mail.com", "password": "rahasia123"})
	assert.Equal(t, 200, code)
	code, _ = call(r, "POST", "/login", "", gin.H{"username": "budi", "password": "salah"})
	assert.Equal(t, 401, code)
	code, _ = call(r, "POST", "/login", "", gin.H{"username": "tidakada", "password": "x"})
	assert.Equal(t, 401, code)
	code, _ = call(r, "POST", "/login", "", "{")
	assert.Equal(t, 400, code)

	// logout & profil
	code, _ = call(r, "POST", "/logout", token, nil)
	assert.Equal(t, 200, code)
	code, out = call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "budi", out["data"].(map[string]any)["username"])

	// update profil
	code, out = call(r, "PUT", "/profile", token, gin.H{"username": "budi_baru", "email": "baru@mail.com", "password": "passwordbaru"})
	assert.Equal(t, 200, code)
	assert.Equal(t, "budi_baru", out["data"].(map[string]any)["username"])
	code, _ = call(r, "POST", "/login", "", gin.H{"username": "budi_baru", "password": "passwordbaru"})
	assert.Equal(t, 200, code)
	code, _ = call(r, "PUT", "/profile", token, gin.H{"username": "budi_baru"}) // tidak berubah
	assert.Equal(t, 200, code)
	code, _ = call(r, "PUT", "/profile", token, "{")
	assert.Equal(t, 400, code)
}

func TestUpdateProfileConflicts(t *testing.T) {
	r := router(newDB(t))
	_, out := call(r, "POST", "/register", "", gin.H{"username": "satu", "email": "satu@x.id", "password": "123456"})
	token := tokenOf(out)
	call(r, "POST", "/register", "", gin.H{"username": "dua", "email": "dua@x.id", "password": "123456"})

	code, _ := call(r, "PUT", "/profile", token, gin.H{"username": "dua"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/profile", token, gin.H{"email": "dua@x.id"})
	assert.Equal(t, 400, code)
}

func TestProfileUserDeleted(t *testing.T) {
	db := newDB(t)
	r := router(db)
	_, out := call(r, "POST", "/register", "", gin.H{"username": "hapus", "email": "h@x.id", "password": "123456"})
	token := tokenOf(out)
	require.NoError(t, db.Exec("DELETE FROM users").Error)

	code, _ := call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "PUT", "/profile", token, gin.H{"username": "xyz"})
	assert.Equal(t, 404, code)
}

func TestDatabaseFailuresReturn500(t *testing.T) {
	db := newDB(t)
	r := router(db)
	_, out := call(r, "POST", "/register", "", gin.H{"username": "tutup", "email": "t@x.id", "password": "123456"})
	token := tokenOf(out)
	closeDB(t, db)

	code, _ := call(r, "POST", "/register", "", gin.H{"username": "baru", "email": "b@x.id", "password": "123456"})
	assert.Equal(t, 500, code)
	code, _ = call(r, "POST", "/login", "", gin.H{"username": "tutup", "password": "123456"})
	assert.Equal(t, 500, code)
	code, _ = call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 500, code)
}

// fakeRepo membungkus repository nyata dan menyuntikkan kegagalan per metode.
type fakeRepo struct {
	Repository
	failUsername, failEmail, failCreate, failUpdate bool
}

var errBoom = errors.New("boom")

func (f *fakeRepo) UsernameExists(u string, id uint) (bool, error) {
	if f.failUsername {
		return false, errBoom
	}
	return f.Repository.UsernameExists(u, id)
}
func (f *fakeRepo) EmailExists(e string, id uint) (bool, error) {
	if f.failEmail {
		return false, errBoom
	}
	return f.Repository.EmailExists(e, id)
}
func (f *fakeRepo) Create(u *User) error {
	if f.failCreate {
		return errBoom
	}
	return f.Repository.Create(u)
}
func (f *fakeRepo) Update(u *User) error {
	if f.failUpdate {
		return errBoom
	}
	return f.Repository.Update(u)
}

func TestServiceErrorBranches(t *testing.T) {
	db := newDB(t)
	real := NewRepository(db)
	seed := &User{Username: "seed", Email: "seed@x.id", Password: "x", Role: RoleMember}
	require.NoError(t, real.Create(seed))
	reg := RegisterRequest{Username: "orang", Email: "o@x.id", Password: "123456"}

	_, err := NewService(&fakeRepo{Repository: real, failUsername: true}, secret).Register(reg)
	assert.ErrorIs(t, err, errBoom)
	_, err = NewService(&fakeRepo{Repository: real, failEmail: true}, secret).Register(reg)
	assert.ErrorIs(t, err, errBoom)
	_, err = NewService(&fakeRepo{Repository: real, failCreate: true}, secret).Register(reg)
	assert.ErrorIs(t, err, errBoom)
	_, err = NewService(real, secret).Register(RegisterRequest{Username: "panjang", Email: "p@x.id", Password: strings.Repeat("x", 80)})
	assert.Error(t, err)

	_, err = NewService(&fakeRepo{Repository: real, failUsername: true}, secret).UpdateProfile(seed.ID, UpdateProfileRequest{Username: "baru"})
	assert.ErrorIs(t, err, errBoom)
	_, err = NewService(&fakeRepo{Repository: real, failEmail: true}, secret).UpdateProfile(seed.ID, UpdateProfileRequest{Email: "baru@x.id"})
	assert.ErrorIs(t, err, errBoom)
	_, err = NewService(real, secret).UpdateProfile(seed.ID, UpdateProfileRequest{Password: strings.Repeat("x", 80)})
	assert.Error(t, err)
	_, err = NewService(&fakeRepo{Repository: real, failUpdate: true}, secret).UpdateProfile(seed.ID, UpdateProfileRequest{Username: "baru"})
	assert.ErrorIs(t, err, errBoom)
}

func TestSignTokenFailure(t *testing.T) {
	db := newDB(t)
	svc := NewService(NewRepository(db), secret)
	orig := signToken
	signToken = func(*jwt.Token, []byte) (string, error) { return "", errBoom }
	defer func() { signToken = orig }()

	_, err := svc.Register(RegisterRequest{Username: "tokenerr", Email: "te@x.id", Password: "123456"})
	assert.ErrorIs(t, err, errBoom)
}

func TestWriteErrorDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	writeError(c, errBoom)
	assert.Equal(t, 500, w.Code)
}
