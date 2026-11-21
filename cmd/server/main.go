// Command server menjalankan REST API Delcom Blog (Golang Gin).
package main

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.Setup(cfg)
	if err != nil {
		log.Fatalf("gagal menyiapkan basis data: %v", err)
	}

	r := server.NewRouter(cfg, db)
	log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", cfg.AppPort)
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}
