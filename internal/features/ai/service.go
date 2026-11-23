package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"app/internal/features/post"
)

// Error domain AI.
var (
	ErrUpstream          = errors.New("layanan AI tidak dapat diakses")
	ErrInvalidAIResponse = errors.New("respons AI tidak berisi artikel yang valid")
)

// Searcher menelusuri web; Completer memanggil model bahasa.
type (
	Searcher interface {
		Search(ctx context.Context, query string) ([]Source, error)
	}
	Completer interface {
		Complete(ctx context.Context, messages []Message) (string, error)
	}
)

// Service menghasilkan draf artikel utuh dengan (opsional) konteks Search Engine.
type Service struct {
	search Searcher
	chat   Completer
}

// NewService membuat Service.
func NewService(search Searcher, chat Completer) *Service {
	return &Service{search: search, chat: chat}
}

type article struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
}

// Generate menjalankan alur: Search Engine -> prompt -> OpenAI -> artikel terstruktur.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	sources := []Source{}
	if req.IncludeWebSearch {
		// Kegagalan penelusuran tidak menggagalkan pembuatan artikel.
		if found, err := s.search.Search(ctx, req.Topic); err == nil {
			sources = found
		}
	}
	raw, err := s.chat.Complete(ctx, buildMessages(req, sources))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvalidAIResponse
	}
	art := parseArticle(raw, req)
	return &GenerateResponse{
		Title:         art.Title,
		Category:      post.ResolveCategory(req.Category, art.Category),
		Excerpt:       art.Excerpt,
		Content:       art.Content,
		Sources:       sources,
		UsedWebSearch: len(sources) > 0,
	}, nil
}

// parseArticle mengurai JSON dari model; bila model membalas teks Markdown biasa,
// seluruh balasan dipakai sebagai isi artikel dengan judul dari topik.
func parseArticle(raw string, req GenerateRequest) article {
	var art article
	if err := json.Unmarshal([]byte(extractJSON(raw)), &art); err != nil ||
		strings.TrimSpace(art.Title) == "" || strings.TrimSpace(art.Content) == "" {
		topic := strings.TrimSpace(req.Topic)
		return article{
			Title:   topic,
			Excerpt: fmt.Sprintf("Artikel ulasan tentang %s.", topic),
			Content: raw,
		}
	}
	art.Title = strings.TrimSpace(art.Title)
	art.Excerpt = strings.TrimSpace(art.Excerpt)
	return art
}

// extractJSON mengambil objek JSON terluar (menyingkirkan pagar ```json dan teks pembuka).
func extractJSON(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return raw
	}
	return raw[start : end+1]
}

func buildMessages(req GenerateRequest, sources []Source) []Message {
	system := "Anda adalah penulis blog profesional berbahasa Indonesia. " +
		"Balas HANYA dengan satu objek JSON valid tanpa teks lain, berisi kunci: " +
		"\"title\" (judul menarik), \"category\" (salah satu dari: " + strings.Join(post.Categories, ", ") + "), " +
		"\"excerpt\" (ringkasan pemikat 1-2 kalimat), dan \"content\" (artikel lengkap dalam format Markdown " +
		"dengan sub-judul, paragraf runtut, dan kesimpulan)."

	var b strings.Builder
	fmt.Fprintf(&b, "Tulis artikel blog tentang topik: %s\n", req.Topic)
	if req.Category != "" {
		fmt.Fprintf(&b, "Kategori yang diinginkan: %s\n", req.Category)
	}
	if len(sources) > 0 {
		b.WriteString("\nGunakan fakta terkini dari hasil penelusuran web berikut dan sitasi dengan penanda [1], [2], dst.:\n")
		for i, src := range sources {
			fmt.Fprintf(&b, "[%d] %s — %s (%s)\n", i+1, src.Title, src.Snippet, src.URL)
		}
	}
	return []Message{{Role: "system", Content: system}, {Role: "user", Content: b.String()}}
}
