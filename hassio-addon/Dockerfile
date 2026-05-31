ARG BUILD_FROM=ghcr.io/home-assistant/base:latest
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG BUILD_VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_REVISION=none
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY src/go.mod src/go.sum ./src/
RUN go -C src mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go -C src build \
  -trimpath \
  -ldflags "-s -w \
    -X github.com/wenisch-tech/proxera-agent/internal/version.Version=${BUILD_VERSION} \
    -X github.com/wenisch-tech/proxera-agent/internal/version.Commit=${BUILD_REVISION} \
    -X github.com/wenisch-tech/proxera-agent/internal/version.BuildDate=${BUILD_DATE}" \
  -o /out/proxera-agent ./cmd/proxera-agent

FROM ${BUILD_FROM}

ARG BUILD_VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_REVISION=none

LABEL \
  io.hass.name="Proxera Agent" \
  io.hass.description="WebSocket tunnel client that proxies inbound tunnel requests to local HTTP services." \
  io.hass.type="addon" \
  io.hass.arch="aarch64|amd64" \
  io.hass.version="${BUILD_VERSION}" \
  org.opencontainers.image.title="proxera-agent" \
  org.opencontainers.image.description="Go client for proxera tunnels" \
  org.opencontainers.image.version="${BUILD_VERSION}" \
  org.opencontainers.image.created="${BUILD_DATE}" \
  org.opencontainers.image.revision="${BUILD_REVISION}" \
  org.opencontainers.image.source="https://github.com/wenisch-tech/proxera-agent"

COPY --from=builder /out/proxera-agent /usr/bin/proxera-agent
COPY hassio-addon/run.sh /run.sh
RUN chmod a+x /run.sh

CMD [ "/run.sh" ]
