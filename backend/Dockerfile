# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build both the server and the migrate CLI; the runtime image carries
# both so deployments can run schema migrations as a separate step.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server  ./cmd/server  && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/server  /app/server
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /src/migrations /app/migrations
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 8080
USER app
# On boot: apply any pending schema migrations, then exec the server.
# Migrations are idempotent — running them again is a no-op when the
# version table is already at the latest revision.
ENTRYPOINT ["/bin/sh", "-c", "/app/migrate up && exec /app/server"]
