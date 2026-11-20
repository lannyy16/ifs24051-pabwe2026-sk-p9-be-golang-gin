package ai

import (
	"github.com/gin-gonic/gin"

	"app/internal/response"
)

// Handler adalah controller POST /api/posts/ai-generate.
type Handler struct{ svc *Service }

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Generate menghasilkan draf artikel dengan AI.
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Topik wajib diisi (minimal 3 karakter)")
		return
	}
	res, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 502, "Gagal menghasilkan artikel dengan AI: "+err.Error())
		return
	}
	response.Success(c, 200, "Draf artikel berhasil dihasilkan", res)
}
