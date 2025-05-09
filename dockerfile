# Stage 1: Build the Vue frontend
FROM --platform=$BUILDPLATFORM node:18 AS frontend-builder
WORKDIR /app
COPY frontend/ .
RUN npm install && npm run build

# Stage 2: Build the Go backend
FROM --platform=$BUILDPLATFORM golang:1.21 AS backend-builder
WORKDIR /app
COPY backend/ .
RUN GOOS=linux GOARCH=$(go env GOARCH) go build -o server .

# Stage 3: Final container with Caddy
FROM --platform=$TARGETPLATFORM caddy:2.7
WORKDIR /app
COPY --from=backend-builder /app/server /app/server
COPY --from=frontend-builder /app/dist /app/frontend
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
CMD ["/app/server"]
