# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api cmd/main.go

# Stage 2: Minimal production image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /api /api

EXPOSE 8080

CMD ["/api"]
