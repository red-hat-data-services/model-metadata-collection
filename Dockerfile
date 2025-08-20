# Multi-stage Dockerfile for model metadata collection
#
# Stage 1: Builder - Build the Go application
FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

# Set working directory
WORKDIR /opt/app-root/src

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
USER root
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Stage 2: Generator - Generate the models catalog
FROM registry.access.redhat.com/ubi9-minimal:latest AS generator

# Install required packages for network access and data generation
RUN microdnf update -y && \
    microdnf install -y ca-certificates && \
    microdnf clean all

# Create working directory
WORKDIR /app

# Copy the built binary from builder stage
COPY --from=builder /opt/app-root/src/build/model-extractor /usr/local/bin/model-extractor

# Copy data files needed for generation
COPY --from=builder /opt/app-root/src/data/ ./data/

# Create output directory
RUN mkdir -p output

# Run the model extractor to generate the catalog
# This will create data/models-catalog.yaml
# Mount Docker config secret to access private registries
RUN --mount=type=secret,id=dockerconfig,target=/root/.docker/config.json \
    model-extractor

# Verify the catalog was generated
RUN ls -la data/models-catalog.yaml

# Stage 3: Final runtime image - Minimal image with just the catalog
FROM registry.access.redhat.com/ubi9-micro:latest AS runtime

# Create directory for mounting by other applications
RUN mkdir -p /app/data

# Copy the generated catalog and index files from the generator stage
COPY --from=generator /app/data/models-catalog.yaml /app/data/
COPY --from=generator /app/data/models-index.yaml /app/data/

# Set proper permissions
RUN chmod 644 /app/data/models-catalog.yaml /app/data/models-index.yaml

# Create a non-root user for security
RUN echo "catalog:x:1001:1001:Catalog User:/app:/sbin/nologin" >> /etc/passwd && \
    echo "catalog:x:1001:" >> /etc/group && \
    chown -R 1001:1001 /app

# Switch to non-root user
USER 1001

# Set working directory
WORKDIR /app

# Volume mount point for external applications
VOLUME ["/app/data"]

# Default command - just keep container running for data access
CMD ["sleep", "infinity"]

# Labels for image metadata
LABEL name="model-metadata-collection" \
      version="1.0" \
      description="Model metadata collection catalog generator" \
      io.k8s.description="Generates and serves model metadata catalog" \
      io.k8s.display-name="Model Metadata Collection" \
      io.openshift.tags="ai,models,metadata,catalog"
