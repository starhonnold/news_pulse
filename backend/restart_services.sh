#!/bin/bash

echo "🔄 Перезапуск сервисов News Pulse..."

# Останавливаем все сервисы
echo "⏹️ Останавливаем сервисы..."
docker-compose down

# Пересобираем образы с изменениями
echo "🔨 Пересобираем образы..."
docker-compose build --no-cache

# Запускаем сервисы заново
echo "🚀 Запускаем сервисы..."
docker-compose up -d

# Ждем запуска
echo "⏳ Ждем запуска сервисов..."
sleep 10

# Проверяем статус
echo "✅ Проверяем статус сервисов..."
docker-compose ps

echo "🎉 Сервисы перезапущены!"
echo "📊 API Gateway: http://localhost:8080"
echo "📰 News Management: http://localhost:8082"
echo "🔍 News Parsing: http://localhost:8081"
echo "💓 Pulse Service: http://localhost:8083"
