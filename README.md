# ifs24051-pabwe2026-sk-p9-be-golang-gin

Backend **Delcom Blog** — REST API blog multi-tier + **AI Blog Generator** terintegrasi Search Engine.
Golang 1.27 · Gin · GORM · PostgreSQL 17 · JWT · Swagger UI. Port **8081**.

## Persiapan

```bash
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
cp .env.example .env          # sesuaikan bila perlu
go mod tidy                   # WAJIB sekali: menghasilkan go.sum & melengkapi go.mod
```

> Commit `go.mod` dan `go.sum` hasil `go mod tidy` ke repositori agar tahap
> *Install Dependencies* (`go mod download && go mod verify`) pada Jenkins berhasil.

## Menjalankan

```bash
go run cmd/server/main.go
# Swagger UI : http://localhost:8081/swagger/index.html
```

Akun bawaan (seeder): `admin / admin` (admin) dan `member / member` (member).

## Pengujian (100% statement coverage)

```bash
go test -v -cover ./internal/...
```

## Build biner

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server
```

## Endpoint

| Method | Endpoint | Akses |
|---|---|---|
| GET | `/swagger/*any`, `/api.json` | Publik |
| GET | `/api/public/categories` | Publik |
| GET | `/api/public/posts?q=&category=` | Publik |
| GET | `/api/public/posts/:slug` | Publik |
| POST | `/api/auth/register`, `/api/auth/login` | Publik |
| POST | `/api/auth/logout` | Login |
| GET/PUT | `/api/auth/profile` | Login |
| GET/POST | `/api/posts` | Penulis |
| GET | `/api/posts/stats` | Penulis |
| POST | `/api/posts/ai-generate` | Penulis |
| GET/PUT/DELETE | `/api/posts/:id` | Penulis |
| PATCH | `/api/posts/:id/toggle-publish` | Penulis |
| GET | `/api/admin/users`, `/api/admin/posts`, `/api/admin/users/:id/posts` | Admin |

Format respons: `{ "status": "success|fail|error", "message": "...", "data": ... }`.

## Kontrak Search Engine (asumsi)

`GET {SEARCH_ENGINE_ENDPOINT}?q=<topik>&limit=5` dengan header `Authorization: Bearer <OPENAI_KEY>`,
balasan `{"results":[{"title","url","snippet"}]}` atau `{"data":{"results":[...]}}`.
Jika penelusuran gagal, artikel tetap dibuat tanpa sumber rujukan. Sesuaikan `internal/features/ai/search.go`
bila kontrak endpoint Delcom berbeda.

## CI/CD

`Jenkinsfile` mengikuti pipeline: Checkout → Install → Test → Build → Trivy → Package → Publish → Deploy.
Aturan `.gitignore` memakai awalan `/` (`/server`, `/main`, `/app`) agar `internal/server/` tidak terabaikan.


## Port deployment Delcom
Aplikasi menggunakan environment `PORT` bila disediakan platform; jika tidak, gunakan `APP_PORT` (default 8081). Jangan commit file `.env` atau secret produksi.
