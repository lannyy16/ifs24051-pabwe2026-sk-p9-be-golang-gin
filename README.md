# ifs24051-pabwe2026-sk-p9-be-golang-gin

Backend **Delcom Blog**: REST API blog multi-tier + **AI Blog Generator** terintegrasi Search Engine.
Golang · Gin · GORM · PostgreSQL · JWT · Swagger UI. Port lokal **8081**.

## Menjalankan lokal

```bash
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
cp .env.example .env          # sesuaikan nilainya
go run cmd/server/main.go
# Swagger UI : http://localhost:8081/swagger/index.html
```

Tabel dibuat otomatis (AutoMigrate). Seeder membuat `admin / admin` (admin), `member / member`
(penulis), dan dua artikel contoh. `go.mod` + `go.sum` sudah lengkap, tidak perlu `go mod tidy`.

Tanpa PostgreSQL, uji cepat dengan SQLite: `DB_CONNECTION=sqlite DB_DATABASE=blog.db`.

## Deploy ke Vercel (serverless)

1. Buat database PostgreSQL gratis di **Neon** atau **Supabase**.
2. Push repositori ini ke GitHub, lalu **Import** ke Vercel. Nama project: `ifs24051-pabwe2026-sk-p9-be-golang-gin`.
3. Isi *Environment Variables* di Vercel: `APP_KEY`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `DB_HOST`, `DB_PORT`,
   `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `DB_SSLMODE=require`, `SEARCH_ENGINE_ENDPOINT`,
   `OPENAI_ENDPOINT`, `OPENAI_KEY`, `OPENAI_MODEL`.
4. Deploy. Cek `https://<project>.vercel.app/` (respons "Delcom Blog API berjalan") dan `/swagger/index.html`.

Cara kerjanya: `vercel.json` meneruskan semua request ke `api/index.go`; `internal/server/pathfix.go`
memulihkan path asli; `database.EnsureSchema` hanya migrate/seed saat tabel belum ada (cold start cepat).

## Pengujian (100% statement coverage)

```bash
go test -v -cover ./...
```

## Endpoint

| Method | Endpoint | Akses |
|---|---|---|
| GET | `/`, `/swagger/*any`, `/api.json` | Publik |
| GET | `/api/public/categories`, `/api/public/posts?q=&category=`, `/api/public/posts/:slug` | Publik |
| POST | `/api/auth/register`, `/api/auth/login` | Publik |
| POST/GET/PUT | `/api/auth/logout`, `/api/auth/profile` | Login |
| GET/POST | `/api/posts`, `/api/posts/stats` | Penulis |
| POST | `/api/posts/ai-generate` | Penulis |
| GET/PUT/DELETE | `/api/posts/:id` | Penulis |
| PATCH | `/api/posts/:id/toggle-publish` | Penulis |
| GET | `/api/admin/users`, `/api/admin/posts`, `/api/admin/users/:id/posts` | Admin |

Format respons: `{ "status": "success|fail|error", "message": "...", "data": ... }`.

## AI Generator

`GET {SEARCH_ENGINE_ENDPOINT}?q=<topik>` (tanpa header Authorization). Balasan boleh berupa `[...]`,
`{"data":[...]}`, atau `{"results":[...]}`. `{OPENAI_ENDPOINT}/chat/completions` dipanggil dengan Bearer `OPENAI_KEY`.
Bila model membalas teks Markdown biasa (bukan JSON), seluruh balasan dipakai sebagai isi artikel.

## CI/CD (Jenkins)

`Jenkinsfile`: Checkout → Install → Test → Build (`./cmd/server`) → Trivy → Package → Publish → Deploy.
