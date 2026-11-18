package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/middleware"
	"ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/response"
)

type Handler struct {
	DB     *gorm.DB
	Secret string
}

type input struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register digunakan untuk mendaftarkan pengguna baru.
func (h Handler) Register(c *gin.Context) {
	var in input

	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, 400, "Payload tidak valid")
		return
	}

	username := strings.TrimSpace(in.Username)
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if username == "" || email == "" || len(in.Password) < 8 {
		response.Fail(c, 400, "Username, email, dan password minimal 8 karakter wajib diisi")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(in.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		response.Error(c, err)
		return
	}

	u := User{
		Username: username,
		Email:    email,
		Password: string(hash),
		Role:     "author",
	}

	if err := h.DB.Create(&u).Error; err != nil {
		response.Fail(c, 409, "Username atau email sudah digunakan")
		return
	}

	u.Password = ""

	response.Success(c, 201, "Registrasi berhasil", u)
}

// Login digunakan untuk autentikasi pengguna.
func (h Handler) Login(c *gin.Context) {
	var in input

	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, 400, "Payload tidak valid")
		return
	}

	login := strings.TrimSpace(in.Username)

	if login == "" || in.Password == "" {
		response.Fail(c, 400, "Username/email dan password wajib diisi")
		return
	}

	var u User

	err := h.DB.Where(
		"username = ? OR email = ?",
		login,
		strings.ToLower(login),
	).First(&u).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, 401, "Username/email atau password salah")
			return
		}

		response.Error(c, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(u.Password),
		[]byte(in.Password),
	); err != nil {
		response.Fail(c, 401, "Username/email atau password salah")
		return
	}

	token, err := middleware.CreateToken(
		u.ID,
		u.Role,
		h.Secret,
	)

	if err != nil {
		response.Error(c, err)
		return
	}

	u.Password = ""

	response.Success(c, 200, "Login berhasil", gin.H{
		"token": token,
		"user":  u,
	})
}

// Profile digunakan untuk mengambil profil pengguna.
func (h Handler) Profile(c *gin.Context) {
	id, exists := c.Get("userID")

	if !exists {
		response.Fail(c, 401, "Pengguna belum terautentikasi")
		return
	}

	var u User

	if err := h.DB.First(&u, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, 404, "Pengguna tidak ditemukan")
			return
		}

		response.Error(c, err)
		return
	}

	u.Password = ""

	response.Success(c, 200, "Profil berhasil diambil", u)
}

// UpdateProfile digunakan untuk memperbarui profil pengguna.
func (h Handler) UpdateProfile(c *gin.Context) {
	id, exists := c.Get("userID")

	if !exists {
		response.Fail(c, 401, "Pengguna belum terautentikasi")
		return
	}

	var in input

	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, 400, "Payload tidak valid")
		return
	}

	username := strings.TrimSpace(in.Username)
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if username == "" || email == "" {
		response.Fail(c, 400, "Username dan email wajib diisi")
		return
	}

	var u User

	if err := h.DB.First(&u, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, 404, "Pengguna tidak ditemukan")
			return
		}

		response.Error(c, err)
		return
	}

	u.Username = username
	u.Email = email

	if err := h.DB.Save(&u).Error; err != nil {
		response.Fail(c, 409, "Username atau email sudah digunakan")
		return
	}

	u.Password = ""

	response.Success(c, 200, "Profil diperbarui", u)
}
