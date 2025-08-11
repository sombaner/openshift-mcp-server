# Multi-stage Dockerfile for OpenShift AI MCP Server with Inference
FROM --platform=linux/amd64 golang:1.24-alpine AS go-builder

# Install git (required for Go modules with GitHub dependencies)
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy Go modules files
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Build the integrated Go server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o integrated-server ./cmd/integrated-server

FROM --platform=linux/amd64 python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY python/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy Python inference server code
COPY python/kubernetes_mcp_server/ ./kubernetes_mcp_server/

# Copy Go binary from builder stage
COPY --from=go-builder /src/integrated-server ./integrated-server

# Copy configuration files
COPY manifests/ ./manifests/

# Create necessary directories
RUN mkdir -p /app/models /app/workspace /tmp

# Set environment variables
ENV PYTHONPATH=/app
ENV PORT=8080
ENV MCP_PORT=8081
ENV INFERENCE_PORT=8080
ENV MCP_PROFILE=cicd
ENV LOG_LEVEL=2
ENV DEFAULT_REGISTRY=quay.io
ENV DEFAULT_NAMESPACE=openshift-ai-mcp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose ports
EXPOSE 8080 8081

# Set user for security (non-root)
RUN groupadd -r appuser && useradd -r -g appuser appuser
RUN chown -R appuser:appuser /app
USER appuser

# Start the integrated server
ENTRYPOINT ["./integrated-server"]