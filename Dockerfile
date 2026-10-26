# syntax=docker/dockerfile:1

# Runtime-only image. The binary is built in GitHub Actions (cross-compiled
# per architecture) and copied in here, so this image contains no toolchain.
FROM gcr.io/distroless/static-debian13:nonroot

ARG BINARY=dist/testgo
COPY ${BINARY} /app/testgo

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/testgo"]
