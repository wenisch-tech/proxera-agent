# proxera-client

Go client for [proxera](https://github.com/wenisch-tech/proxera). It creates a persistent tunnel over WebSocket and proxies inbound tunnel requests to local HTTP services.

## Features

- WebSocket tunnel client with `X-Proxera-Token` authentication.
- Full frame handling for `REGISTER_ACK`, `REQUEST`, `RESPONSE`, `PING`, `PONG`, `ERROR`.
- Local HTTP proxying with hop-by-hop header stripping.
- Configurable heartbeat and reconnect backoff with jitter.
- Structured JSON logging for tunnel lifecycle and proxy traffic.
- CLI flags with environment variable support.

## Configuration

Required:

- `PROXERA_SERVER_URL` (for example `wss://proxera.example.com/tunnel`)
- `PROXERA_API_KEY`

Optional:

- `PROXERA_LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
- `PROXERA_HEARTBEAT_INTERVAL` (default `30s`)
- `PROXERA_HEARTBEAT_TIMEOUT` (default `10s`)
- `PROXERA_RECONNECT_BASE` (default `1s`)
- `PROXERA_RECONNECT_MAX` (default `60s`)
- `PROXERA_REQUEST_TIMEOUT` (default `30s`)
- `PROXERA_CONCURRENCY_LIMIT` (default `100`)

Flags override environment values:

```bash
go -C src run ./cmd/proxera-client \
  --server-url wss://proxera.example.com/tunnel \
  --api-key "$PROXERA_API_KEY" \
  --log-level info
```

## Build

```bash
make build
```

All Go source and the module definition live under `src/`.


## Test

```bash
make test
```

## Docker

```bash
docker build -t ghcr.io/wenisch-tech/proxera-client:dev .
```

## Helm

Helm chart lives in `charts/proxera-client` and is packaged during release.

## Release assets

CI release uploads:

- `proxera-client-linux-amd64`
- `proxera-client-linux-arm64`
- `proxera-client-windows-amd64.exe`
- `proxera-client-windows-arm64.exe`
- `SHA256SUMS`
- CycloneDX SBOM
- Cosign bundles and provenance attestations
- Helm chart archive and signature bundle
