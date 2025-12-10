# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build binary
RUN go build -o /bin/code-indexer \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    .

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-client \
    tzdata

# Create non-root user
RUN addgroup -g 1000 indexer && \
    adduser -D -u 1000 -G indexer indexer

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/code-indexer /app/code-indexer

# Create repos directory and set ownership
RUN mkdir -p /repos && chown -R indexer:indexer /app /repos

# Switch to non-root user
USER indexer

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep -f code-indexer || exit 1

# Run the indexer
ENTRYPOINT ["/app/code-indexer"]
CMD ["-mode", "serve"]
