// Package database mengelola koneksi, migrasi, dan seeder basis data.
package database

import (
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
)

// Open membuka koneksi GORM dengan dialector yang diberikan.
func Open(d gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(d, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
}

// ConnectPostgres menghubungkan aplikasi ke PostgreSQL (runtime).
func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	return Open(postgres.Open(cfg.DSN()))
}

// ConnectSQLiteMemory membuka SQLite in-memory (dipakai unit test mandiri).
func ConnectSQLiteMemory() (*gorm.DB, error) {
	db, err := Open(sqlite.Open(":memory:"))
	if err == nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}
	return db, err
}

// Migrate menjalankan AutoMigrate untuk seluruh model.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&auth.User{}, &post.Post{})
}

type seedUser struct {
	username string
	email    string
	password string
	role     string
}

// Seed menyuntikkan akun admin (dari .env) dan member bawaan bila belum ada.
func Seed(db *gorm.DB, cfg *config.Config) error {
	seeds := []seedUser{
		{cfg.AdminUsername, cfg.AdminUsername + "@delcom.org", cfg.AdminPassword, auth.RoleAdmin},
		{"member", "member@delcom.org", "member", auth.RoleMember},
	}
	for _, s := range seeds {
		var count int64
		if err := db.Model(&auth.User{}).Where("username = ?", s.username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := auth.User{Username: s.username, Email: s.email, Password: string(hash), Role: s.role}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}
	return nil
}
