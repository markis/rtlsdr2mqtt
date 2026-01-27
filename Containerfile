# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETVARIANT

# Install base dependencies
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
  apk add \
  pkgconf \
  git \
  ca-certificates \
  build-base \
  librtlsdr-dev \
  libusb-dev

# Set build environment variables based on target platform
ENV CGO_ENABLED=1
ENV GOOS=linux
RUN case "${TARGETARCH}" in \
  "arm") \
  echo "export GOARCH=arm" >> /tmp/buildenv && \
  echo "export GOARM=7" >> /tmp/buildenv \
  ;; \
  "arm64") \
  echo "export GOARCH=arm64" >> /tmp/buildenv \
  ;; \
  "amd64") \
  echo "export GOARCH=amd64" >> /tmp/buildenv \
  ;; \
  *) \
  echo "export GOARCH=${TARGETARCH}" >> /tmp/buildenv \
  ;; \
  esac

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

# Copy source code
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  . /tmp/buildenv && go build -a -o rtlsdr2mqtt ./cmd/rtlsdr2mqtt

# Production stage
FROM alpine:3.21

# Enable healthcheck by default
ENV HEALTHCHECK_FILE=/var/run/healthcheck

# Install runtime dependencies
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
  apk add \
  ca-certificates \
  librtlsdr \
  libusb \
  tzdata

# Copy the health check script
COPY --chmod=755 scripts/healthcheck.sh /usr/bin/healthcheck.sh

# Copy binary from builder stage (rtlamr is now integrated as a Go library)
COPY --from=builder --chmod=755 /app/rtlsdr2mqtt /usr/bin/rtlsdr2mqtt

# Use SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

# Health check: verify rtlsdr2mqtt is receiving messages
HEALTHCHECK --interval=60s --timeout=10s --start-period=30s --retries=3 \
  CMD ["/usr/bin/healthcheck.sh"]

ENTRYPOINT ["/usr/bin/rtlsdr2mqtt"]
