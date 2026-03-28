# Build stage with Bun
FROM oven/bun:1 AS builder

WORKDIR /app

# Copy package files
COPY package.json bun.lockb* ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy source code
COPY . .

# Build the app
RUN bun run build

# Runtime stage (using Bun's built-in server)
FROM oven/bun:1-alpine

WORKDIR /app

# Copy built assets
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 3000

# Use Bun to serve static files
CMD ["bun", "x", "serve", "-s", "build", "-l", "3000"]
