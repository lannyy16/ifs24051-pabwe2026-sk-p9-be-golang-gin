package database

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/features/auth"
)

func memDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := ConnectSQLiteMemory()
	require.NoError(t, err)
	return db
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func TestConnectPostgresFailsWithoutServer(t *testing.T) {
	cfg := &config.Config{DBHost: "127.0.0.1", DBPort: "1", DBUsername: "x", DBPassword: "x", DBDatabase: "x", DBSSLMode: "disable"}
	_, err := ConnectPostgres(cfg)
	assert.Error(t, err)
}

func TestConnectSQLiteMemory(t *testing.T) {
	db := memDB(t)
	assert.NoError(t, Migrate(db))
}

func TestOpenError(t *testing.T) {
	_, err := Open(sqlite.Open("/nonexistent-dir/xx/yy.db"))
	assert.Error(t, err)
}

func TestMigrateError(t *testing.T) {
	db := memDB(t)
	closeDB(t, db)
	assert.Error(t, Migrate(db))
}

func TestSeedCreatesAndIsIdempotent(t *testing.T) {
	db := memDB(t)
	require.NoError(t, Migrate(db))
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}

	require.NoError(t, Seed(db, cfg))
	require.NoError(t, Seed(db, cfg))

	var users []auth.User
	require.NoError(t, db.Order("id").Find(&users).Error)
	require.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, auth.RoleAdmin, users[0].Role)
	assert.Equal(t, "member", users[1].Username)
	assert.Equal(t, auth.RoleMember, users[1].Role)
}

func TestSeedErrors(t *testing.T) {
	// Kegagalan Count (DB tertutup).
	db := memDB(t)
	require.NoError(t, Migrate(db))
	closeDB(t, db)
	assert.Error(t, Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: "admin"}))

	// Kegagalan bcrypt (password > 72 byte).
	db = memDB(t)
	require.NoError(t, Migrate(db))
	assert.Error(t, Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: strings.Repeat("x", 100)}))

	// Kegagalan Create (email sudah dipakai username lain).
	db = memDB(t)
	require.NoError(t, Migrate(db))
	require.NoError(t, db.Create(&auth.User{Username: "orang", Email: "admin@delcom.org", Password: "x", Role: "member"}).Error)
	assert.Error(t, Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: "admin"}))
}
