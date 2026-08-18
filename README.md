# xray-geocheck

Checks latency, geolocation and reputation of every egress IP from an Xray subscription. Results are published on a dark status page and as Prometheus metrics.

The container downloads pinned **xray-core** and **geocheck** binaries at startup (versions live in [`deps.json`](deps.json)). Each subscription host is fronted by a local SOCKS5 inbound. Ping is a wall-clock HTTP GET through that SOCKS5 hop, not ICMP. geocheck is invoked as `geocheck -p HOST:PORT --json -4 -q`.

TLS is not terminated here; put the service behind an external reverse proxy if you need HTTPS.

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

- Status page: http://localhost:8080
- Prometheus: http://localhost:3113/metrics (`METRICS_HOST` / `METRICS_PORT` / `METRICS_BASE_PATH`)
- JSON snapshot: http://localhost:8080/api/status

Manual refresh (status page buttons, or):

- `POST /api/refresh` — re-fetch the subscription immediately, then ping and geocheck every host
- `POST /api/refresh/proxy` — body `{"id":"..."}` ping + geocheck one host

Exit IPs in the JSON API and UI are masked (`104.28.193.121` → `104.***.***.*21`). The in-memory store keeps the raw values.

## Environment

| Variable | Default | Notes |
| --- | --- | --- |
| `SUBSCRIPTION_URL` | required | Remnawave / Xray subscription URL |
| `SUBSCRIPTION_URL_UPDATE_INTERVAL_MINUTES` | `60` | How often to re-fetch the subscription. Refresh all always re-fetches immediately, before geocheck. |
| `USER_AGENT` | `xray-geocheck` | Sent on the subscription GET. Use a client UA (for tests: `Happ/4.1.0/Windows`) when the panel only returns JSON for known clients. |
| `XRAY_CORE_URL` | from `deps.json` | Optional override of the xray-core archive URL |
| `GEOCHECK_URL` | from `deps.json` | Optional override of the geocheck archive URL |
| `PING_CHECK_URL` | `https://www.gstatic.com/generate_204` | HTTP GET target for latency |
| `GEO_CHECK_INTERVAL_MINUTES` | `720` | Full geocheck sweep |
| `PING_CHECK_INTERVAL_MINUTES` | `5` | Ping sweep |
| `LISTEN_ADDR` | `:8080` | Status page bind address inside the container |
| `METRICS_HOST` | `0.0.0.0` | Metrics listen address (separate HTTP server) |
| `METRICS_PORT` | `3113` | Metrics listen port |
| `METRICS_BASE_PATH` | `/metrics` | Prometheus scrape path on the metrics server. A leading slash is added if missing. `/` is rejected. |
| `METRICS_USERNAME` | empty | HTTP basic auth user for the metrics endpoint. Must be paired with `METRICS_PASSWORD`. |
| `METRICS_PASSWORD` | empty | HTTP basic auth password for the metrics endpoint. |

Subscription requests always send:

- `x-hwid` — 16-character id persisted under `/data/hwid`
- `x-device-os` / `x-ver-os`
- `x-device-model: xray-geocheck`
- `user-agent`

Both Xray JSON arrays (`{ remarks, outbounds }`) and base64 share-link lists are accepted.

## Docker notes

`cap_add: [NET_RAW]` is set so geocheck can open a raw ICMP socket when the kernel allows it. Path traces through SOCKS5 are often empty; HTTP geolocation, reputation and access checks still run.

Binaries are **not** baked into the image. They are fetched on every cold start into `BIN_DIR` (default `/opt/xray-geocheck/bin`).
