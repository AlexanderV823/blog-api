# --- Этап сборки ---
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Копируем и кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код проекта
COPY . .

# Собираем статически слинкованный бинарник Go
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# --- Этап запуска ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Забираем готовый бинарник из этапа сборки
COPY --from=builder /app/main .

# Запуск приложения
CMD ["./main"]
