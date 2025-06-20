FROM golang:1.24.3-bullseye AS builder

# Install build dependencies including eccodes
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libeccodes-dev \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o weather-fetch-go cmd/weather-fetch/main.go

# Stage 2: Runtime stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libeccodes0 \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

WORKDIR /app

COPY --from=builder /app/weather-fetch-go .

RUN mkdir -p tmp storage && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE ${PORT:-80}

ENV PORT=80

CMD ["./weather-fetch-go"]
