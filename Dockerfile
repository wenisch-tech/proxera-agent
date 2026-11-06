FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
ARG VERSION=dev
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
  -ldflags "-s -w -X github.com/wenisch-tech/proxera-agent/internal/version.Version=${VERSION} -X github.com/wenisch-tech/proxera-agent/internal/version.Commit=${BUILD_REVISION} -X github.com/wenisch-tech/proxera-agent/internal/version.BuildDate=${BUILD_DATE}" \
  -o /out/proxera-agent ./cmd/proxera-agent

FROM cgr.dev/chainguard/static:latest
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_REVISION=none

LABEL org.opencontainers.image.title="proxera-agent" \
      org.opencontainers.image.description="Go client for proxera tunnels" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${BUILD_REVISION}" \
      org.opencontainers.image.source="https://github.com/wenisch-tech/proxera-agent"

COPY --from=builder /out/proxera-agent /usr/local/bin/proxera-agent
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/proxera-agent"]
