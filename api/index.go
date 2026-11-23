// Package handler adalah entry point fungsi serverless Vercel (/api/index).
package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

var (
	mu     sync.Mutex
	router http.Handler

	// Dapat diganti pada pengujian.
	connect = database.Connect
	ensure  = database.EnsureSchema
)

// initRouter menyiapkan koneksi DB, migrate/seed (bila perlu), dan router sekali per instance.
// Jika gagal, request berikutnya akan mencoba lagi.
func initRouter() (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()

	if router != nil {
		return router, nil
	}

	cfg := config.Load()
	db, err := connect(cfg)
	if err != nil {
		return nil, err
	}
	tunePool(db)
	if err := ensure(db, cfg); err != nil {
		return nil, err
	}

	router = server.RestoreOriginalPath(server.NewRouter(cfg, db))
	return router, nil
}

// tunePool menjaga jumlah koneksi per instance tetap kecil (banyak instance berjalan paralel).
func tunePool(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}
}

// Handler adalah fungsi yang dipanggil Vercel untuk setiap request.
func Handler(w http.ResponseWriter, r *http.Request) {
	rt, err := initRouter()
	if err != nil {
		log.Printf("init gagal: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","message":"Server gagal inisialisasi"}`))
		return
	}
	rt.ServeHTTP(w, r)
}
