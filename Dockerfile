# Build the status-page SPA.
FROM node:22-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build the Go service with the SPA embedded.
FROM golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist ./webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/xray-geocheck ./cmd/xray-geocheck

# Runtime image. xray-core and geocheck are downloaded on startup.
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=backend /out/xray-geocheck /usr/local/bin/xray-geocheck
COPY deps.json /opt/xray-geocheck/deps.json
ENV DEPS_FILE=/opt/xray-geocheck/deps.json
ENV BIN_DIR=/opt/xray-geocheck/bin
ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080
WORKDIR /opt/xray-geocheck
VOLUME ["/data"]
EXPOSE 8080 3113
ENTRYPOINT ["/usr/local/bin/xray-geocheck"]
