# Multi-stage build for Go application with React frontend

# Stage 1: Build Frontend
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o syncservice cmd/syncservice/main.go

# Stage 3: Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=backend-builder /app/syncservice .

# Copy frontend build from frontend-builder
COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Copy config directory
COPY config/ ./config/

# Expose port
EXPOSE 8080

# Run the application
CMD ["./syncservice", "-config", "config/sync-config.yaml"]
