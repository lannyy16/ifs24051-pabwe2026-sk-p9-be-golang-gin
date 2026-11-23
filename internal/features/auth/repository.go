package auth

import (
	"strings"

	"gorm.io/gorm"
)

// Repository mengabstraksi akses data pengguna.
type Repository interface {
	Create(u *User) error
	Update(u *User) error
	FindByID(id uint) (*User, error)
	FindByIdentity(identity string) (*User, error)
	UsernameExists(username string, excludeID uint) (bool, error)
	EmailExists(email string, excludeID uint) (bool, error)
}

type gormRepository struct{ db *gorm.DB }

// NewRepository membuat Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(u *User) error { return r.db.Create(u).Error }

func (r *gormRepository) Update(u *User) error { return r.db.Save(u).Error }

func (r *gormRepository) FindByID(id uint) (*User, error) {
	var u User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) FindByIdentity(identity string) (*User, error) {
	var users []User
	if err := r.db.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)", identity, identity).Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	for i := range users {
		if users[i].Username == identity {
			return &users[i], nil
		}
	}
	lower := strings.ToLower(identity)
	for i := range users {
		if strings.ToLower(users[i].Email) == lower {
			return &users[i], nil
		}
	}
	return &users[0], nil
}

func (r *gormRepository) UsernameExists(username string, excludeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&User{}).Where("LOWER(username) = LOWER(?) AND id <> ?", username, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) EmailExists(email string, excludeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&User{}).Where("LOWER(email) = LOWER(?) AND id <> ?", email, excludeID).Count(&n).Error
	return n > 0, err
}
