# Stage 1: Сборка
FROM golang:1.24.5 as builder

WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service .

# Stage 2: Запуск
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из stage 1 - ПРАВИЛЬНЫЙ ПУТЬ: /app/order-service
COPY --from=builder /app/order-service .

# Копируем статические файлы веб-интерфейса - ПРАВИЛЬНЫЙ ПУТЬ: /app/web
COPY --from=builder /app/web ./web

EXPOSE 8081

CMD ["./order-service"]