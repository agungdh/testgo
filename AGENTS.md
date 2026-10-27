# AGENTS.md

Panduan untuk agent AI yang bekerja di repo ini. Baca dulu sebelum mengubah kode.

## Ringkasan Proyek

`testgo` adalah REST API backend (CRUD produk) yang dipakai sebagai basis
layanan Go. Fokus repo ini adalah backend saja: tidak ada frontend, tidak ada
template HTML, dan semua response berupa JSON.

## Stack

| Komponen | Pilihan | Catatan |
|---|---|---|
| Bahasa | Go 1.27.3 | lihat `go.mod`, jangan turunkan versinya tanpa alasan |
| Router | `go-chi/chi/v5` | sub-router per resource lewat `Routes()` |
| ORM | `uptrace/bun` v1.3 | dialect `pgdialect`, driver `pgdriver` (bukan `lib/pq`) |
| Database | PostgreSQL 18 | via `docker compose` |
| Migrasi | `pressly/goose/v3` | SQL, di-embed ke binary |
| Config | `joho/godotenv` | `.env` opsional, env proses menang |
| ID | `google/uuid` | PK produk bertipe `uuid` |
| Logging | `log/slog` | stdlib, structured key-value |
| Cache | Valkey 9.1 (compose) | **belum dipakai kode**; belum ada client di `go.mod` |
| Deploy | Docker distroless + GitHub Actions | binary di-build di CI, bukan di Dockerfile |

## Perintah Penting

```bash
make dev                              # jalankan app (auto-load .env, auto-migrate)
make test                             # go test ./...
go vet ./...                          # jalankan sebelum selesai
go build ./...                        # pastikan semua package compile

make migrate-up                       # terapkan migrasi
make migrate-down                     # rollback SATU migrasi terakhir
make migrate-status                   # status migrasi
make migrate-create name=nama_baru    # buat file migrasi goose baru

docker compose up -d postgres         # DB lokal saja
docker compose up -d                  # + valkey, adminer (8083), redis commander (8084)
make build                            # binary ke build/testgo
```

Tidak ada formatter/linter eksternal di repo ini. Gunakan `gofmt` (default
`go build`/editor) dan `go vet`.

## Struktur

```
main.go                                 # entrypoint: config -> db -> repo -> server + graceful shutdown
cmd/migrate/main.go                     # CLI goose: up | down | status | version
internal/config/config.go               # loader .env + helper getString/getInt/getBool/getDuration
internal/db/db.go                       # bun + pgdriver, set connection pool
internal/db/migrate.go                  # runner goose dengan //go:embed migrations/*.sql
internal/db/migrations/                 # migrasi SQL (format goose)
internal/models/product.go              # model bun + request DTO + validasi
internal/repository/product.go          # interface + implementasi in-memory (untuk test)
internal/repository/postgres_product.go # implementasi PostgreSQL (dipakai app)
internal/handler/product.go             # HTTP handler + helper JSON/error
internal/router/router.go               # routing, middleware, /health, 404/405 handler
internal/router/router_test.go          # test end-to-end HTTP pakai repo in-memory
```

Alur request: `router` -> `handler` -> `repository.ProductRepository` -> implementasi
`postgresProductRepository` atau `memoryProductRepository`.

## Konvensi Kode

- **Bahasa komentar, pesan error, dan pesan validasi: Indonesia.** Ikuti gaya yang
  sudah ada (`"gagal mengambil data produk"`, `"tidak boleh negatif"`). Jangan
  mencampur bahasa dalam satu file.
- Selalu bungkus error dengan konteks: `fmt.Errorf("set dialek goose: %w", err)`.
- Pakai `context.Context` sebagai parameter pertama di layer db/repository/handler.
- Handler **tidak** menulis SQL dan repository **tidak** menyentuh `http.ResponseWriter`.
- Semua akses DB lewat `*bun.DB`. Jangan menambah `database/sql` query mentah
  kecuali benar-benar perlu.
- Error yang bisa dipetakan ke HTTP memakai sentinel `repository.ErrNotFound`.
  Error lain di-log dengan `slog.Error` lalu jadi 500.
- Log memakai `slog` dengan pasangan key-value, bukan `fmt.Printf`.
- Response JSON selalu snake_case untuk field DB (`created_at`, `updated_at`).

### Format Response

```json
{ "data": { ... } }
{ "data": [ ... ], "total": 3 }
{ "error": { "message": "..." } }
{ "error": { "message": "validasi gagal", "fields": { "name": "wajib diisi" } } }
```

