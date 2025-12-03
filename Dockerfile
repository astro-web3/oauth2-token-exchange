FROM bufbuild/buf:1.47.2 AS proto

WORKDIR /workspace

COPY pb ./pb

WORKDIR /workspace/pb

RUN buf generate

FROM golang:1.25.3-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates

# Copy dependency files for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

COPY --from=proto /workspace/pb/gen ./pb/gen

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -trimpath \
    -tags netgo \
    -o authz \
    ./cmd/authz

FROM alpine:3.21

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary and config
COPY --from=builder /build/authz .
COPY --from=builder /build/config ./config

# Expose gRPC server port
EXPOSE 8123

CMD ["./authz"]


