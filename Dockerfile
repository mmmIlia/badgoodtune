# syntax=docker/dockerfile:1.7

FROM node:24.21.0-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27.1-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ ./cmd/
COPY db/ ./db/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 badgoodtune && \
    adduser -S -D -H -u 10001 -G badgoodtune badgoodtune
WORKDIR /app
COPY --from=go-builder --chown=10001:10001 /out/server /out/migrate /app/
COPY --from=web-builder --chown=10001:10001 /src/web/dist /app/web
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/server"]
