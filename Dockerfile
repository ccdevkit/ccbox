# TDD exemption: Infrastructure config with no testable logic (Principle VII amendment).

# Stage 1: Build Go binaries
FROM golang:1.24 AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ccptproxy ./cmd/ccptproxy
COPY cmd/ccclipd ./cmd/ccclipd
COPY cmd/ccdebug ./cmd/ccdebug
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/ccptproxy ./cmd/ccptproxy && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/ccclipd  ./cmd/ccclipd && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/ccdebug  ./cmd/ccdebug

# Stage 2: Runtime image
# Multi-arch: linux/amd64 + linux/arm64 (FR-033)
FROM node:22-slim

# Install system packages (I14)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    ca-certificates \
    netcat-openbsd \
    gosu \
    openssh-client \
    jq \
    ripgrep \
    make \
    build-essential \
    python3 \
    vim-tiny \
    xvfb \
    xclip \
    && rm -rf /var/lib/apt/lists/*

# Create claude user with UID 1001 to avoid conflict with node user at UID 1000 (I13)
RUN groupadd --gid 1001 claude && \
    useradd --uid 1001 --gid 1001 --create-home --shell /bin/bash claude

# Claude Code is NOT installed in the base image.
# It is installed at local-image-build time via generateDockerfile()
# so the version matches the host CLI (or --use override).

# Copy container binaries from builder
COPY --from=builder /out/ccptproxy /opt/ccbox/bin/ccptproxy
COPY --from=builder /out/ccclipd  /opt/ccbox/bin/ccclipd
COPY --from=builder /out/ccdebug  /opt/ccbox/bin/ccdebug

# Ensure binaries are executable
RUN chmod +x /opt/ccbox/bin/*

# Create ~/.local/bin directory (claude will be installed here by local image)
RUN mkdir -p /home/claude/.local/bin \
    && chown -R claude:claude /home/claude/.local

# Create shims directory (populated at runtime by ccptproxy --setup)
RUN mkdir -p /opt/ccbox/bin/shims \
    && chown claude:claude /opt/ccbox/bin/shims

# Set PATH so shims, ccbox binaries, and claude local bin are discoverable (I3)
ENV PATH=/opt/ccbox/bin/shims:/opt/ccbox/bin:/home/claude/.local/bin:$PATH

# No Docker socket mount (FR-030)

COPY entrypoint.sh /opt/ccbox/entrypoint.sh
RUN chmod +x /opt/ccbox/entrypoint.sh

ENTRYPOINT ["/opt/ccbox/entrypoint.sh"]
