package post

import (
	"errors"
	"time"

	"app/internal/features/auth"
)

// Categories adalah daftar kategori artikel yang valid.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

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

// UpdateRequest adalah payload pembaruan parsial: kolom yang tidak dikirim tidak diubah.
type UpdateRequest struct {
	Title          *string `json:"title" binding:"omitempty,max=255"`
	Category       *string `json:"category"`
	Excerpt        *string `json:"excerpt"`
	Content        *string `json:"content"`
	CoverImagePath *string `json:"cover_image_path"`
	IsPublished    *bool   `json:"is_published"`
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
	AuthorName     string     `json:"author_name"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PublicPostResponse adalah representasi artikel untuk endpoint publik.
// UserID sengaja tidak disertakan agar informasi internal pemilik tidak terekspos.
type PublicPostResponse struct {
	ID             uint       `json:"id"`
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
	AuthorName     string     `json:"author_name"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// StatsResponse adalah ringkasan 3 kartu statistik penulis.
type StatsResponse struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

// AdminUser adalah baris monitoring pengguna beserta jumlah artikelnya.
type AdminUser struct {
	ID             uint      `gorm:"column:id" json:"id"`
	Username       string    `gorm:"column:username" json:"username"`
	Email          string    `gorm:"column:email" json:"email"`
	Role           string    `gorm:"column:role" json:"role"`
	TotalArticles  int64     `gorm:"column:total_articles" json:"total_articles"`
	TotalPublished int64     `gorm:"column:total_published" json:"total_published"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}
