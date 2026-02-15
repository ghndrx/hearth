# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the app
RUN npm run build

# Runtime stage
FROM node:20-alpine

WORKDIR /app

# Install serve for static hosting
RUN npm install -g serve

# Copy built assets
COPY --from=builder /app/build ./build

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 3000

CMD ["serve", "-s", "build", "-l", "3000"]
