package database

import (
	"log"
	"strings"

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
	return gorm.Open(d, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// ConnectPostgres menghubungkan aplikasi ke PostgreSQL.
func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	return Open(postgres.Open(cfg.DSN()))
}

// ConnectSQLiteFile membuka SQLite berbasis berkas (atau ":memory:").
func ConnectSQLiteFile(path string) (*gorm.DB, error) {
	db, err := Open(sqlite.Open(path))
	if err == nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}
	return db, err
}

// ConnectSQLiteMemory membuka SQLite in-memory untuk unit test.
func ConnectSQLiteMemory() (*gorm.DB, error) {
	return ConnectSQLiteFile(":memory:")
}

var (
	connectPostgres = ConnectPostgres
	connectSQLite   = ConnectSQLiteFile
	connectMemory   = ConnectSQLiteMemory
)

// Connect memilih basis data: PostgreSQL bila tersedia. Jika PostgreSQL tidak
// dapat dihubungi (atau DB_CONNECTION=sqlite), aplikasi memakai SQLite agar
// server tetap hidup dan tidak mati saat start (penyebab 502 Bad Gateway).
func Connect(cfg *config.Config) (*gorm.DB, error) {
	if strings.EqualFold(cfg.DBConnection, "sqlite") {
		return connectSQLite(cfg.SQLitePath)
	}

	db, err := connectPostgres(cfg)
	if err == nil {
		return db, nil
	}

	log.Printf("PostgreSQL tidak dapat dihubungi (%v); memakai SQLite %q", err, cfg.SQLitePath)
	db, err = connectSQLite(cfg.SQLitePath)
	if err != nil {
		log.Printf("SQLite berkas gagal dibuka (%v); memakai SQLite in-memory", err)
		return connectMemory()
	}
	return db, nil
}

// prepare menjalankan migrasi lalu seeding akun bawaan.
func prepare(db *gorm.DB, cfg *config.Config) error {
	if err := Migrate(db); err != nil {
		return err
	}
	return Seed(db, cfg)
}

// Setup menghubungkan, memigrasi, dan men-seed basis data. Bila PostgreSQL
// terhubung tetapi migrasi/seed gagal (mis. skema lama yang bentrok), aplikasi
// beralih ke SQLite in-memory yang bersih agar server tetap hidup.
func Setup(cfg *config.Config) (*gorm.DB, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err = prepare(db, cfg); err == nil {
		return db, nil
	}

	log.Printf("migrasi/seed gagal (%v); memakai SQLite in-memory", err)
	mem, err := connectMemory()
	if err != nil {
		return nil, err
	}
	if err := prepare(mem, cfg); err != nil {
		return nil, err
	}
	return mem, nil
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

// Seed membuat akun admin dan member jika belum tersedia.
func Seed(db *gorm.DB, cfg *config.Config) error {
	seeds := []seedUser{
		{
			username: cfg.AdminUsername,
			email:    cfg.AdminUsername + "@delcom.org",
			password: cfg.AdminPassword,
			role:     auth.RoleAdmin,
		},
		{
			username: "member",
			email:    "member@delcom.org",
			password: "member",
			role:     auth.RoleMember,
		},
	}

	for i, s := range seeds {
		var user auth.User

		err := db.Where("username = ?", s.username).First(&user).Error

		if err == nil {
			// Perbaiki role akun admin jika sebelumnya salah.
			// Password akun yang sudah ada tidak diubah.
			if i == 0 && user.Role != auth.RoleAdmin {
				if err := db.Model(&user).
					Update("role", auth.RoleAdmin).Error; err != nil {
					return err
				}
			}

			continue
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword(
			[]byte(s.password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return err
		}

		user = auth.User{
			Username: s.username,
			Email:    s.email,
			Password: string(hash),
			Role:     s.role,
		}

		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	return nil
}
