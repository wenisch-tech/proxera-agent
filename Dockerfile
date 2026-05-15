FROM golang:1.24-alpine AS builder
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_REVISION=none
WORKDIR /workspace

COPY src/go.mod src/go.sum ./src/
RUN go -C src mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C src build \
  -trimpath \
  -ldflags "-s -w -X github.com/wenisch-tech/proxera-client/internal/version.Version=${VERSION} -X github.com/wenisch-tech/proxera-client/internal/version.Commit=${BUILD_REVISION} -X github.com/wenisch-tech/proxera-client/internal/version.BuildDate=${BUILD_DATE}" \
  -o /out/proxera-client ./cmd/proxera-client

FROM cgr.dev/chainguard/static:latest
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG BUILD_REVISION=none

LABEL org.opencontainers.image.title="proxera-client" \
      org.opencontainers.image.description="Go client for proxera tunnels" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${BUILD_REVISION}" \
      org.opencontainers.image.source="https://github.com/wenisch-tech/proxera-client"

COPY --from=builder /out/proxera-client /usr/local/bin/proxera-client
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/proxera-client"]
