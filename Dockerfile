FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 go build -o /jobtracker ./cmd/server

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Create directories for data and db
RUN mkdir -p /app/data /app/db

# Copy binary
COPY --from=builder /jobtracker /app/jobtracker

# Copy web assets (templates, static files)
COPY web/ /app/web/

EXPOSE 8080

ENV JOBTRACKER_ADDR=0.0.0.0:8080
ENV JOBTRACKER_REPO_ROOT=/app
ENV JOBTRACKER_DB_PATH=/app/db/index.sqlite

CMD ["/app/jobtracker"]
