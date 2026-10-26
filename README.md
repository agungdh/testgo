# testgo

REST API CRUD produk sederhana menggunakan [chi](https://github.com/go-chi/chi) (router) dan penyimpanan in-memory (tanpa database).

## Menjalankan

Semua konfigurasi dibaca dari file `.env`:

```bash
cp .env.example .env   # sekali saja, lalu sesuaikan bila perlu
go run .
```

Server berjalan di `http://localhost:8080`. Ubah port lewat `PORT` di `.env`,
lalu jalankan ulang aplikasi.

`.env` tidak ikut di-commit; template-nya ada di `.env.example`. Di production
yang tidak memakai `.env`, variabel tetap bisa dikirim lewat environment proses
dan nilainya menang atas isi `.env`.

| Variabel | Default | Keterangan |
|---|---|---|
| `APP_ENV` | `development` | mode aplikasi |
| `PORT` | `8080` | port HTTP server |
| `DATABASE_URL` | `postgres://admin:admin@localhost:5432/simonjp?sslmode=disable` | koneksi PostgreSQL |
| `DB_MAX_OPEN_CONNS` | `10` | maksimal koneksi terbuka |
| `DB_MAX_IDLE_CONNS` | `2` | maksimal koneksi idle |
| `DB_CONN_MAX_LIFETIME` | `5m` | umur maksimal koneksi |
| `DB_CONN_MAX_IDLE_TIME` | `1m` | umur maksimal koneksi idle |

Variabel `POSTGRES_*`, `VALKEY_*`, `ADMINER_PORT`, dan `REDIS_COMMANDER_*` hanya
dipakai oleh `docker compose`.

## Perintah Make

```bash
make help         # daftar perintah
make env          # buat .env dari .env.example jika belum ada
make dev          # jalankan aplikasi untuk development
make build        # build binary lokal ke build/testgo
make run          # build lalu jalankan binary
make test         # jalankan semua test
make clean        # hapus hasil build
```

Port diubah lewat `PORT` di `.env`, bukan lewat argumen `make`.

## Service lokal (Docker Compose)

```bash
docker compose up -d
```

Postgres, Valkey, Adminer, dan Redis Commander memakai nilai dari `.env`.
Port default: `5432` (Postgres), `6379` (Valkey), `8083` (Adminer), dan `8084`
(Redis Commander).

## Docker

```bash
docker build -t testgo:latest .
docker run --rm -p 8080:8080 testgo:latest
```

Image tidak membawa file `.env`. Tanpa konfigurasi tambahan aplikasi memakai
default `PORT=8080`; di production nilainya bisa diberikan lewat environment
container.

Image runtime-only (binary di-build di GitHub Actions, cross-compile per arsitektur):

- **runtime**: `gcr.io/distroless/static-debian13:nonroot`, hanya berisi binary (sekitar 18 MB), berjalan sebagai user non-root, tanpa shell/package manager. Build multi-arch (linux/amd64, linux/arm64) dilakukan di CI via matrix tanpa QEMU.

Karena distroless tidak punya shell maupun `curl`, health check sebaiknya dilakukan dari luar container (load balancer, Kubernetes probe, atau `docker inspect`).

### Level microarchitecture (`GOAMD64`)

Default build memakai `GOAMD64=v2` (butuh SSE4.2/POPCNT, tersedia di CPU x86-64 sejak sekitar 2008). Turunkan ke `v1` hanya jika target masih CPU generasi lama. Nilai ini adalah **minimum requirement**, bukan sekadar optimasi:

| Level | Butuh fitur CPU |
|-------|-----------------|
| `v1` | baseline x86-64 |
| `v2` | SSE3, SSE4.1, SSE4.2, POPCNT, CMPXCHG16B, LAHF/SAHF |
| `v3` | + AVX, AVX2, BMI1/BMI2, FMA, LZCNT, MOVBE |
| `v4` | + AVX-512F, AVX-512BW, AVX-512CD, AVX-512DQ, AVX-512VL |

Binary level tinggi **tidak** jalan di CPU level rendah (runtime akan berhenti dengan pesan `This program can only be run on AMD64 processors with vN microarchitecture support.`). Sebaliknya, binary level rendah tetap jalan di CPU level lebih tinggi.

```bash
make docker-build GOAMD64=v1         # paling kompatibel (baseline x86-64)
make docker-build GOAMD64=v3         # image untuk CPU minimum v3
make build-prod PROD_GOAMD64=v3      # binary lokal minimum v3
docker build --build-arg GOAMD64=v2 -t testgo:v2 .
```

## Struktur

```
main.go                          # entrypoint + graceful shutdown
internal/config/config.go        # loader .env + konfigurasi
internal/models/product.go       # model & validasi request
internal/repository/product.go   # interface + implementasi in-memory
internal/handler/product.go      # HTTP handler CRUD
internal/router/router.go        # routing & middleware
```

## Endpoint

| Method | Path            | Keterangan            |
|--------|-----------------|-----------------------|
| GET    | `/health`       | health check          |
| GET    | `/products`     | daftar produk         |
| POST   | `/products`     | buat produk           |
| GET    | `/products/{id}`| detail produk         |
| PUT    | `/products/{id}`| update penuh          |
| PATCH  | `/products/{id}`| update sebagian       |
| DELETE | `/products/{id}`| hapus produk          |

## Contoh

```bash
# Create
curl -X POST http://localhost:8080/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"Kopi Gayo","price":45000,"stock":10}'

# List
curl http://localhost:8080/products

# Get
curl http://localhost:8080/products/<id>

# Update (partial)
curl -X PATCH http://localhost:8080/products/<id> \
  -H 'Content-Type: application/json' \
  -d '{"stock":5}'

# Delete
curl -X DELETE http://localhost:8080/products/<id> -i
```

## Test

```bash
go test ./...
```

## Catatan

Data disimpan di memori, sehingga hilang saat server dimatikan. Untuk memakai database, cukup buat implementasi baru dari `repository.ProductRepository` lalu daftarkan di `main.go`.
