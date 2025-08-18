package templates

// Docker and deployment templates

// DockerfileTemplate template
const DockerfileTemplate = `FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Copy the binary
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
`

// DockerComposeTemplate template
const DockerComposeTemplate = `version: '3.8'

services:
  {{.ServiceName}}:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://user:password@postgres:5432/{{.ServiceName}}_db?sslmode=disable
      - LOG_LEVEL=info
    depends_on:
      - postgres
    networks:
      - {{.ServiceName}}_network

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB={{.ServiceName}}_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - {{.ServiceName}}_network

  migrate:
    image: migrate/migrate
    networks:
      - {{.ServiceName}}_network
    volumes:
      - ./migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://user:password@postgres:5432/{{.ServiceName}}_db?sslmode=disable",
      "up"
    ]
    depends_on:
      - postgres

volumes:
  postgres_data:

networks:
  {{.ServiceName}}_network:
    driver: bridge
`
