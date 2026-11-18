# Backend Delcom Blog (P9)
Go/Gin REST API, GORM/PostgreSQL, JWT, Clean Architecture Feature-First.

1. Buat database PostgreSQL `delcom_blog`.
2. Salin `.env.example` menjadi `.env`, sesuaikan `DATABASE_URL` dan `JWT_SECRET`.
3. `go mod tidy`
4. `go run ./cmd/server`

API: `http://localhost:8081`; Swagger placeholder UI: `/swagger/index.html`.
Tes: `go test -v -cover ./internal/...`.
Registrasi membuat role `author`; ubah role akun admin melalui PostgreSQL. AI membutuhkan `OPENAI_API_KEY`. Integrasi search engine perlu provider endpoint yang sesuai dengan format JSON yang disepakati; contoh ini mengembalikan sumber kosong dan tidak mengarang sumber. `logout` stateless sehingga token dihapus oleh frontend.