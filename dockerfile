# Build Stage
FROM golang:1.25-alpine AS builder

# Install git and ca-certificates (required for downloading Go modules securely)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# The asterisk makes go.sum optional so the build doesn't crash if it's missing
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a strictly static Linux binary
ARG APP_NAME
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /app/bin/service ./cmd/${APP_NAME}/main.go

# Production Stage (Ultra lightweight)
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/service .

CMD ["./service"]
