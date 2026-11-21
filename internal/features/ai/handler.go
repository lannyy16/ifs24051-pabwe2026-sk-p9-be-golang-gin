package ai

import (
	"strings"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

// Handler menangani HTTP request untuk fitur AI.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler baru.
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// Generate menghasilkan draf artikel menggunakan AI.
//
// POST /api/posts/ai-generate
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest

	// Validasi request JSON.
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "Topic wajib diisi (minimal 3 karakter)")
		return
	}

	// Memastikan topik tidak hanya berisi spasi.
	req.Topic = strings.TrimSpace(req.Topic)

	if len([]rune(req.Topic)) < 3 {
		response.Fail(c, 400, "Topic wajib diisi (minimal 3 karakter)")
		return
	}

	// Menghapus spasi berlebih pada kategori.
	req.Category = strings.TrimSpace(req.Category)

	// Memproses pembuatan artikel.
	result, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		response.Error(
			c,
			502,
			"Gagal menghasilkan artikel dengan AI: "+err.Error(),
		)
		return
	}

	// Mengirim hasil artikel.
	response.Success(
		c,
		200,
		"Draf artikel berhasil dihasilkan",
		result,
	)
}
