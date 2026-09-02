# --- Этап сборки (Build Stage) ---
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код (включая pkg/database/migrations/)
COPY . .

# Собираем бинарник с отключенным CGO под Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# --- Этап запуска (Run Stage) ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Копируем только скомпилированный бинарник.
# SQL-файл уже «вшит» внутрь него благодаря go:embed!
COPY --from=builder /app/main .

# Запуск приложения
CMD ["./main"]
