# Notification Service

Сервис уведомлений для проекта "Пульс Новостей". Обеспечивает создание, доставку и управление уведомлениями пользователей через различные каналы (WebSocket, email, push, SMS).

## Основные функции

- **📨 Управление уведомлениями** - CRUD операции с уведомлениями
- **🔌 Real-time доставка** - WebSocket соединение с API Gateway
- **📊 Статистика** - аналитика по уведомлениям пользователей
- **🎯 Типизированные уведомления** - новости, пульсы, системные сообщения
- **⏰ Автоматическая очистка** - удаление старых и истекших уведомлений
- **🔄 Event-driven архитектура** - асинхронная обработка событий

## Архитектура

```
notification-service/
├── cmd/main.go                    # Точка входа
├── internal/
│   ├── config/                    # Конфигурация
│   ├── database/                  # База данных и миграции
│   ├── handlers/                  # HTTP API handlers
│   ├── models/                    # Модели данных
│   ├── repository/                # Data access layer
│   └── services/                  # Бизнес-логика
│       ├── notification_service.go  # Основной сервис
│       ├── websocket_service.go     # WebSocket клиент
│       └── event_processor.go       # Обработчик событий
├── config/config.yaml             # Конфигурация
└── Dockerfile                     # Docker образ
```

## Типы уведомлений

### 📰 Новостные оповещения (news_alert)
```json
{
  "type": "news_alert",
  "title": "Новая важная новость",
  "message": "Заголовок новости\n\nКраткое содержание",
  "data": {
    "news_id": 123,
    "news_url": "https://...",
    "source_name": "РИА Новости",
    "category": "Политика"
  }
}
```

### 🔔 Обновления пульсов (pulse_update)
```json
{
  "type": "pulse_update", 
  "title": "Обновление пульса \"Технологии\"",
  "message": "Найдено 5 новых новостей по вашим критериям",
  "data": {
    "pulse_id": 456,
    "pulse_name": "Технологии",
    "news_count": 5,
    "update_type": "new_news"
  }
}
```

### ⚙️ Системные сообщения (system_message)
```json
{
  "type": "system_message",
  "title": "Системное уведомление", 
  "message": "Запланированное обслуживание сервиса",
  "data": {
    "message_type": "maintenance",
    "priority": "high",
    "action_url": "/maintenance"
  }
}
```

## API Endpoints

### Создание уведомлений
```
POST /api/notifications                    # Общее создание
POST /api/notifications/news-alert         # Новостное оповещение
POST /api/notifications/pulse-update       # Обновление пульса
POST /api/notifications/system-message     # Системное сообщение
```

### Управление уведомлениями
```
GET    /api/notifications/{id}             # Получить уведомление
PATCH  /api/notifications/{id}             # Пометить как прочитанное
DELETE /api/notifications/{id}             # Удалить уведомление
```

### Пользовательские уведомления
```
GET   /api/users/{id}/notifications        # Список уведомлений
GET   /api/users/{id}/notifications/stats  # Статистика
GET   /api/users/{id}/notifications/unread-count  # Количество непрочитанных
PATCH /api/users/{id}/notifications/mark-all-read # Пометить все как прочитанные
```

### Системные
```
GET  /health              # Health check
GET  /api/stats           # Статистика сервиса
POST /api/test/websocket  # Тест WebSocket соединения
GET  /metrics             # Prometheus метрики
```

## Быстрый старт

### Docker Compose (рекомендуемый)
```bash
cd backend
docker-compose up notification-service
```

### Локальная разработка
```bash
cd backend/notification-service
go mod download
go run cmd/main.go
```

## Конфигурация

### Основные переменные окружения
```bash
# База данных
POSTGRES_HOST=postgres
POSTGRES_DB=news_pulse
POSTGRES_USER=news_pulse_user
POSTGRES_PASSWORD=your_password

# WebSocket
WEBSOCKET_GATEWAY_URL=ws://api-gateway:8080/ws

# Микросервисы
NEWS_MANAGEMENT_SERVICE_URL=http://news-management-service:8082
PULSE_SERVICE_URL=http://pulse-service:8083

# Логирование
LOG_LEVEL=info
```

