// Command app menjalankan REST API Delcom Blog (Golang Gin).
// Berkas ini ada di root agar `go build -o server .` (dipakai platform deploy) berhasil;
// cmd/server/main.go berisi entry point yang sama.
package main

import (
	"log"

	"app/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
