package post

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

// Handler adalah controller endpoint artikel publik, penulis, dan admin.
type Handler struct{ svc *Service }

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, 404, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Fail(c, 403, err.Error())
	case errors.Is(err, ErrInvalidCategory):
		response.Fail(c, 400, err.Error())
	default:
		response.Error(c, 500, "Terjadi kesalahan pada server")
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, 400, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

// Categories: GET /api/public/categories.
func (h *Handler) Categories(c *gin.Context) {
	response.Success(c, 200, "Kategori berhasil diambil", Categories)
}

// PublicList: GET /api/public/posts?q=&category=.
func (h *Handler) PublicList(c *gin.Context) {
	res, err := h.svc.ListPublic(c.Query("q"), c.Query("category"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diambil", res)
}

// PublicDetail: GET /api/public/posts/:slug.
func (h *Handler) PublicDetail(c *gin.Context) {
	res, err := h.svc.GetPublic(c.Param("slug"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diambil", res)
}

// List: GET /api/posts.
func (h *Handler) List(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.ListMine(u.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diambil", res)
}

// Stats: GET /api/posts/stats.
func (h *Handler) Stats(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.Stats(u.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Statistik berhasil diambil", res)
}

// Detail: GET /api/posts/:id.
func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.Get(u.ID, u.Role, id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diambil", res)
}

// Create: POST /api/posts.
func (h *Handler) Create(c *gin.Context) {
	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Data artikel tidak valid: "+err.Error())
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.Create(u.ID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 201, "Artikel berhasil dibuat", res)
}

// Update: PUT /api/posts/:id.
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Data artikel tidak valid: "+err.Error())
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.Update(u.ID, u.Role, id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diperbarui", res)
}

// TogglePublish: PATCH /api/posts/:id/toggle-publish.
func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.TogglePublish(u.ID, u.Role, id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Status publikasi berhasil diubah", res)
}

// Delete: DELETE /api/posts/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, _ := middleware.CurrentUser(c)
	if err := h.svc.Delete(u.ID, u.Role, id); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil dihapus", nil)
}

// AdminUsers: GET /api/admin/users.
func (h *Handler) AdminUsers(c *gin.Context) {
	res, err := h.svc.AdminUsers()
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Pengguna berhasil diambil", res)
}

// AdminPosts: GET /api/admin/posts.
func (h *Handler) AdminPosts(c *gin.Context) {
	res, err := h.svc.AdminPosts()
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel berhasil diambil", res)
}

// AdminUserPosts: GET /api/admin/users/:id/posts.
func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	res, err := h.svc.AdminUserPosts(id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, 200, "Artikel pengguna berhasil diambil", res)
}
