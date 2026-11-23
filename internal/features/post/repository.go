package post

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository mengabstraksi akses data artikel.
type Repository interface {
	Create(p *Post) error
	Update(p *Post) error
	Delete(p *Post) error
	FindByID(id uint) (*Post, error)
	FindPublishedBySlug(slug string) (*Post, error)
	ListPublished(q, category string) ([]Post, error)
	ListByUser(userID uint) ([]Post, error)
	ListAll() ([]Post, error)
	SlugExists(slug string, excludeID uint) (bool, error)
	CountByUser(userID uint) (total, published int64, err error)
	ListUsersWithCounts() ([]AdminUser, error)
}

type gormRepository struct{ db *gorm.DB }

// NewRepository membuat Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(p *Post) error {
	return r.db.Omit(clause.Associations).Create(p).Error
}

func (r *gormRepository) Update(p *Post) error {
	return r.db.Omit(clause.Associations).Save(p).Error
}

func (r *gormRepository) Delete(p *Post) error {
	return r.db.Delete(&Post{}, p.ID).Error
}

func (r *gormRepository) FindByID(id uint) (*Post, error) {
	var p Post
	if err := r.db.Preload("User").First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) FindPublishedBySlug(slug string) (*Post, error) {
	var p Post
	if err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) ListPublished(q, category string) ([]Post, error) {
	tx := r.db.Preload("User").Where("is_published = ?", true)
	if q = strings.ToLower(strings.TrimSpace(q)); q != "" {
		like := "%" + q + "%"
		tx = tx.Where("(LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?)", like, like, like)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	var posts []Post
	err := tx.Order("published_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) ListByUser(userID uint) ([]Post, error) {
	var posts []Post
	err := r.db.Preload("User").Where("user_id = ?", userID).Order("id DESC").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) ListAll() ([]Post, error) {
	var posts []Post
	err := r.db.Preload("User").Order("id DESC").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) SlugExists(slug string, excludeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&Post{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) CountByUser(userID uint) (int64, int64, error) {
	var row struct {
		Total     int64
		Published int64
	}
	err := r.db.Model(&Post{}).
		Select("COUNT(*) AS total, COUNT(CASE WHEN is_published THEN 1 END) AS published").
		Where("user_id = ?", userID).
		Scan(&row).Error
	return row.Total, row.Published, err
}

func (r *gormRepository) ListUsersWithCounts() ([]AdminUser, error) {
	var rows []AdminUser
	err := r.db.Table("users").
		Select("users.id AS id, users.username AS username, users.email AS email, users.role AS role, users.created_at AS created_at, " +
			"COUNT(posts.id) AS total_articles, COUNT(CASE WHEN posts.is_published THEN 1 END) AS total_published").
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Group("users.id, users.username, users.email, users.role, users.created_at").
		Order("users.id").
		Scan(&rows).Error
	return rows, err
}
