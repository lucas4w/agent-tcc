# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia os arquivos de dependências primeiro (melhor cache)
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila o binário (static, sem CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main .

# Runtime stage (imagem mínima)
FROM alpine:3.20

WORKDIR /app

# Necessário para certificados TLS (conexões com banco, etc.)
RUN apk --no-cache add ca-certificates

# Copia só o binário
COPY --from=builder /app/main .
COPY .env .

#EXPOSE 8080

CMD ["./main"]