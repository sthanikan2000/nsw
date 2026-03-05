FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/oga ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --system appgroup \
    && useradd --system --gid appgroup --create-home appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /out/oga /app/oga
COPY --from=builder --chown=appuser:appgroup /app/data/forms /app/data/forms

USER appuser

EXPOSE 8081

CMD ["/app/oga"]
