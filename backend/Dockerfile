# Build stage
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o meerkat .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata shadow su-exec libc6-compat

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

# Copy entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose the port
EXPOSE 8080

# Run as root initially; entrypoint drops to appuser after adjusting UID/GID
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./meerkat"]
