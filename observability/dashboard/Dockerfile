# syntax=docker/dockerfile:1.4
# Multi-stage build for React dashboard

# Build arguments for customization
ARG NODE_VERSION=20
ARG ALPINE_VERSION=3.20
ARG NGINX_VERSION=1.25-alpine3.20

# Using Alpine 3.20 (stable) to fix CVE-2025-46394 and CVE-2024-58251
FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION} AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies with BuildKit cache mount
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production && npm cache clean --force

# Copy source code
COPY . .

# Build the application with BuildKit cache mount
RUN --mount=type=cache,target=/root/.npm \
    npm run build

# Production stage with metadata
FROM nginx:${NGINX_VERSION} AS production

# Metadata
LABEL org.opencontainers.image.title="O-RAN Dashboard"
LABEL org.opencontainers.image.description="O-RAN Observability Dashboard"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Install additional packages for health checks and clean cache
RUN apk add --no-cache curl && \
    rm -rf /var/cache/apk/*

# Copy built assets from builder stage with proper ownership
COPY --from=builder --chown=nginx:nginx /app/dist /usr/share/nginx/html

# Copy nginx configuration with proper ownership
COPY --chown=nginx:nginx nginx.conf /etc/nginx/nginx.conf

# Create nginx directories
RUN mkdir -p /var/log/nginx /var/cache/nginx

# Set proper permissions
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/log/nginx && \
    chown -R nginx:nginx /var/cache/nginx && \
    chmod -R 755 /usr/share/nginx/html

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

# Expose port
EXPOSE 3000

# Switch to nginx user
USER nginx

# Start nginx
CMD ["nginx", "-g", "daemon off;"]