### Настройки уведомлений
```yaml
notifications:
  max_notifications_per_user: 1000
  notification_ttl_days: 30
  auto_cleanup_enabled: true
  auto_cleanup_interval: "24h"
```

## Использование

### Создание новостного оповещения
```bash
curl -X POST http://localhost:8084/api/notifications/news-alert \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "data": {
      "news_id": 123,
      "title": "Важная новость",
      "summary": "Краткое содержание",
      "url": "https://example.com/news/123",
      "source_name": "РИА Новости",
      "category": "Политика",
      "published_at": "2024-01-15T10:00:00Z"
    }
  }'
```

### Получение уведомлений пользователя
```bash
curl "http://localhost:8084/api/users/1/notifications?page=1&page_size=20&is_read=false"
```

### Статистика пользователя
```bash
curl "http://localhost:8084/api/users/1/notifications/stats"
```

## WebSocket интеграция

Сервис автоматически подключается к API Gateway через WebSocket для real-time доставки уведомлений:

```yaml
websocket:
  gateway_url: "ws://api-gateway:8080/ws"
  reconnect_enabled: true
  reconnect_interval: 5s
  max_reconnect_attempts: 10
```

При создании уведомления оно автоматически отправляется через WebSocket:
```json
{
  "type": "notification_broadcast",
  "data": {
    "notification_type": "notification_created",
    "payload": { /* notification object */ },
    "user_id": 123,
    "timestamp": "2024-01-15T10:00:00Z"
  }
}
```

## База данных

### Таблицы
- **`notifications`** - основные уведомления
- **`user_notification_settings`** - настройки пользователей
- **`user_devices`** - устройства для push уведомлений
- **`notification_templates`** - шаблоны уведомлений
- **`notification_delivery_logs`** - логи доставки

### Автоматическая очистка
- Удаление истекших уведомлений (по `expires_at`)
- Удаление старых уведомлений (по `notification_ttl_days`)
- Регулярная очистка каждые 24 часа

## Event-driven архитектура

### Процессор событий
- **5 воркеров** для параллельной обработки
- **Retry механизм** с экспоненциальной задержкой
- **Буферизация событий** (1000 событий)
- **Таймауты обработки** (30 секунд)

### Создание события
```go
event := &models.NotificationEvent{
    Type:    "news_alert",
    UserID:  123,
    Title:   "Новость",
    Message: "Содержание",
    Data:    map[string]interface{}{"key": "value"},
}

err := notificationService.ProcessEvent(event)
```

## Мониторинг

### Health Check
```bash
curl http://localhost:8094/health
```

### Метрики
```bash
curl http://localhost:9094/metrics
```

### Статистика
- Общее количество уведомлений
- Непрочитанные уведомления  
- Статистика по типам
- Активность за последние 7 дней
- Статус WebSocket соединения

## Расширяемость

### Добавление нового типа уведомления
1. Добавьте тип в `models/models.go`
2. Создайте шаблон в `config.yaml`
3. Добавьте метод в `NotificationService`
4. Добавьте HTTP handler

### Добавление канала доставки
1. Создайте новый сервис (email, push, sms)
2. Интегрируйте с `NotificationService`
3. Добавьте конфигурацию
4. Обновите логи доставки

## Безопасность

- Валидация всех входящих данных
- Лимиты на количество уведомлений
- Автоматическое удаление старых данных
- Structured logging для аудита
- Health checks для мониторинга

## Производительность

- Асинхронная обработка событий
- Batch операции для массовых уведомлений
- Connection pooling для базы данных
- In-memory очередь сообщений
- Graceful shutdown

## Будущие возможности

### Push уведомления
- Firebase Cloud Messaging (FCM)
- Apple Push Notification Service (APNS)
- Web Push notifications

### Email уведомления
- SMTP интеграция
- HTML шаблоны
- Дайджесты новостей

### SMS уведомления
- Twilio интеграция
- SMS.RU поддержка
- Международные SMS

## Docker

### Сборка
```bash
docker build -t notification-service .
```

### Запуск
```bash
docker run -p 8084:8084 \
  -e POSTGRES_HOST=host.docker.internal \
  -e WEBSOCKET_GATEWAY_URL=ws://host.docker.internal:8080/ws \
  notification-service
```

## Лицензия

MIT License
