# Dockerfile for O-RAN NLP Service
# Multi-stage build for production-ready image

FROM python:3.11-slim AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements-nlp-service.txt .

# Install Python dependencies
RUN pip install --user --no-cache-dir -r requirements-nlp-service.txt

# Production stage
FROM python:3.11-slim

WORKDIR /app

# Copy Python dependencies from builder
COPY --from=builder /root/.local /root/.local

# Copy application code
COPY intent_parser.py .
COPY schema_validator.py .
COPY nlp_service.py .
COPY qos_schema.json .

# Make sure scripts in .local are usable
ENV PATH=/root/.local/bin:$PATH

# Create non-root user
RUN useradd -m -u 1000 nlp && \
    chown -R nlp:nlp /app

USER nlp

# Expose port
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8082/health')"

# Run application
CMD ["python", "nlp_service.py"]
