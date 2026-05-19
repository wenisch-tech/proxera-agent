
# proxera-agent
![proxera-agent](docs/img/proxera-agent-banner.png)

Go client for [proxera](https://github.com/wenisch-tech/proxera). It creates a persistent tunnel over WebSocket and proxies inbound tunnel requests to local HTTP services. Can be deployed as a Kubernetes workload via Helm, run as a standalone binary, or installed as a [Home Assistant add-on](hassio-addon/README.md).

## Features

- WebSocket tunnel client with `X-Proxera-Token` authentication.
- Full frame handling for `REGISTER_ACK`, `REQUEST`, `RESPONSE`, `PING`, `PONG`, `ERROR`.
- Local HTTP proxying with hop-by-hop header stripping.
- Configurable heartbeat and reconnect backoff with jitter.
- Structured JSON logging for tunnel lifecycle and proxy traffic.
- CLI flags with environment variable support.

## Quickstart


### Deploying with Helm

Add the chart repository and install the agent into your cluster:

```bash
helm repo add wenisch-tech https://charts.wenisch.tech
helm repo update
helm upgrade --install proxera-agent wenisch-tech/proxera-agent \
  --namespace proxera \
  --create-namespace \
  --set config.serverUrl="wss://proxera.example.com/tunnel" \
  --set secret.apiKey="$PROXERA_API_KEY"
```

Or supply values via a file:

```yaml
# values.yaml
config:
  serverUrl: "wss://proxera.example.com/tunnel"
secret:
  apiKey: "<your-api-key>"
```

```bash
helm upgrade --install proxera-agent wenisch-tech/proxera-agent \
  --namespace proxera \
  --create-namespace \
  -f values.yaml
```

### Using a pre-built binary

Download the latest binary for your platform from the [releases page](https://github.com/wenisch-tech/proxera-agent/releases), then run it directly:

```bash
# Linux / macOS
chmod +x proxera-agent-linux-amd64
./proxera-agent-linux-amd64 \
  --server-url wss://proxera.example.com/tunnel \
  --api-key "$PROXERA_API_KEY"
```

```powershell
# Windows
.\proxera-agent-windows-amd64.exe `
  --server-url wss://proxera.example.com/tunnel `
  --api-key "$env:PROXERA_API_KEY"
```

Verify the download against the published `SHA256SUMS` file before running:

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

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
go -C src run ./cmd/proxera-agent \
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
docker build -t ghcr.io/wenisch-tech/proxera-agent:dev .
```

## Helm

Helm chart lives in `charts/proxera-agent` and is packaged during release.

## Release assets

CI release uploads:

- `proxera-agent-linux-amd64`
- `proxera-agent-linux-arm64`
- `proxera-agent-windows-amd64.exe`
- `proxera-agent-windows-arm64.exe`
- `SHA256SUMS`
- CycloneDX SBOM
- Cosign bundles and provenance attestations
- Helm chart archive and signature bundle
