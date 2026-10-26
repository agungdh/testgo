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

ARG TARGETOS=linux
ARG TARGETARCH=amd64
# Level microarchitecture minimum binary. v2 = butuh SSE4.2/POPCNT (CPU 2008+).
ARG GOAMD64=v2

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOAMD64=${GOAMD64} \
    go build -trimpath -ldflags="-s -w" -o /out/testgo .

# ---------- Stage 2: runtime ----------
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/testgo /app/testgo

ENV PORT=8080
EXPOSE 8080

# Jalankan sebagai user non-root bawaan distroless (uid/gid 65532).
USER nonroot:nonroot

ENTRYPOINT ["/app/testgo"]
