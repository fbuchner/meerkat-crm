# Build stage
FROM node:lts-alpine AS builder

WORKDIR /app

# Copy package files first for better caching
COPY package.json yarn.lock* package-lock.json* ./

# Install dependencies
RUN if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; fi

# Copy source code
COPY . .

# Build the app with the API URL injected at build time
ARG REACT_APP_API_URL
ENV REACT_APP_API_URL=${REACT_APP_API_URL}

RUN if [ -f yarn.lock ]; then yarn build; else npm run build; fi

# Production stage - serve with unprivileged nginx
FROM nginxinc/nginx-unprivileged:alpine

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder
COPY --from=builder /app/build /usr/share/nginx/html

# nginx-unprivileged runs on port 8080 by default
EXPOSE 8080

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
