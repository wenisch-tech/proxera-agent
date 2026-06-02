# Proxera Agent — Home Assistant Add-on

![Logo](../docs/img/proxera-agent-banner.png)

Go client for [proxera](https://github.com/wenisch-tech/proxera). Creates a persistent WebSocket tunnel and proxies inbound requests to local HTTP services running on your Home Assistant host.

## How it works

The add-on connects outbound to your proxera server using a WebSocket connection authenticated with an API key. The server can then forward HTTP requests through the tunnel to services running locally on your Home Assistant instance — no port forwarding or public IP required.

```
Internet → proxera server → WebSocket tunnel → proxera-agent → local service (e.g. :8123)
```

## Prerequisites

- A running [proxera](https://github.com/wenisch-tech/proxera) server reachable over `wss://` or `ws://`
- An API key issued by your proxera server

## Installation

[![Add repository to Home Assistant](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fwenisch-tech%2Fproxera-agent)

Click the button above, or add the repository manually:

1. In Home Assistant, go to **Settings → Add-ons → Add-on Store**
2. Click the menu (⋮) in the top right and choose **Repositories**
3. Add the following URL and click **Add**:
   ```
   https://github.com/wenisch-tech/proxera-agent
   ```
4. Find **Proxera Agent** in the store and click **Install**

## Configuration

After installation, go to the add-on's **Configuration** tab and fill in at minimum `server_url` and `api_key`.

| Option | Required | Default | Description |
|---|---|---|---|
| `server_url` | Yes | — | WebSocket URL of your proxera server, e.g. `wss://proxera.example.com/tunnel` |
| `api_key` | Yes | — | Authentication token sent as `X-Proxera-Token` |
| `log_level` | No | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `heartbeat_interval` | No | `30s` | How often a ping is sent to keep the tunnel alive |
| `heartbeat_timeout` | No | `10s` | How long to wait for a pong before treating the connection as dead |
| `reconnect_base` | No | `1s` | Initial backoff duration before the first reconnect attempt |
| `reconnect_max` | No | `60s` | Maximum backoff duration between reconnect attempts |
| `request_timeout` | No | `30s` | Timeout for proxied HTTP requests to local services |
| `concurrency_limit` | No | `100` | Maximum number of in-flight proxied requests |

Duration values use Go duration syntax: `30s`, `1m`, `500ms`.

### Example configuration

```yaml
server_url: wss://proxera.example.com/tunnel
api_key: "your-api-key-here"
log_level: info
```

## Starting the add-on

1. Save your configuration
2. Go to the **Info** tab and click **Start**
3. Check the **Log** tab — you should see:
   ```
   Starting proxera-agent...
   Server URL: wss://proxera.example.com/tunnel
   ```
   followed by a log line confirming the tunnel is registered with the server.

The add-on is configured with `startup: services`, so it starts automatically before Home Assistant Core on every boot.

## Exposing Home Assistant

When the route points at Home Assistant itself, disable **Forward client IP headers** on that route in the Proxera Admin UI. The add-on simply forwards the request headers it receives from the Proxera server to Home Assistant; this setting is controlled on the server route, not in the add-on.

With the route option disabled, Proxera strips `X-Forwarded-For`, `X-Real-IP`, `Forwarded`, and related client IP headers before the request reaches Home Assistant. It still sends `X-Forwarded-Host`, `X-Forwarded-Proto`, and `X-Forwarded-Port`, so Home Assistant can still build correct external URLs.

In Home Assistant's `configuration.yaml`, keep `http.use_x_forwarded_for` unset or set it to `false` unless you explicitly need Home Assistant to trust the original public client IP:

```yaml
http:
  use_x_forwarded_for: false
```

If you set `use_x_forwarded_for: true`, Home Assistant requires a correct `trusted_proxies` entry for the immediate proxy source. Even then, changing forwarded client IP chains can cause Home Assistant login flows to fail with `IP address changed`.

## Troubleshooting

**Add-on fails to start**
Check the **Log** tab. The most common causes are a missing `server_url` or `api_key`, or a server URL that does not start with `ws://` or `wss://`.

**Tunnel connects but requests time out**
Ensure the local service your proxera server is routing to is actually running on the Home Assistant host and accessible on the expected host and port.

**Home Assistant login fails after credentials**
Disable **Forward client IP headers** on the Home Assistant route in Proxera, and keep Home Assistant's `http.use_x_forwarded_for` unset or set to `false`. This avoids Home Assistant rejecting the auth flow when `X-Forwarded-For` or `X-Real-IP` changes between the login-flow request and token exchange.

**Home Assistant reports `IP address changed`**
This is normally caused by forwarded client IP headers. Check the route first: **Forward client IP headers** should be off for the Home Assistant route. Then check Home Assistant's `http` configuration and remove `use_x_forwarded_for: true` unless you also maintain an accurate `trusted_proxies` list and accept the login-flow behavior.

**Frequent disconnections**
Try lowering `heartbeat_interval` (e.g. `15s`) or increasing `heartbeat_timeout` (e.g. `20s`) if you have a high-latency connection to your proxera server.

**Enable debug logging**
Set `log_level: debug` and restart the add-on to see detailed frame-level traffic.

## Links

- [proxera server](https://github.com/wenisch-tech/proxera) — the server-side component
- [Full source & Helm chart](https://github.com/wenisch-tech/proxera-agent)
- [Changelog](https://github.com/wenisch-tech/proxera-agent/blob/main/CHANGELOG.md)
