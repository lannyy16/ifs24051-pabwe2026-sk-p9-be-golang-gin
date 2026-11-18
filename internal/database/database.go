
package database

import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/features/auth"
    "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/features/post"
)

func Connect(dsn string) (*gorm.DB, error) {
    return gorm.Open(
        postgres.Open(dsn),
        &gorm.Config{},
    )
}

func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &auth.User{},
        &post.Post{},
    )
}
