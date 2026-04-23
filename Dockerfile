FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Python AI builder stage
FROM python:3.11-slim AS python-builder

WORKDIR /app

# Copy Python scripts
COPY scripts/ ./scripts/

# Install Python dependencies
RUN pip install --no-cache-dir \
    openai \
    anthropic \
    litellm \
    mem0ai \
    requests \
    websocket-client

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates python3 py3-pip

WORKDIR /app

# Copy Go binary
COPY --from=builder /app/server .

# Copy Python scripts and install dependencies
COPY --from=python-builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=python-builder /app/scripts ./scripts

# Make Python scripts executable
RUN chmod +x ./scripts/*.py

EXPOSE 8080

CMD ["./server"]