Status yang dipakai: `200` OK, `201` Created (+ header `Location`), `204` No Content,
`400` body JSON invalid, `404` tidak ditemukan, `405` method salah, `422` validasi gagal,
`500` error server.

## Database & Migrasi

- Migrasi **SQL**, bukan Go: `make migrate-create name=tambah_kategori`, lalu isi
  blok `-- +goose Up` dan `-- +goose Down`.
- File migrasi di-embed (`//go:embed`). Setelah membuat migrasi, binary harus
  di-build ulang agar migrasi ikut masuk.
- `DB_AUTO_MIGRATE=true` (default) menjalankan `up` saat app start. Set `false`
  di production dan jalankan `make migrate-up` terpisah.
- `down` hanya me-rollback satu migrasi terakhir. Jangan mengubah migrasi yang
  sudah pernah jalan di environment lain; buat migrasi baru.
- Kolom waktu pakai `timestamptz` + `DEFAULT now()`; validasi bisnis juga
  ditegakkan sebagai `CHECK` constraint (mis. `price >= 0`).
- Pertahankan pengaturan pool kecil di `internal/db/db.go` (target VPS 1 core /
  1 GB RAM). Jangan naikkan default tanpa alasan jelas.

## Konfigurasi

- Tambah variabel baru di tiga tempat: `internal/config/config.go`,
  `.env.example`, dan tabel variabel di `README.md`.
- `.env` **jangan pernah** di-commit (sudah ada di `.gitignore`). Jangan menulis
  kredensial nyata ke `.env.example`.
- Variabel `POSTGRES_*` dan `VALKEY_*` hanya untuk `docker compose`, bukan dibaca app.
- Env proses selalu menang atas `.env`, jadi jangan andalkan `.env` di production.

## Testing

- `go test ./...` harus lulus **tanpa** database. Test HTTP memakai
  `repository.NewMemoryProductRepository()`.
- Saat menambah endpoint, tambahkan test di `internal/router/router_test.go`
  mengikuti pola `doRequest` + alur CRUD.
- Jangan membuat test yang butuh PostgreSQL berjalan, kecuali diminta eksplisit
  dan diberi build tag/skip otomatis.

## Build & Deploy

- `.github/workflows/docker.yml` cross-compile per arch (amd64/arm64) di runner
  native, lalu push image + manifest multi-arch ke GHCR.
- `Dockerfile` hanya runtime: `gcr.io/distroless/static-debian13:nonroot`. Tidak ada
  shell, `curl`, atau package manager di dalam image — jangan menambah `RUN`.
- Build pakai `CGO_ENABLED=0`. Jangan menambah dependency yang butuh cgo.
- Default `GOAMD64=v2` adalah **minimum requirement** CPU. Turunkan ke `v1` hanya
  jika target memang CPU lama.
- Health check dilakukan dari luar container (LB/probe), bukan di dalam image.

## Aturan untuk Agent

1. Baca file terkait dulu; jangan menebak nama package atau signature.
2. Perubahan sekecil mungkin dan konsisten dengan pola yang sudah ada. Jangan
   refactor struktur di luar permintaan.
3. Jangan menambah dependency baru bila stdlib sudah cukup. Jika perlu, jalankan
   `go get` dan jelaskan alasannya.
4. Jangan menghapus atau mengubah `internal/repository/product.go` (repo in-memory)
   karena dipakai test.
5. Jangan commit `.env`, `build/`, atau `dist/`.
6. Sebelum menyatakan selesai: `gofmt`, `go vet ./...`, `go build ./...`,
   `go test ./...`.
7. Jika mengubah endpoint, perbarui tabel Endpoint + contoh `curl` di `README.md`.
8. Jangan menyentuh riwayat git (`reset --hard`, `push --force`, amend commit
   lama) kecuali diminta eksplisit.

## Checklist Menambah Resource Baru

1. `internal/db/migrations/<timestamp>_create_<tabel>.sql` (up + down).
2. `internal/models/<resource>.go` — model bun + `Create...Request`/`Update...Request`
   + `Validate()`.
3. `internal/repository/<resource>.go` — interface + implementasi in-memory.
4. `internal/repository/postgres_<resource>.go` — implementasi bun.
5. `internal/handler/<resource>.go` — handler + `Routes()`.
6. `internal/router/router.go` — mount route baru.
7. `internal/router/router_test.go` — test alur CRUD.
8. `README.md` — struktur, endpoint, contoh `curl`.
9. `main.go` — wire repository baru.
