# syntax=docker/dockerfile:1

# ---------- Stage 1: builder ----------
FROM golang:1.27.3-trixie AS builder

WORKDIR /src

# Binary statis (tanpa cgo) agar bisa jalan di distroless static.
ENV CGO_ENABLED=0

# Cache dependency layer terpisah supaya rebuild tidak download ulang.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Level microarchitecture minimum binary.
# x86-64: v2 = butuh SSE4.2/POPCNT (CPU 2008+).
ARG GOAMD64=v2
# ARM64: v8.0 = ARMv8-A baseline.
ARG GOARM64=v8.0

# GOOS/GOARCH mengikuti platform image (buildx memilih image per platform),
# jadi cross-compile tidak perlu TARGETARCH yang bergantung pada auto-inject.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOAMD64=${GOAMD64} GOARM64=${GOARM64} \
    go build -trimpath -ldflags="-s -w" -o /out/testgo .

# ---------- Stage 2: runtime ----------
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/testgo /app/testgo

ENV PORT=8080
EXPOSE 8080

# Jalankan sebagai user non-root bawaan distroless (uid/gid 65532).
USER nonroot:nonroot

ENTRYPOINT ["/app/testgo"]
