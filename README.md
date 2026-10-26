# testgo

REST API CRUD produk sederhana menggunakan [chi](https://github.com/go-chi/chi) (router) dan penyimpanan in-memory (tanpa database).

## Menjalankan

```bash
go run .
```

Server berjalan di `http://localhost:8080` (ubah dengan env `PORT`).

## Perintah Make

```bash
make help         # daftar perintah
make dev          # jalankan untuk development (auto-reload jika air terpasang)
make build        # build binary lokal ke build/testgo
make build-prod   # build binary production (linux/amd64, statis, stripped)
make run          # build lalu jalankan binary
make docker-build # build image Docker (distroless)
make docker-run   # jalankan image Docker
make test         # jalankan test
make check        # fmt + vet + test
make clean        # hapus hasil build
```

Port bisa diubah saat menjalankan: `make dev PORT=3000`.

Auto-reload memakai [air](https://github.com/air-verse/air); jika belum ada, `make dev` otomatis memakai `go run` biasa. Install dengan `make install-air`.

## Docker

```bash
make docker-build                      # build image testgo:latest
docker run --rm -p 8080:8080 testgo:latest
```

Atau langsung dengan Docker:

```bash
docker build -t testgo:latest .
docker run --rm -p 8080:8080 -e PORT=8080 testgo:latest
```

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
