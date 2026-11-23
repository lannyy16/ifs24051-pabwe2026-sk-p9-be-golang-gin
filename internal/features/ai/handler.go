package ai

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"app/internal/features/post"
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
		response.Fail(c, 400, "Field topic wajib diisi (minimal 3 karakter)")
		return
	}
	topic := strings.TrimSpace(req.Topic)
	if len([]rune(topic)) < 3 {
		response.Fail(c, 400, "Field topic wajib diisi (minimal 3 karakter)")
		return
	}
	res, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		// Fallback agar fitur tetap dapat digunakan ketika provider AI tidak
		// dikonfigurasi atau sedang tidak tersedia. Ini draf template, bukan
		// hasil generasi model eksternal.
		category := post.ResolveCategory(req.Category)
		res = &GenerateResponse{
			Title: topic, Category: category,
			Excerpt: fmt.Sprintf("Artikel pengantar mengenai %s.", topic),
			Content: fmt.Sprintf("# %s\n\n## Pendahuluan\n\n%s merupakan topik yang dapat dibahas dari berbagai sudut pandang.\n\n## Pembahasan\n\nJelaskan konsep utama, manfaat, tantangan, serta contoh penerapan %s berdasarkan referensi yang tepercaya.\n\n## Kesimpulan\n\nRangkum poin-poin penting dan sertakan sumber yang relevan sebelum menerbitkan artikel ini.", topic, topic, topic),
			Sources: []Source{}, UsedWebSearch: false,
		}
		response.Success(c, 200, "Draf template dibuat karena layanan AI tidak tersedia", res)
		return
	}
	response.Success(c, 200, "Draf artikel berhasil dihasilkan", res)
}
