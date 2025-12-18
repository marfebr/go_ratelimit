# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia go.mod e go.sum
COPY go.mod go.sum ./

# Download de dependências
RUN go mod download

# Copia código fonte
COPY . .

# Build da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ratelimiter ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Copia binário do stage de build
COPY --from=builder /app/bin/ratelimiter .

# Expõe porta 8080
EXPOSE 8080

# Comando de execução
CMD ["./ratelimiter"]
