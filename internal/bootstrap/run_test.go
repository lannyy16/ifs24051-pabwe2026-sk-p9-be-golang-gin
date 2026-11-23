package bootstrap

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"app/internal/config"
)

func TestRunStartsServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", "file:bootstrap_ok?mode=memory&cache=shared")
	t.Setenv("PORT", "4321")

	var gotAddr string
	orig := listen
	listen = func(_ *gin.Engine, addr string) error { gotAddr = addr; return nil }
	defer func() { listen = orig }()

	assert.NoError(t, Run())
	assert.Equal(t, ":4321", gotAddr)
}

func TestRunPropagatesServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", "file:bootstrap_err?mode=memory&cache=shared")

	orig := listen
	listen = func(*gin.Engine, string) error { return errors.New("port dipakai") }
	defer func() { listen = orig }()

	assert.ErrorContains(t, Run(), "port dipakai")
}

func TestRunDatabaseFailures(t *testing.T) {
	orig, origEnsure := listen, ensure
	listen = func(*gin.Engine, string) error {
		t.Fatal("server tidak boleh berjalan saat database gagal")
		return nil
	}
	defer func() { listen, ensure = orig, origEnsure }()

	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", "/direktori-tidak-ada/x/y.db")
	assert.ErrorContains(t, Run(), "gagal terhubung ke database")

	t.Setenv("DB_DATABASE", "file:bootstrap_schema?mode=memory&cache=shared")
	ensure = func(*gorm.DB, *config.Config) error { return errors.New("migrate gagal") }
	assert.ErrorContains(t, Run(), "gagal menyiapkan skema database")
}

func TestDefaultListenReturnsErrorForInvalidAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Error(t, listen(gin.New(), ":-1"))
}
