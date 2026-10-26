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
make test         # jalankan test
make check        # fmt + vet + test
make clean        # hapus hasil build
```

Port bisa diubah saat menjalankan: `make dev PORT=3000`.

Auto-reload memakai [air](https://github.com/air-verse/air); jika belum ada, `make dev` otomatis memakai `go run` biasa. Install dengan `make install-air`.

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
