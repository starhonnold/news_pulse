# API Gateway Dockerfile
# Многоэтапная сборка для оптимизации размера образа

# Этап сборки
FROM golang:1.21-alpine AS builder

# Установка необходимых пакетов
RUN apk add --no-cache git ca-certificates tzdata curl

# Создание пользователя для безопасности
RUN adduser -D -s /bin/sh -u 1001 appuser

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o api-gateway \
    ./cmd/main.go

# Финальный этап
FROM scratch

# Импорт пользователя и группы из builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Копирование SSL сертификатов
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копирование временных зон
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копирование curl
COPY --from=builder /usr/bin/curl /usr/bin/curl

# Копирование скомпилированного приложения
COPY --from=builder /app/api-gateway /api-gateway

# Копирование конфигурационных файлов
COPY --from=builder /app/config /config

# Создание директории для логов
USER appuser

# Установка переменных окружения
ENV APP_ENV=production
ENV TZ=UTC

# Открытие портов
EXPOSE 8080 8090 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/api-gateway", "health"] || exit 1

# Команда запуска
ENTRYPOINT ["/api-gateway"]
