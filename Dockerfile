# Build stage - Backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o hearth ./cmd/hearth

# Build stage - Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Production image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries
COPY --from=backend-builder /build/hearth .
COPY --from=frontend-builder /build/build ./static

# Create non-root user
RUN adduser -D -u 1000 hearth
RUN mkdir -p /data && chown hearth:hearth /data
USER hearth

# Configuration
ENV PORT=8080
ENV DATABASE_URL=sqlite:///data/hearth.db
ENV STORAGE_PATH=/data/uploads

EXPOSE 8080
VOLUME /data

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/hearth"]
CMD ["serve"]
