// Package database mengelola koneksi, migrasi, dan seeder basis data.
package database

import (
	"fmt"
	"strings"
	"time"

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

// Connect memilih SQLite (DB_CONNECTION=sqlite) atau PostgreSQL (default).
func Connect(cfg *config.Config) (*gorm.DB, error) {
	if cfg.DBConnection == "sqlite" {
		return Open(sqlite.Open(cfg.DBDatabase))
	}
	return ConnectPostgres(cfg)
}

// ConnectPostgres menghubungkan aplikasi ke PostgreSQL. PreferSimpleProtocol membuatnya
// kompatibel dengan connection pooler (Supabase/Neon/PgBouncer) pada lingkungan serverless.
func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	return Open(postgres.New(postgres.Config{DSN: cfg.DSN(), PreferSimpleProtocol: true}))
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

// seedUsers: akun dari .env, akun wajib modul (admin/admin), dan member/member.
func seedUsers(cfg *config.Config) []seedUser {
	adminUser := strings.TrimSpace(cfg.AdminUsername)
	if adminUser == "" {
		adminUser = "admin"
	}
	adminEmail := strings.ToLower(adminUser)
	if !strings.Contains(adminEmail, "@") {
		adminEmail += "@delcom.org"
	}
	users := []seedUser{
		{adminUser, adminEmail, cfg.AdminPassword, auth.RoleAdmin},
		{"member", "member@delcom.org", "member", auth.RoleMember},
	}
	if !strings.EqualFold(adminUser, "admin") {
		users = append(users, seedUser{"admin", "admin@delcom.org", "admin", auth.RoleAdmin})
	}
	return users
}

// schemaReady bernilai true bila tabel ada dan setiap akun bawaan sudah ada dengan role
// (serta password untuk akun admin) yang sesuai.
func schemaReady(db *gorm.DB, cfg *config.Config) bool {
	m := db.Migrator()
	if !m.HasTable(&auth.User{}) || !m.HasTable(&post.Post{}) {
		return false
	}
	for _, s := range seedUsers(cfg) {
		var u auth.User
		err := db.Where("LOWER(username) = LOWER(?)", s.username).Order("id").Limit(1).Find(&u).Error
		if err != nil || u.ID == 0 || !strings.EqualFold(u.Role, s.role) {
			return false
		}
		if s.role == auth.RoleAdmin && bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(s.password)) != nil {
			return false
		}
	}
	return true
}

// EnsureSchema menjalankan Migrate + Seed hanya bila belum pernah dilakukan, sehingga
// cold start serverless cepat dan beberapa instance paralel tidak saling bentrok.
func EnsureSchema(db *gorm.DB, cfg *config.Config) error {
	if schemaReady(db, cfg) {
		return nil
	}
	if err := Migrate(db); err != nil {
		return err
	}
	return Seed(db, cfg)
}

// Seed menyuntikkan akun admin (dari .env), member, dan dua artikel contoh bila belum ada.
func Seed(db *gorm.DB, cfg *config.Config) error {
	ids := map[string]uint{}
	for _, s := range seedUsers(cfg) {
		var existing auth.User
		err := db.Where("LOWER(username) = LOWER(?)", s.username).Order("id").Limit(1).Find(&existing).Error
		if err != nil {
			return err
		}
		if existing.ID != 0 {
			// Pastikan role (dan password untuk admin) akun bawaan selalu sesuai konfigurasi.
			updates := map[string]any{}
			if existing.Role != s.role {
				updates["role"] = s.role
			}
			if s.role == auth.RoleAdmin && bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte(s.password)) != nil {
				hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
				if err != nil {
					return err
				}
				updates["password"] = string(hash)
			}
			if len(updates) > 0 {
				if err := db.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}
			ids[s.username] = existing.ID
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
		ids[s.username] = user.ID
	}
	return seedPosts(db, cfg, ids)
}

func seedPosts(db *gorm.DB, cfg *config.Config, ids map[string]uint) error {
	var count int64
	if err := db.Model(&post.Post{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	adminID := ids[strings.TrimSpace(cfg.AdminUsername)]
	if adminID == 0 {
		adminID = ids["admin"]
	}
	now := time.Now()
	samples := []post.Post{
		{
			UserID: ids["member"], Title: "Pengenalan Bahasa Pemrograman Golang Modern",
			Slug: "pengenalan-bahasa-pemrograman-golang-modern", Category: "Teknologi",
			Excerpt:     "Panduan memahami Golang dan Gin untuk membangun REST API yang cepat dan rapi.",
			Content:     "# Belajar Golang\n\nGolang adalah bahasa pemrograman open source buatan Google yang dirancang untuk performa tinggi, efisiensi memori, dan konkurensi yang mudah lewat goroutine.\n\n## Mengapa Gin?\n\nGin adalah web framework ringan dengan router cepat, middleware, dan binding JSON yang praktis.\n\n## Penutup\n\nMulailah dari proyek kecil, tulis pengujian sejak awal, dan biarkan arsitektur tumbuh secara bertahap.",
			IsPublished: true, PublishedAt: &now,
		},
		{
			UserID: adminID, Title: "Penerapan AI dalam Kurikulum Pendidikan 2026",
			Slug: "penerapan-ai-dalam-kurikulum-pendidikan-2026", Category: "Pendidikan",
			Excerpt:     "Bagaimana kecerdasan buatan mengubah cara belajar dan mengajar di perguruan tinggi.",
			Content:     "# Pendidikan Modern Berbasis AI\n\nKecerdasan buatan membantu proses belajar mengajar menjadi lebih interaktif dan adaptif.\n\n## Peluang\n\n- Umpan balik instan untuk mahasiswa\n- Materi yang menyesuaikan kecepatan belajar\n- Asisten penulisan dan penelitian\n\n## Tantangan\n\nIntegritas akademik dan literasi digital tetap menjadi fondasi yang harus dijaga.",
			IsPublished: true, PublishedAt: &now,
		},
	}
	for i := range samples {
		if err := db.Create(&samples[i]).Error; err != nil {
			return fmt.Errorf("seed artikel: %w", err)
		}
	}
	return nil
}
