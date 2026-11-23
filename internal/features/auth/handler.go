package auth

import (
	"errors"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

// Handler adalah controller endpoint /api/auth/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register godoc: POST /api/auth/register.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Data registrasi tidak valid: "+err.Error())
		return
	}
	res, err := h.svc.Register(req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 201, "Registrasi berhasil", res)
}

// Login godoc: POST /api/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Username dan password wajib diisi")
		return
	}
	res, err := h.svc.Login(req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Login berhasil", res)
}

// Logout godoc: POST /api/auth/logout (token JWT bersifat stateless).
func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, 200, "Logout berhasil", nil)
}

// Profile godoc: GET /api/auth/profile.
func (h *Handler) Profile(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.Profile(u.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Profil berhasil diambil", res)
}

// UpdateProfile godoc: PUT /api/auth/profile.
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Data profil tidak valid: "+err.Error())
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.UpdateProfile(u.ID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Profil berhasil diperbarui", res)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrEmailTaken):
		response.Fail(c, 400, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		response.Fail(c, 401, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.Fail(c, 404, err.Error())
	default:
		response.Error(c, 500, "Terjadi kesalahan pada server")
	}
}
