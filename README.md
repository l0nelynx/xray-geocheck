# xray-[geocheck](https://github.com/remnawave/geocheck)

Original idea - [xray-checker](https://github.com/kutovoys/xray-checker)  
This project using [geocheck](https://github.com/remnawave/geocheck)

Checks latency, geolocation and reputation of every egress IP from an Xray subscription. Results are published on a dark status page and as Prometheus metrics.

The container downloads pinned **xray-core** and **geocheck** binaries at startup (versions live in `[deps.json](deps.json)`). Each subscription host is fronted by a local SOCKS5 inbound. Ping is a wall-clock HTTP GET through that SOCKS5 hop, not ICMP. geocheck is invoked as `geocheck -p HOST:PORT --json -4 -q`.

TLS is not terminated here; put the service behind an external reverse proxy if you need HTTPS.

Live UI demo (sample data, no live subscription): [https://l0nelynx.github.io/xray-geocheck/](https://l0nelynx.github.io/xray-geocheck/)

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

- Status page: [http://localhost:8080](http://localhost:8080)
- Prometheus: [http://localhost:3113/metrics](http://localhost:3113/metrics) (`METRICS_HOST` / `METRICS_PORT` / `METRICS_BASE_PATH`)
- JSON snapshot: [http://localhost:8080/api/status](http://localhost:8080/api/status)

Manual refresh (status page buttons, or):

- `POST /api/refresh` — re-fetch the subscription immediately, then ping and geocheck every host
- `POST /api/refresh/proxy` — body `{"id":"..."}` ping + geocheck one host

Exit IPs in the JSON API and UI are masked (`104.28.193.121` → `104.***.***.*21`). The in-memory store keeps the raw values.

## Environment


| Variable                                   | Default                                | Notes                                                                                                                               |
| ------------------------------------------ | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `SUBSCRIPTION_URL`                         | required                               | Remnawave / Xray subscription URL                                                                                                   |
| `SUBSCRIPTION_URL_UPDATE_INTERVAL_MINUTES` | `60`                                   | How often to re-fetch the subscription. Refresh all always re-fetches immediately, before geocheck.                                 |
| `USER_AGENT`                               | `xray-geocheck`                        | Sent on the subscription GET. Use a client UA (for tests: `Happ/4.1.0/Windows`) when the panel only returns JSON for known clients. |
| `XRAY_CORE_URL`                            | from `deps.json`                       | Optional override of the xray-core archive URL                                                                                      |
| `GEOCHECK_URL`                             | from `deps.json`                       | Optional override of the geocheck archive URL                                                                                       |
| `PING_CHECK_URL`                           | `https://www.gstatic.com/generate_204` | HTTP GET target for latency                                                                                                         |
| `GEO_CHECK_INTERVAL_MINUTES`               | `720`                                  | Full geocheck sweep                                                                                                                 |
| `PING_CHECK_INTERVAL_MINUTES`              | `5`                                    | Ping sweep                                                                                                                          |
| `LISTEN_ADDR`                              | `:8080`                                | Status page bind address inside the container                                                                                       |
| `UI_BASE_PATH`                             | `/`                                    | Public URL prefix. For nginx `location /geocheck/` set `/geocheck`. Assets and `/api/*` are served under this prefix.               |
| `METRICS_HOST`                             | `0.0.0.0`                              | Metrics listen address (separate HTTP server)                                                                                       |
| `METRICS_PORT`                             | `3113`                                 | Metrics listen port                                                                                                                 |
| `METRICS_BASE_PATH`                        | `/metrics`                             | Prometheus scrape path on the metrics server. A leading slash is added if missing. `/` is rejected.                                 |
| `METRICS_USERNAME`                         | empty                                  | HTTP basic auth user for the metrics endpoint. Must be paired with `METRICS_PASSWORD`.                                              |
| `METRICS_PASSWORD`                         | empty                                  | HTTP basic auth password for the metrics endpoint.                                                                                  |


Subscription requests always send:

- `x-hwid` — 16-character id persisted under `/data/hwid`
- `x-device-os` / `x-ver-os`
- `x-device-model: xray-geocheck`
- `user-agent`

Both Xray JSON arrays (`{ remarks, outbounds }`) and base64 share-link lists are accepted.

## Docker notes

`cap_add: [NET_RAW]` is set so geocheck can open a raw ICMP socket when the kernel allows it. Path traces through SOCKS5 are often empty; HTTP geolocation, reputation and access checks still run.

Binaries are **not** baked into the image. They are fetched on every cold start into `BIN_DIR` (default `/opt/xray-geocheck/bin`).

## Reverse proxy

Set `UI_BASE_PATH` to the nginx location (no trailing slash). Example for `/geocheck/`:

```nginx
location /geocheck/ {
    auth_basic "Private";
    auth_basic_user_file /etc/nginx/.htpasswd;
    proxy_hide_header WWW-Authenticate;
    proxy_set_header Authorization "";
    proxy_pass http://xray-geocheck/;
    include /etc/nginx/conf.d/includes/proxy-params.conf;
}
location = /geocheck {
    return 301 /geocheck/;
}
```

`UI_BASE_PATH=/geocheck`

## Releases

GitHub Actions **Release** is manual: Actions → Release → Run workflow → `tag` like `v1.2.3`.

That job:

- Pushes `ghcr.io/l0nelynx/xray-geocheck:<tag>` and `:latest` (linux/amd64 + linux/arm64)
- Builds Go binaries for linux/windows/darwin (amd64 + arm64, plus windows amd64)
- Creates a GitHub Release with those binaries and `SHA256SUMS`

```bash
docker pull ghcr.io/l0nelynx/xray-geocheck:v1.2.3
```

The package is private while this repository is private; `docker login ghcr.io` is required.

Binaries on the release: `xray-geocheck_<tag>_<os>_<arch>` (`.exe` on Windows). The status page is embedded; xray-core and geocheck are still downloaded at startup from `deps.json`.

If a git tag or GitHub Release with that version already exists, the workflow fails instead of overwriting.

## Demo

The GitHub Pages site is a static Vite build (`VITE_DEMO=true`) with a canned snapshot. Refresh buttons spin and leave the same sample rows.

Local preview:

```bash
cd web
npm ci
npm run build:demo
npm run preview
```

Pages deploys on every push to `main`. Repo Settings → Pages → Source must be **GitHub Actions**. GitHub Free does not publish Pages from a private repository.

## Grafana

Import `[grafana/dashboard.json](grafana/dashboard.json)` (Dashboards → New → Import). Pick your Prometheus datasource. Filter by proxy at the top.

Scrape example:

```yaml
- job_name: xray-geocheck
  scrape_interval: 30s
  static_configs:
    - targets: ["xray-geocheck:3113"]
```

