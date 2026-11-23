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
type Service struct{ repo Repository }

// NewService membuat Service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Slugify mengubah judul menjadi slug URL (huruf kecil, angka, tanda hubung).
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

// ReadingTime menghitung estimasi waktu baca dalam menit (kata / 200 wpm, minimal 1).
func ReadingTime(content string) int {
	words := len(strings.Fields(content))
	minutes := (words + wordsPerMinute - 1) / wordsPerMinute
	if minutes < 1 {
		return 1
	}
	return minutes
}

// NormalizeCategory memvalidasi kategori (kosong => Umum).
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

// ResolveCategory memilih kandidat valid pertama, atau kategori default.
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

func makeExcerpt(content string) string {
	text := strings.Join(strings.Fields(content), " ")
	runes := []rune(text)
	if len(runes) <= 160 {
		return text
	}
	return strings.TrimSpace(string(runes[:160])) + "…"
}

// ToResponse mengubah Post menjadi PostResponse.
func ToResponse(p *Post) PostResponse {
	return PostResponse{
		ID: p.ID, UserID: p.UserID, Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		Content: p.Content, CoverImagePath: p.CoverImagePath, IsPublished: p.IsPublished,
		PublishedAt: p.PublishedAt, ReadingTime: ReadingTime(p.Content),
		Author:     AuthorInfo{ID: p.User.ID, Username: p.User.Username},
		AuthorName: p.User.Username,
		CreatedAt:  p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ToPublicResponse mengubah Post menjadi DTO publik tanpa user_id.
func ToPublicResponse(p *Post) PublicPostResponse {
	return PublicPostResponse{
		ID: p.ID, Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		Content: p.Content, CoverImagePath: p.CoverImagePath, IsPublished: p.IsPublished,
		PublishedAt: p.PublishedAt, ReadingTime: ReadingTime(p.Content),
		Author: AuthorInfo{ID: p.User.ID, Username: p.User.Username}, AuthorName: p.User.Username,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toResponses(posts []Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, ToResponse(&posts[i]))
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

// owned mengambil artikel dan memastikan pemanggil adalah pemilik (atau admin).
// Tidak ada => ErrNotFound; milik orang lain => ErrForbidden.
func (s *Service) owned(userID uint, role string, id uint) (*Post, error) {
	p, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if p.UserID != userID && !strings.EqualFold(role, "admin") {
		return nil, ErrForbidden
	}
	return p, nil
}

// ListPublic mengembalikan artikel terbit dengan pencarian & filter kategori.
func (s *Service) ListPublic(q, category string) ([]PublicPostResponse, error) {
	posts, err := s.repo.ListPublished(q, category)
	if err != nil {
		return nil, err
	}
	out := make([]PublicPostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, ToPublicResponse(&posts[i]))
	}
	return out, nil
}

// GetPublic mengembalikan satu artikel terbit berdasarkan slug.
func (s *Service) GetPublic(slug string) (*PublicPostResponse, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	res := ToPublicResponse(p)
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
func (s *Service) Get(userID uint, role string, id uint) (*PostResponse, error) {
	p, err := s.owned(userID, role, id)
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
		UserID: userID, Title: strings.TrimSpace(req.Title), Slug: slug, Category: category,
		Excerpt: strings.TrimSpace(req.Excerpt), Content: req.Content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath), IsPublished: req.IsPublished,
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
	return s.Get(userID, "", p.ID)
}

// Update memperbarui artikel secara parsial (kolom yang tidak dikirim tidak berubah).
func (s *Service) Update(userID uint, role string, id uint, req UpdateRequest) (*PostResponse, error) {
	p, err := s.owned(userID, role, id)
	if err != nil {
		return nil, err
	}
	if req.Category != nil {
		category, err := NormalizeCategory(*req.Category)
		if err != nil {
			return nil, err
		}
		p.Category = category
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		title := strings.TrimSpace(*req.Title)
		if title != p.Title {
			slug, err := s.uniqueSlug(title, p.ID)
			if err != nil {
				return nil, err
			}
			p.Slug = slug
		}
		p.Title = title
	}
	if req.Content != nil && strings.TrimSpace(*req.Content) != "" {
		p.Content = *req.Content
	}
	if req.Excerpt != nil {
		p.Excerpt = strings.TrimSpace(*req.Excerpt)
	}
	if p.Excerpt == "" {
		p.Excerpt = makeExcerpt(p.Content)
	}
	if req.CoverImagePath != nil {
		p.CoverImagePath = strings.TrimSpace(*req.CoverImagePath)
	}
	if req.IsPublished != nil {
		setPublished(p, *req.IsPublished)
	}
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	res := ToResponse(p)
	return &res, nil
}

// setPublished mengatur status terbit beserta published_at.
func setPublished(p *Post, published bool) {
	if published && (!p.IsPublished || p.PublishedAt == nil) {
		now := time.Now()
		p.PublishedAt = &now
	}
	if !published {
		p.PublishedAt = nil
	}
	p.IsPublished = published
}

// TogglePublish membalik status Draf <-> Terbit beserta published_at.
func (s *Service) TogglePublish(userID uint, role string, id uint) (*PostResponse, error) {
	p, err := s.owned(userID, role, id)
	if err != nil {
		return nil, err
	}
	setPublished(p, !p.IsPublished)
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	res := ToResponse(p)
	return &res, nil
}

// Delete menghapus artikel milik penulis (atau oleh admin).
func (s *Service) Delete(userID uint, role string, id uint) error {
	p, err := s.owned(userID, role, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(p)
}

// Stats mengembalikan 3 kartu statistik penulis.
func (s *Service) Stats(userID uint) (*StatsResponse, error) {
	total, published, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}
	return &StatsResponse{TotalArticles: total, TotalSelesai: published, BelumSelesai: total - published}, nil
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

// AdminUserPosts mengembalikan seluruh artikel milik pengguna tertentu (kosong bila tidak ada).
func (s *Service) AdminUserPosts(userID uint) ([]PostResponse, error) {
	return s.ListMine(userID)
}
