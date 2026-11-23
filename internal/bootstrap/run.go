// Package bootstrap menyiapkan dan menjalankan server HTTP (dipakai main.go di root dan cmd/server).
package bootstrap

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

// Dapat diganti pada pengujian.
var (
	listen = func(r *gin.Engine, addr string) error { return r.Run(addr) }
	ensure = database.EnsureSchema
)

// Run memuat konfigurasi, menghubungkan database, menyiapkan skema, lalu menjalankan server.
// Fungsi ini blok sampai server berhenti.
func Run() error {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke database: %w", err)
	}
	if err := ensure(db, cfg); err != nil {
		return fmt.Errorf("gagal menyiapkan skema database: %w", err)
	}

	r := server.NewRouter(cfg, db)
	log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", cfg.AppPort)
	return listen(r, cfg.Addr())
}
