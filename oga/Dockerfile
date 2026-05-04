FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/oga ./cmd/server

RUN mkdir -p /out/app /out/data

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW OGA Backend Service"

COPY --from=builder --chown=65532:65532 /out/app /app
COPY --from=builder --chown=65532:65532 /out/data /data
COPY --from=builder --chown=65532:65532 /out/oga /app/oga
COPY --from=builder --chown=65532:65532 /app/data/forms /app/data/forms

WORKDIR /app

# Runtime settings
USER 65532:65532

# Default port (override via OGA_PORT env var)
EXPOSE 8081

CMD ["/app/oga"]
