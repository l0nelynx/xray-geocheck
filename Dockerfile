# Build stages run on the builder's native arch (not QEMU).
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist ./webui/dist
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/xray-geocheck ./cmd/xray-geocheck

FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS certs
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Runtime image. xray-core and geocheck are downloaded on startup.
FROM debian:bookworm-slim
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=backend /out/xray-geocheck /usr/local/bin/xray-geocheck
COPY deps.json /opt/xray-geocheck/deps.json
ENV DEPS_FILE=/opt/xray-geocheck/deps.json
ENV BIN_DIR=/opt/xray-geocheck/bin
ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
WORKDIR /opt/xray-geocheck
VOLUME ["/data"]
EXPOSE 8080 3113
ENTRYPOINT ["/usr/local/bin/xray-geocheck"]
