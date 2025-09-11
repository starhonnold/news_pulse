# Notification Service Dockerfile
# Многоэтапная сборка для оптимизации размера образа

# Этап сборки
FROM golang:1.21-alpine AS builder

# Установка необходимых пакетов
RUN apk add --no-cache git ca-certificates tzdata

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
    -o notification-service \
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

# Копирование скомпилированного приложения
COPY --from=builder /app/notification-service /notification-service

# Копирование конфигурационных файлов
COPY --from=builder /app/config /config

# Создание директории для логов
USER appuser

# Установка переменных окружения
ENV APP_ENV=production
ENV TZ=UTC

# Открытие портов
EXPOSE 8084 8094 9094

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/notification-service", "health"] || exit 1

# Команда запуска
ENTRYPOINT ["/notification-service"]
