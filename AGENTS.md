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
| Database | PostgreSQL 18 | **satu-satunya database** yang dipakai; lihat aturan skema di bawah |
| Migrasi | `pressly/goose/v3` | SQL, di-embed ke binary |
| Config | `joho/godotenv` | `.env` opsional, env proses menang |
| ID | `google/uuid` | UUID v4 untuk identifier publik (lihat aturan skema) |
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
- Nama field JSON `snake_case`. Identifier publik selalu `uuid` / `*_uuid`
  (lihat [Skema Tabel & Penamaan](#skema-tabel--penamaan)).

### Yang Boleh Dilihat FE

FE **tidak boleh** tahu id internal maupun kolom audit. Aturan keras:

- Jangan pernah menambahkan `json:"..."` untuk `id` atau kolom yang berakhiran `_id`.
  Pakai `json:"-"`.
- Jangan pernah mengirim `created_by`, `created_at`, `updated_by`, `updated_at`,
  `deleted_by`, `deleted_at` ke response.
- Yang di-expose hanya `uuid` (record itu sendiri) dan `<relasi>_uuid` (foreign key).
- UUID **selalu** dibuat server (`uuid.NewString()`), jangan pernah diterima dari body
  request dan jangan pernah dipakai sebagai input `WHERE` untuk id internal.

### Format Response

```json
{ "data": { "uuid": "...", "name": "...", "price": 45000 } }
{ "data": [ { "uuid": "...", "category_uuid": "..." } ], "next_cursor": "...", "has_more": true }
{ "error": { "message": "..." } }
{ "error": { "message": "validasi gagal", "fields": { "name": "wajib diisi" } } }
```

Status yang dipakai: `200` OK, `201` Created (+ header `Location`), `204` No Content,
`400` body JSON invalid, `404` tidak ditemukan, `405` method salah, `422` validasi gagal,
`500` error server.

## Skema Tabel & Penamaan

PostgreSQL adalah satu-satunya database. Tidak ada MySQL, SQLite, atau ORM lain.
**Setiap tabel** wajib mengikuti bentuk di bawah, tanpa terkecuali.

```sql
CREATE TABLE IF NOT EXISTS products (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid        uuid        NOT NULL,

    name        text        NOT NULL,
    price       numeric(12, 2) NOT NULL DEFAULT 0,
    category_id bigint      REFERENCES categories (id),

    created_by  bigint,
    created_at  timestamptz,
    updated_by  bigint,
    updated_at  timestamptz,
    deleted_by  bigint,
    deleted_at  timestamptz
);
```

### Kolom wajib

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | `bigint GENERATED ALWAYS AS IDENTITY` | PK internal. **Tidak pernah** di-expose |
| `uuid` | `uuid NOT NULL` | UUID v4, identifier publik. Tanpa index tambahan |
| `created_by` | `bigint` | nullable, **tanpa FK** |
| `created_at` | `timestamptz` | nullable |
| `updated_by` | `bigint` | nullable, **tanpa FK** |
| `updated_at` | `timestamptz` | nullable |
| `deleted_by` | `bigint` | nullable, **tanpa FK** |
| `deleted_at` | `timestamptz` | nullable, penanda soft delete |

Catatan penting:

- Tipe `*_by` mengikuti tipe `id` internal, yaitu `bigint` — **bukan** uuid. Isinya
  id internal aktor, hanya untuk catatan/audit di DB.
- `*_by` sengaja **tanpa foreign key**. Jangan menambahkan `REFERENCES` ke kolom audit.
- Semua kolom audit **nullable** di level DB. Aplikasi tetap wajib mengisi
  `created_by`/`created_at` saat insert dan `updated_by`/`updated_at` saat update.
- Semua `*_at` memakai `timestamptz`, tidak pernah `timestamp` atau `timestamp without
  time zone`.
- **Jangan menambah index kecuali memang wajib**: primary key, unique constraint, dan
  foreign key. Index untuk pencarian, filter, atau sort **tidak** dibuat di awal —
  optimasi dilakukan nanti setelah ada bukti query lambat, bukan spekulasi.
  `uuid` dan `deleted_at` tidak di-index secara default.
- Keunikan `uuid` dijamin oleh generator v4 di aplikasi, bukan oleh DB. Jangan
  menambah `UNIQUE` atau index lain tanpa persetujuan.
- `id` **bukan** autoincrement manual: jangan mengirim nilai `id` dari Go, biarkan
  `GENERATED ALWAYS AS IDENTITY` yang mengisi.

### Penamaan yang di-expose

| Konsep | Kolom DB | Field JSON |
|---|---|---|
| Primary key | `id` (bigint) | tidak ada |
| Identifier publik | `uuid` | `uuid` |
| Foreign key | `category_id` | `category_uuid` |

- Relasi di DB tetap memakai `*_id` + FK ke `id` internal.
- Di level API, seluruh identifier menjadi `uuid` atau berakhiran `_uuid`. Tidak ada
  field JSON yang berakhiran `_id`.
- Handler/repository yang menerima `category_uuid` harus me-resolve-nya ke
  `category_id` internal sebelum query.

### Soft delete

- Hapus data = `UPDATE ... SET deleted_by = ?, deleted_at = now()`. Jangan `DELETE`.
- `deleted_at`/`deleted_by` nullable by design: `NULL` = baris aktif, terisi = terhapus.
  Kolom ini **tidak pernah** masuk `ORDER BY`; semua query baca sudah menyaring
  `deleted_at IS NULL`, jadi baris yang tersisa selalu aktif.
- **Semua** query baca wajib menyaring `deleted_at IS NULL`.
- `GetByUUID` mengembalikan `ErrNotFound` bila baris sudah ter-soft-delete.

## List: Cursor Pagination, Filter, Sort

Semua endpoint list memakai **cursor pagination**, bukan offset. Tidak ada
`page`, `offset`, atau `total`.

```
GET /products?after=<uuid>&limit=20&sort=-price&name=kopi
```

| Param | Default | Aturan |
|---|---|---|
| `after` | kosong | `uuid` baris terakhir halaman sebelumnya. Kosong = halaman pertama |
| `limit` | `20` | 1–100. Di luar rentang → `422` |
| `sort` | `id DESC` | satu kolom bisnis, prefix `-` untuk DESC |
| filter | – | whitelist per resource, mis. `name`, `stock_min` |

Aturan:

- Cursor adalah **uuid baris terakhir**, bukan id internal. `after` tidak pernah
  membawa `id`, offset, atau nomor halaman.
- Kalau `after` bukan uuid valid, atau barisnya tidak ada / sudah ter-soft-delete →
  `400` `{"error":{"message":"cursor tidak valid"}}`. Jangan bocorkan apakah
  barisnya pernah ada.
- Query mengambil `limit+1` baris untuk menentukan `has_more`. `next_cursor` diisi
  `uuid` baris ke-`limit` hanya bila `has_more = true`; selain itu `null`.
- `total` tidak dikirim. Kalau butuh hitungan, buat endpoint terpisah; jangan
  menambahkannya ke response list.
- Jangan pakai `OFFSET`: lambat di tabel besar dan hasil bergeser saat ada insert.

### Bentuk query

Urutan selalu diakhiri `id` sebagai tiebreaker (uuid v4 acak, tidak bisa dipakai
mengurutkan). Resolve cursor dulu, baru ambil halaman:

```sql
-- 1. resolve cursor: uuid -> sort value + id
SELECT id, name FROM products WHERE uuid = $1 AND deleted_at IS NULL;

-- 2. halaman berikutnya, sort default id DESC
SELECT * FROM products
WHERE deleted_at IS NULL AND id < $cursor_id
ORDER BY id DESC
LIMIT $limit + 1;
```

Untuk kolom sort lain, pakai row-value comparison agar tetap satu query:

```sql
SELECT * FROM products
WHERE deleted_at IS NULL
  AND (name, id) < ($cursor_name, $cursor_id)   -- DESC; ASC pakai >
ORDER BY name DESC, id DESC
LIMIT $limit + 1;
```

- Default `id DESC` dipilih justru karena `id` selalu terisi dan monotonik sehingga
  urutannya stabil antar-halaman. `uuid` v4 acak, tidak bisa dipakai mengurutkan.
- `deleted_at` nullable tidak ikut soal: ia tidak pernah jadi kolom sort, dan
  `deleted_at IS NULL` sudah difilter di semua query.
- Kolom audit (`*_by`, `*_at`) nullable dan tidak pernah masuk whitelist sort.
- Kalau ada kolom bisnis yang nullable, jangan masukkan ke whitelist sort: row
  comparison dengan NULL menghasilkan NULL (bukan true), sehingga barisnya hilang
  dari hasil.
- Jangan menambah index komposit `(kolom_sort, id)` selama belum ada bukti query
  lambat. Keyset pagination tetap benar tanpa index; optimasi ditunda.
- Sort hanya satu kolom + tiebreaker `id`. Jangan menerima multi-kolom.
- `id` boleh jadi default sort, tapi jangan masukkan `id` ke whitelist `sort` publik.

### Filter & sort wajib whitelist

- Jangan pernah menempelkan nama kolom dari query string langsung ke SQL. Petakan
  nama publik ke kolom yang diizinkan lewat map/kode.
- Parameter di luar whitelist → `400`, supaya kesalahan FE cepat ketahuan.
- Filter tidak boleh membuka kolom audit atau id internal. Filter relasi memakai
  `*_uuid`, contoh `?category_uuid=<uuid>`, lalu di-resolve ke `category_id`.
- Cursor tidak menyimpan sort/filter. FE wajib memakai `sort` dan filter yang sama
  selama menelusuri halaman; server tidak bisa mendeteksi perubahan di tengah jalan.

### Bentuk response list

```json
{
  "data": [ { "uuid": "..." } ],
  "next_cursor": "uuid-baris-terakhir",
  "has_more": true
}
```

## Database & Migrasi

- Migrasi **SQL**, bukan Go: `make migrate-create name=tambah_kategori`, lalu isi
  blok `-- +goose Up` dan `-- +goose Down`.
- File migrasi di-embed (`//go:embed`). Setelah membuat migrasi, binary harus
  di-build ulang agar migrasi ikut masuk.
- `DB_AUTO_MIGRATE=true` (default) menjalankan `up` saat app start. Set `false`
  di production dan jalankan `make migrate-up` terpisah.
- `down` hanya me-rollback satu migrasi terakhir. Jangan mengubah migrasi yang
  sudah pernah jalan di environment lain; buat migrasi baru.
- Validasi bisnis juga ditegakkan sebagai `CHECK` constraint (mis. `price >= 0`).
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
- Response API tidak boleh membocorkan id internal atau kolom audit. Saat menyentuh
  handler, pastikan test tetap mengecek bahwa field `id`, `*_id`, dan `*_by` tidak
  pernah muncul di body response.
- Untuk endpoint list, uji juga: halaman pertama, `has_more`, `next_cursor`, halaman
  kedua lewat `after`, `after` tidak valid → `400`, dan `limit` di luar rentang →
  `422`.

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
9. Setiap tabel baru **wajib** memakai pola `id` internal + `uuid` publik + kolom
   audit + soft delete. Jangan membuat tabel tanpa itu walau terlihat sederhana.
10. Jangan pernah mengekspos `id`, `*_id`, atau kolom audit ke JSON. Kalau ragu,
    default-nya `json:"-"`.
11. Endpoint list wajib cursor pagination (`?after=<uuid>`, default `id DESC`) dengan
    filter/sort whitelist. Jangan menambah `offset`, `page`, atau `total`.

## Checklist Menambah Resource Baru

1. Migrasi `internal/db/migrations/<timestamp>_create_<tabel>.sql`:
   `id` identity, `uuid`, kolom bisnis, enam kolom audit, plus blok
   `-- +goose Down`. Jangan menambah index selain yang wajib (PK/unique/FK).
2. `internal/models/<resource>.go` — model bun (`ID` dan kolom audit `json:"-"`,
   `UUID` → `json:"uuid"`) + `Create...Request`/`Update...Request` + `Validate()`.
3. `internal/repository/<resource>.go` — interface + implementasi in-memory
   (pakai `uuid.NewString()`, hormati `deleted_at`). Method list menerima params
   (`After`, `Limit`, `Sort`, filter) dan mengembalikan `Page` berisi data +
   `NextCursor` + `HasMore`.
4. `internal/repository/postgres_<resource>.go` — implementasi bun: lookup lewat
   `uuid`, filter `deleted_at IS NULL`, delete = soft delete, list memakai keyset
   `(kolom_sort, id)` + `LIMIT limit+1`.
5. `internal/handler/<resource>.go` — handler + `Routes()`, resolve `*_uuid` ke
   `*_id` internal, `created_by`/`updated_by` diisi dari konteks, parse
   `after`/`limit`/`sort` + validasi whitelist.
6. `internal/router/router.go` — mount route baru.
7. `internal/router/router_test.go` — test alur CRUD + pastikan tidak ada `id`/audit
   di response.
8. `main.go` — wire repository baru.
9. `README.md` — struktur, endpoint, contoh `curl` (pakai `uuid`).

## Status Penerapan

Tabel `products` dan endpoint CRUD-nya sudah mengikuti aturan di atas: `id`
internal + `uuid` publik, kolom audit, soft delete, cursor pagination, serta
filter/sort whitelist. Migrasi `create_products` ditulis dari awal saat tabel
di-reset, jadi tidak ada migrasi alter terpisah.

Pengecualian sementara: aplikasi belum punya autentikasi, sehingga
`created_by`, `updated_by`, dan `deleted_by` selalu `NULL`. `created_at`,
`updated_at`, dan `deleted_at` tetap diisi server. Saat auth ditambahkan, isi
kolom `*_by` dari aktor di konteks tanpa mengubah bentuk response.
