FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/api ./cmd/migrate ./cmd/seed ./cmd/invite ./cmd/export

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends poppler-utils ca-certificates \
    && rm -rf /var/lib/apt/lists/* && useradd -r -u 10001 app
COPY --from=go /out/ /usr/local/bin/
COPY --from=web /web/dist /srv/web
WORKDIR /app
ENV WEB_DIR=/srv/web PORT=8080
USER app
EXPOSE 8080
CMD ["api"]
