package app

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func Run() {
	// Membaca konfigurasi aplikasi dari environment.
	cfg := config.Load()

	// Menghubungkan aplikasi ke PostgreSQL.
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("gagal terhubung ke PostgreSQL: %v", err)
	}

	// Menjalankan migrasi database.
	if err := database.Migrate(db); err != nil {
		log.Fatalf("gagal migrasi basis data: %v", err)
	}

	// Memasukkan data awal jika diperlukan.
	if err := database.Seed(db, cfg); err != nil {
		log.Fatalf("gagal seeding akun bawaan: %v", err)
	}

	// Membuat router Gin.
	r := server.NewRouter(cfg, db)

	// Menampilkan alamat Swagger.
	log.Printf(
		"Swagger UI: http://localhost:%s/swagger/index.html",
		cfg.AppPort,
	)

	// Menjalankan server.
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}
