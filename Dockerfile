FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-s -w' -o code-indexer .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git openssh-client

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/code-indexer .

# Create repos directory
RUN mkdir -p /repos

# Expose HTTP port
EXPOSE 8080

# Run as non-root user
RUN addgroup -g 1000 indexer && \
    adduser -D -u 1000 -G indexer indexer && \
    chown -R indexer:indexer /app /repos

USER indexer

ENTRYPOINT ["./code-indexer"]
CMD ["-mode", "serve"]
