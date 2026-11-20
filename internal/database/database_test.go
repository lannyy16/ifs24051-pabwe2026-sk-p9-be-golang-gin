package database

import (
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
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

func TestConnectChoosesDialect(t *testing.T) {
	db, err := Connect(&config.Config{DBConnection: "sqlite", DBDatabase: ":memory:"})
	require.NoError(t, err)
	assert.NotNil(t, db)

	_, err = Connect(&config.Config{DBConnection: "pgsql", DBHost: "127.0.0.1", DBPort: "1", DBSSLMode: "disable"})
	assert.Error(t, err)
}

func TestEnsureSchema(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}
	db := memDB(t)

	require.NoError(t, EnsureSchema(db, cfg)) // migrate + seed
	var posts int64
	db.Model(&post.Post{}).Count(&posts)
	assert.EqualValues(t, 2, posts)
	assert.True(t, schemaReady(db, cfg))
	require.NoError(t, EnsureSchema(db, cfg)) // sudah siap: tidak melakukan apa pun

	// tabel ada tetapi akun bawaan belum lengkap => belum siap
	db2 := memDB(t)
	require.NoError(t, Migrate(db2))
	assert.False(t, schemaReady(db2, cfg))

	// migrasi gagal
	db3 := memDB(t)
	closeDB(t, db3)
	assert.Error(t, EnsureSchema(db3, cfg))
}

func TestSeedPostErrors(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "admin"}

	// tabel posts tidak ada => Count gagal
	db := memDB(t)
	require.NoError(t, db.AutoMigrate(&auth.User{}))
	assert.Error(t, Seed(db, cfg))

	// pembuatan artikel gagal
	db = memDB(t)
	require.NoError(t, Migrate(db))
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_posts", func(tx *gorm.DB) {
		if tx.Statement.Table == "posts" {
			_ = tx.AddError(errors.New("boom"))
		}
	}))
	assert.ErrorContains(t, Seed(db, cfg), "seed artikel")
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

func TestSeedAddsModuleAdminAndFixesRole(t *testing.T) {
	db := memDB(t)
	require.NoError(t, Migrate(db))
	// akun "admin" lama yang keliru bertipe member
	require.NoError(t, db.Create(&auth.User{Username: "admin", Email: "admin@delcom.org", Password: "x", Role: "member"}).Error)

	cfg := &config.Config{AdminUsername: "root", AdminPassword: "rootpass"}
	assert.False(t, schemaReady(db, cfg))
	require.NoError(t, Seed(db, cfg))
	assert.True(t, schemaReady(db, cfg))

	var admin, root auth.User
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	require.NoError(t, db.Where("username = ?", "root").First(&root).Error)
	assert.Equal(t, auth.RoleAdmin, admin.Role)
	assert.Equal(t, auth.RoleAdmin, root.Role)

	// kegagalan memperbarui role
	db2 := memDB(t)
	require.NoError(t, Migrate(db2))
	require.NoError(t, db2.Create(&auth.User{Username: "admin", Email: "admin@delcom.org", Password: "x", Role: "member"}).Error)
	require.NoError(t, db2.Callback().Update().Before("gorm:update").Register("fail_update", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("boom"))
	}))
	assert.Error(t, Seed(db2, &config.Config{AdminUsername: "admin", AdminPassword: "admin"}))
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
