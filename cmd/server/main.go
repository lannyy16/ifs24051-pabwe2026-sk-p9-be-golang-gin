package main

import (
 "log"
 "github.com/joho/godotenv"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/config"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/database"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/server"
)

func main() {
 _ = godotenv.Load()
 cfg := config.Load()
 db, err := database.Connect(cfg.DatabaseURL)
 if err != nil { log.Fatal(err) }
 if err = database.Migrate(db); err != nil { log.Fatal(err) }
 if err = server.NewRouter(db, cfg).Run(":"+cfg.Port); err != nil { log.Fatal(err) }
}