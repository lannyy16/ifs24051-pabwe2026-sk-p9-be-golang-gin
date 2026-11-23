// Command server menjalankan REST API Delcom Blog (Golang Gin).
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
