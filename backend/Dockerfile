# Build stage
FROM golang:alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-linkmode external -extldflags "-static"' -o meerkat .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

# Create directories for data persistence
RUN mkdir -p /app/static/photos /app/data && \
    chown -R appuser:appgroup /app

# Set default environment
ENV PORT=8080
ENV SQLITE_DB_PATH=/app/data/meerkat.db
ENV PROFILE_PHOTO_DIR=/app/static/photos

# Copy the binary from builder
COPY --from=builder /app/meerkat .

# Copy static assets if any
COPY --from=builder /app/static/styles.css ./static/

# Switch to non-root user
USER appuser

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./meerkat"]
