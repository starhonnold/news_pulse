# API Gateway

Центральный шлюз для проекта "Пульс Новостей". Обеспечивает единую точку входа для всех запросов к микросервисам с аутентификацией, авторизацией, rate limiting и WebSocket поддержкой.

## Основные функции

- **🚀 Маршрутизация** - прокси запросов к микросервисам
- **🔐 Аутентификация** - JWT токены с refresh механизмом
- **⚡ Rate Limiting** - защита от злоупотреблений
- **🌐 CORS** - поддержка cross-origin запросов
- **🔌 WebSocket** - real-time соединения
- **📊 Мониторинг** - health checks и метрики
- **🛡️ Безопасность** - заголовки безопасности

## Архитектура

```
api-gateway/
├── cmd/main.go                    # Точка входа
├── internal/
│   ├── config/                    # Конфигурация
│   ├── handlers/                  # HTTP обработчики
│   ├── middleware/                # Middleware (auth, rate limit, CORS)
│   ├── models/                    # Модели данных
│   └── services/                  # Прокси и WebSocket сервисы
├── config/config.yaml             # Конфигурация
└── Dockerfile                     # Docker образ
```

## API Endpoints

### Аутентификация
```
POST /api/auth/login       # Вход
POST /api/auth/register    # Регистрация
POST /api/auth/refresh     # Обновление токена
```

### Прокси к микросервисам
```
/api/news/*               → News Management Service (8082)
/api/news/parse/*         → News Parsing Service (8081)
/api/pulses/*             → Pulse Service (8083)
```

### Системные
```
GET  /health              # Health check
GET  /api/stats           # Статистика
WS   /ws                  # WebSocket соединения
GET  /metrics             # Prometheus метрики
```

## Быстрый старт

### Docker Compose (рекомендуемый)
```bash
cd backend
docker-compose up api-gateway
```

### Локальная разработка
```bash
cd backend/api-gateway
go mod download
go run cmd/main.go
```

## Конфигурация

### Основные переменные окружения
```bash
# Сервисы
NEWS_PARSING_SERVICE_URL=http://news-parsing-service:8081
NEWS_MANAGEMENT_SERVICE_URL=http://news-management-service:8082
PULSE_SERVICE_URL=http://pulse-service:8083

# Аутентификация
JWT_SECRET=your-secret-key
AUTH_ENABLED=true

# Функции
RATE_LIMITING_ENABLED=true
CORS_ENABLED=true
WEBSOCKET_ENABLED=true
```

## Использование

### Аутентификация
```bash
# Регистрация
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"user","email":"user@example.com","password":"password123"}'

# Вход
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"password123"}'

# Использование токена
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/news
```

### WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

// Отправка ping
ws.send(JSON.stringify({
  type: 'ping',
  data: { timestamp: Date.now() }
}));
```

## Маршрутизация

API Gateway автоматически маршрутизирует запросы:

| Путь | Сервис | Порт |
|------|--------|------|
| `/api/news` (кроме parse) | News Management | 8082 |
| `/api/news/parse`, `/api/parsing` | News Parsing | 8081 |
| `/api/pulses`, `/api/feeds` | Pulse Service | 8083 |

## Middleware

### Порядок выполнения
1. **Request ID** - уникальный ID для каждого запроса
2. **Recovery** - восстановление после panic
3. **Security Headers** - заголовки безопасности
4. **CORS** - cross-origin поддержка
5. **Logging** - логирование запросов
6. **Rate Limiting** - ограничение запросов
7. **Authentication** - проверка JWT токенов

### Rate Limiting
- **Глобальный лимит**: 1000 запросов/мин
- **На пользователя**: 60 запросов/мин
- **Анонимные**: 10 запросов/мин
- **Белый список IP** для исключений

## Мониторинг

### Health Check
```bash
curl http://localhost:8090/health
```

### Метрики
```bash
curl http://localhost:9090/metrics
```

### Логи
Структурированные JSON логи со всеми HTTP запросами:
```json
{
  "level": "info",
  "msg": "HTTP request completed",
  "method": "GET",
  "path": "/api/news",
  "status_code": 200,
  "duration_ms": 45,
  "user_id": 1,
  "request_id": "abc123"
}
```

## Безопасность

- JWT аутентификация с refresh токенами
- Rate limiting по IP и пользователям
- CORS с настраиваемыми origins
- Security headers (HSTS, CSP, XSS Protection)
- Валидация всех входящих данных

## Производительность

- Reverse proxy с connection pooling
- In-memory rate limiting
- Graceful shutdown
- Health checks для всех сервисов
- Метрики для мониторинга

## Разработка

### Добавление нового микросервиса
1. Добавьте конфигурацию в `config.yaml`
2. Обновите `determineTargetService()` в `proxy.go`
3. Добавьте health check endpoint

### Тестирование
```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```

## Docker

### Сборка
```bash
docker build -t api-gateway .
```

### Запуск
```bash
docker run -p 8080:8080 \
  -e NEWS_MANAGEMENT_SERVICE_URL=http://host.docker.internal:8082 \
  -e AUTH_ENABLED=false \
  api-gateway
```

## Лицензия

MIT License
