package post

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const wordsPerMinute = 200

// Service berisi logika bisnis artikel.
type Service struct {
	repo Repository
}

// NewService membuat Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Slugify mengubah judul menjadi slug URL.
func Slugify(title string) string {
	var b strings.Builder
	dash := false

	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}

	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		return "artikel"
	}

	return slug
}

// ReadingTime menghitung estimasi waktu baca dalam menit.
func ReadingTime(content string) int {
	words := len(strings.Fields(content))
	minutes := (words + wordsPerMinute - 1) / wordsPerMinute

	if minutes < 1 {
		return 1
	}

	return minutes
}

// NormalizeCategory memvalidasi kategori.
func NormalizeCategory(category string) (string, error) {
	category = strings.TrimSpace(category)

	if category == "" {
		return DefaultCategory, nil
	}

	for _, c := range Categories {
		if strings.EqualFold(c, category) {
			return c, nil
		}
	}

	return "", ErrInvalidCategory
}

// ResolveCategory memilih kandidat kategori valid pertama.
func ResolveCategory(candidates ...string) string {
	for _, cand := range candidates {
		if strings.TrimSpace(cand) == "" {
			continue
		}

		if c, err := NormalizeCategory(cand); err == nil {
			return c
		}
	}

	return DefaultCategory
}

// makeExcerpt membuat ringkasan maksimal 160 karakter.
func makeExcerpt(content string) string {
	text := strings.Join(strings.Fields(content), " ")
	runes := []rune(text)

	if len(runes) <= 160 {
		return text
	}

	return strings.TrimSpace(string(runes[:160])) + "\u2026"
}

// ToResponse mengubah Post menjadi PostResponse.
func ToResponse(p *Post) PostResponse {
	return PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    ReadingTime(p.Content),
		Author: AuthorInfo{
			ID:       p.User.ID,
			Username: p.User.Username,
		},
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// ToPublicResponse mengubah Post menjadi respons artikel publik.
func ToPublicResponse(p *Post) PublicPostResponse {
	return PublicPostResponse{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		CoverImagePath: p.CoverImagePath,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    ReadingTime(p.Content),
		AuthorName:     p.User.Username,
	}
}

func toResponses(posts []Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))

	for i := range posts {
		out = append(out, ToResponse(&posts[i]))
	}

	return out
}

func toPublicResponses(posts []Post) []PublicPostResponse {
	out := make([]PublicPostResponse, 0, len(posts))

	for i := range posts {
		out = append(out, ToPublicResponse(&posts[i]))
	}

	return out
}

func (s *Service) uniqueSlug(title string, excludeID uint) (string, error) {
	base := Slugify(title)
	candidate := base

	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(candidate, excludeID)
		if err != nil {
			return "", err
		}

		if !exists {
			return candidate, nil
		}

		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// mine mengambil artikel dan memeriksa kepemilikannya.
func (s *Service) mine(userID, id uint) (*Post, error) {
	p, err := s.repo.FindByID(id)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	if p.UserID != userID {
		return nil, ErrForbidden
	}

	return p, nil
}

// ListPublic mengembalikan artikel terbit untuk pengunjung publik.
func (s *Service) ListPublic(q, category string) ([]PublicPostResponse, error) {
	posts, err := s.repo.ListPublished(q, category)
	if err != nil {
		return nil, err
	}

	return toPublicResponses(posts), nil
}

// GetPublic mengembalikan satu artikel terbit berdasarkan slug.
func (s *Service) GetPublic(slug string) (*PostResponse, error) {
	p, err := s.repo.FindPublishedBySlug(slug)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	res := ToResponse(p)
	return &res, nil
}

// ListMine mengembalikan seluruh artikel milik penulis.
func (s *Service) ListMine(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	return toResponses(posts), nil
}

// Get mengembalikan satu artikel milik penulis.
func (s *Service) Get(userID, id uint) (*PostResponse, error) {
	p, err := s.mine(userID, id)
	if err != nil {
		return nil, err
	}

	res := ToResponse(p)
	return &res, nil
}

// Create membuat artikel baru dengan slug unik.
func (s *Service) Create(userID uint, req PostRequest) (*PostResponse, error) {
	category, err := NormalizeCategory(req.Category)
	if err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(req.Title, 0)
	if err != nil {
		return nil, err
	}

	p := &Post{
		UserID:         userID,
		Title:          strings.TrimSpace(req.Title),
		Slug:           slug,
		Category:       category,
		Excerpt:        strings.TrimSpace(req.Excerpt),
		Content:        req.Content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath),
		IsPublished:    req.IsPublished,
	}

	if p.Excerpt == "" {
		p.Excerpt = makeExcerpt(req.Content)
	}

	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	}

	if err := s.repo.Create(p); err != nil {
		return nil, err
	}

	return s.Get(userID, p.ID)
}

// Update memperbarui artikel milik penulis.
func (s *Service) Update(userID, id uint, req PostRequest) (*PostResponse, error) {
	p, err := s.mine(userID, id)
	if err != nil {
		return nil, err
	}

	category, err := NormalizeCategory(req.Category)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)

	if title != p.Title {
		slug, err := s.uniqueSlug(title, p.ID)
		if err != nil {
			return nil, err
		}

		p.Slug = slug
	}

	p.Title = title
	p.Category = category
	p.Excerpt = strings.TrimSpace(req.Excerpt)

	if p.Excerpt == "" {
		p.Excerpt = makeExcerpt(req.Content)
	}

	p.Content = req.Content
	p.CoverImagePath = strings.TrimSpace(req.CoverImagePath)

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	res := ToResponse(p)
	return &res, nil
}

// TogglePublish membalik status Draf dan Terbit.
func (s *Service) TogglePublish(userID, id uint) (*PostResponse, error) {
	p, err := s.mine(userID, id)
	if err != nil {
		return nil, err
	}

	p.IsPublished = !p.IsPublished

	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	res := ToResponse(p)
	return &res, nil
}

// Delete menghapus artikel milik penulis.
func (s *Service) Delete(userID, id uint) error {
	p, err := s.mine(userID, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(p)
}

// Stats mengembalikan statistik artikel penulis.
func (s *Service) Stats(userID uint) (*StatsResponse, error) {
	total, published, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}

	draft := total - published

	return &StatsResponse{
		Total:         total,
		Published:     published,
		Draft:         draft,
		TotalArticles: total,
		TotalSelesai:  published,
		BelumSelesai:  draft,
	}, nil
}

// AdminUsers mengembalikan seluruh pengguna beserta jumlah artikelnya.
func (s *Service) AdminUsers() ([]AdminUser, error) {
	users, err := s.repo.ListUsersWithCounts()
	if err != nil {
		return nil, err
	}

	if users == nil {
		users = []AdminUser{}
	}

	return users, nil
}

// AdminPosts mengembalikan seluruh artikel dari semua penulis.
func (s *Service) AdminPosts() ([]PostResponse, error) {
	posts, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}

	return toResponses(posts), nil
}

// AdminUserPosts mengembalikan artikel milik pengguna tertentu.
func (s *Service) AdminUserPosts(userID uint) ([]PostResponse, error) {
	exists, err := s.repo.UserExists(userID)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrNotFound
	}

	return s.ListMine(userID)
}
