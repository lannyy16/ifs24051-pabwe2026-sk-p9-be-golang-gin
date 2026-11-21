package post

import (
	"errors"
	"time"

	"app/internal/features/auth"
)

// Categories adalah daftar kategori artikel yang valid.
var Categories = []string{
	"Teknologi",
	"Pendidikan",
	"Gaya Hidup",
	"Bisnis",
	"Opini",
	"Umum",
}

// DefaultCategory dipakai bila kategori kosong.
const DefaultCategory = "Umum"

// Error domain artikel.
var (
	ErrNotFound        = errors.New("artikel tidak ditemukan")
	ErrForbidden       = errors.New("anda tidak memiliki akses ke artikel ini")
	ErrInvalidCategory = errors.New("kategori tidak valid")
)

// Post adalah model tabel posts.
type Post struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         uint      `gorm:"not null;index"`
	User           auth.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Title          string    `gorm:"size:255;not null"`
	Slug           string    `gorm:"size:255;uniqueIndex;not null"`
	Category       string    `gorm:"size:100;not null"`
	Excerpt        string    `gorm:"type:text"`
	Content        string    `gorm:"type:text"`
	CoverImagePath string    `gorm:"type:text"`
	IsPublished    bool      `gorm:"not null"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PostRequest adalah payload membuat / memperbarui artikel.
type PostRequest struct {
	Title          string `json:"title" binding:"required,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

// AuthorInfo adalah ringkasan penulis pada respons artikel.
type AuthorInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// PostResponse adalah representasi artikel pada API.
type PostResponse struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	AuthorName     string     `json:"author_name"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	Author         AuthorInfo `json:"author"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// StatsResponse adalah ringkasan statistik artikel penulis.
// Field lama dipertahankan agar kompatibel dengan frontend.
type StatsResponse struct {
	Total         int64 `json:"total"`
	Published     int64 `json:"published"`
	Draft         int64 `json:"draft"`
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

// AdminUser adalah baris monitoring pengguna beserta jumlah artikelnya.
type AdminUser struct {
	ID             uint   `gorm:"column:id" json:"id"`
	Username       string `gorm:"column:username" json:"username"`
	Email          string `gorm:"column:email" json:"email"`
	Role           string `gorm:"column:role" json:"role"`
	TotalPosts     int64  `gorm:"column:total_posts" json:"total_posts"`
	TotalPublished int64  `gorm:"column:total_published" json:"total_published"`
}
