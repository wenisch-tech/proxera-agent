# Proxera Agent

Creates a persistent WebSocket tunnel to your proxera server and proxies inbound requests to local HTTP services on your Home Assistant host.

## Configuration

| Option | Required | Default | Description |
|---|---|---|---|
| `server_url` | Yes | — | WebSocket URL of your proxera server, e.g. `wss://proxera.example.com/tunnel` |
| `api_key` | Yes | — | Authentication token (`X-Proxera-Token`) |
| `log_level` | No | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `heartbeat_interval` | No | `30s` | Ping interval for keepalive |
| `heartbeat_timeout` | No | `10s` | Pong timeout before the connection is considered dead |
| `reconnect_base` | No | `1s` | Initial backoff before first reconnect attempt |
| `reconnect_max` | No | `60s` | Maximum backoff between reconnect attempts |
| `request_timeout` | No | `30s` | Timeout for proxied HTTP requests to local services |
| `concurrency_limit` | No | `100` | Max in-flight proxied requests |

Duration values use Go duration syntax: `30s`, `1m`, `500ms`.

Only `server_url` and `api_key` are required. All other options fall back to the defaults listed above when left empty.

For full installation instructions, troubleshooting, and examples see the [README](https://github.com/wenisch-tech/proxera-agent/blob/main/hassio-addon/README.md).
