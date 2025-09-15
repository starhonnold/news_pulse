# News Parsing Service Dockerfile (Simplified - No Chromium)
# Многоэтапная сборка для оптимизации размера образа

# Этап сборки
FROM golang:1.24-alpine AS builder

# Установка только необходимых пакетов (без Chromium)
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
    -o news-parsing-service \
    ./cmd/main.go

# Финальный этап (упрощенный)
FROM alpine:3.18

# Установка только необходимых пакетов
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Импорт пользователя и группы из builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Копирование SSL сертификатов
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копирование временных зон
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копирование скомпилированного приложения
COPY --from=builder /app/news-parsing-service /news-parsing-service

# Копирование конфигурационных файлов
COPY --from=builder /app/config /config

# Создание директории для логов и домашней директории пользователя
RUN mkdir -p /var/log /home/appuser && chown -R appuser:appuser /var/log /home/appuser

# Установка переменных окружения
ENV APP_ENV=production
ENV TZ=UTC

# Создание пользователя
USER appuser

# Открытие портов
EXPOSE 8081 8091 9091

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/news-parsing-service", "health"] || exit 1

# Команда запуска
ENTRYPOINT ["/news-parsing-service"]
