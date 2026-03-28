# Hearth Dockerfile
# Multi-stage build for minimal, secure production image
# Security best practices applied throughout

ARG GO_VERSION=1.22
ARG NODE_VERSION=22
ARG ALPINE_VERSION=3.20

# =============================================================================
# Stage 1: Build backend
# =============================================================================
FROM golang:${GO_VERSION}-alpine AS backend-builder

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /build

# Cache dependencies separately
COPY backend/go.mod backend/go.sum ./
RUN go mod download && go mod verify

# Copy source and build
COPY backend/ ./

# Build with security hardening flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -trimpath \
    -o hearth \
    ./cmd/hearth

# =============================================================================
# Stage 2: Build frontend
# =============================================================================
FROM node:${NODE_VERSION}-alpine AS frontend-builder

WORKDIR /build

# Cache dependencies
COPY frontend/package*.json ./
RUN npm ci --ignore-scripts --no-audit

# Copy source and build
COPY frontend/ ./
RUN npm run build

# =============================================================================
# Stage 3: Production image
# =============================================================================
FROM alpine:${ALPINE_VERSION}

# Metadata
LABEL org.opencontainers.image.title="Hearth"
LABEL org.opencontainers.image.description="Self-hosted Discord alternative with E2EE"
LABEL org.opencontainers.image.source="https://github.com/ghndrx/hearth"
LABEL org.opencontainers.image.licenses="MIT"

# Security: Install minimal packages
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/cache/apk/* /tmp/*

# Security: Create non-root user and group
RUN addgroup -g 1000 hearth && \
    adduser -D -u 1000 -G hearth -h /app -s /sbin/nologin hearth

WORKDIR /app

# Create data directories with correct ownership
RUN mkdir -p /data/uploads /data/db && \
    chown -R hearth:hearth /data

# Copy binaries with correct ownership
COPY --from=backend-builder --chown=hearth:hearth /build/hearth .
COPY --from=frontend-builder --chown=hearth:hearth /build/build ./public

# Security: Run as non-root user
USER hearth

# Environment defaults
ENV HOST=0.0.0.0 \
    PORT=8080 \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    STORAGE_PATH=/data/uploads

# Expose port (non-privileged)
EXPOSE 8080

# Mount point for persistent data
VOLUME ["/data"]

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/hearth"]
