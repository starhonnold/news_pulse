# Pulse Service

Сервис управления персонализированными пульсами для проекта "Пульс Новостей". Позволяет пользователям создавать настраиваемые новостные ленты на основе выбранных источников и категорий.

## Функциональность

- **Управление пульсами** - создание, редактирование, удаление пользовательских пульсов
- **Персонализированные ленты** - генерация новостных лент на основе критериев пульса
- **Гибкая фильтрация** - по источникам, категориям, датам, релевантности
- **Умная сортировка** - по времени, релевантности, персональному скору
- **In-memory кеширование** для высокой производительности
- **Автоматическое обновление** пульсов по расписанию
- **Статистика и аналитика** по пульсам и новостям
- **REST API** для интеграции с фронтендом

## Архитектура

```
├── cmd/
│   └── main.go                 # Точка входа приложения
├── internal/
│   ├── cache/                  # In-memory кеширование
│   ├── config/                 # Конфигурация
│   ├── database/               # Работа с БД
│   ├── handlers/               # HTTP обработчики
│   ├── models/                 # Модели данных
│   ├── repository/             # Слой данных
│   └── services/               # Бизнес-логика
├── config/
│   └── config.yaml            # Конфигурационный файл
├── Dockerfile                 # Docker образ
└── go.mod                     # Go модули
```

## API Endpoints

### Управление пульсами

#### Создать новый пульс
```
POST /api/pulses
```

**Тело запроса:**
```json
{
  "name": "Технологии и ИИ",
  "description": "Новости о технологиях и искусственном интеллекте",
  "refresh_interval_min": 30,
  "source_ids": [1, 2, 3],
  "category_ids": [4, 6],
  "is_active": true,
  "is_default": false
}
```

#### Получить список пульсов пользователя
```
GET /api/pulses?page=1&page_size=10&is_active=true
```

**Параметры фильтрации:**
- `is_active` - только активные пульсы (true/false)
- `is_default` - только дефолтные пульсы (true/false)
- `keywords` - поиск по названию и описанию
- `created_from` - дата создания от (YYYY-MM-DD)
- `created_to` - дата создания до (YYYY-MM-DD)
- `page` - номер страницы
- `page_size` - размер страницы
- `sort_by` - поле сортировки (created_at, updated_at, name)
- `sort_order` - порядок сортировки (asc, desc)

#### Получить пульс по ID
```
GET /api/pulses/{id}
```

#### Получить дефолтный пульс пользователя
```
GET /api/pulses/default
```

#### Обновить пульс
```
PUT /api/pulses/{id}
```

#### Удалить пульс
```
DELETE /api/pulses/{id}
```

### Персонализированные ленты

#### Получить персонализированную ленту
```
GET /api/pulses/{id}/feed?page=1&page_size=20
```

**Параметры фильтрации:**
- `page` - номер страницы
- `page_size` - размер страницы
- `date_from` - дата от (YYYY-MM-DD)
- `date_to` - дата до (YYYY-MM-DD)
- `min_score` - минимальный скор релевантности (0.0-1.0)
- `sort_by` - поле сортировки (published_at, relevance_score, personal_score)
- `sort_order` - порядок сортировки (asc, desc)

#### Получить последние новости пульса
```
GET /api/pulses/{id}/feed/latest?limit=20
```

#### Получить трендовые новости пульса
```
GET /api/pulses/{id}/feed/trending?limit=20
```

### Административные функции

#### Очистить кеш
```
POST /api/cache/clear
```

#### Health Check
```
GET /health
```

## Установка и запуск

### Локальная разработка

1. **Установка зависимостей:**
```bash
go mod download
```

2. **Настройка конфигурации:**
```bash
cp config/config.yaml config/config.local.yaml
# Отредактируйте config.local.yaml под ваши нужды
```

3. **Запуск сервиса:**
```bash
go run cmd/main.go
```

### Docker

1. **Сборка образа:**
```bash
docker build -t pulse-service .
```

2. **Запуск контейнера:**
```bash
docker run -p 8083:8083 \
  -e POSTGRES_HOST=host.docker.internal \
  -e POSTGRES_PASSWORD=your_password \
  -e X-User-ID=1 \
  pulse-service
```

### Docker Compose

Сервис уже добавлен в `docker-compose.yml`:

```yaml
services:
  pulse-service:
    build: ./pulse-service
    ports:
      - "8083:8083"
      - "8093:8093"  # Health check
      - "9093:9093"  # Metrics
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - API_MAX_PULSES_PER_USER=10
    depends_on:
      - postgres
```

## Конфигурация

### Основные параметры

```yaml
server:
  port: 8083
  host: "0.0.0.0"

database:
  host: "postgres"
  port: 5432
  user: "news_pulse_user"
  password: "news_pulse_secure_password_2024"
  dbname: "news_pulse"

api:
  max_pulses_per_user: 10           # Максимум пульсов на пользователя
  max_sources_per_pulse: 50         # Максимум источников в пульсе
  max_categories_per_pulse: 10      # Максимум категорий в пульсе
  max_news_per_feed: 100           # Максимум новостей в ленте
  default_feed_page_size: 20       # Размер страницы ленты по умолчанию

pulse:
  min_refresh_interval: 5          # Минимальный интервал обновления (мин)
  max_refresh_interval: 1440       # Максимальный интервал обновления (мин)
  default_refresh_interval: 30     # Интервал обновления по умолчанию (мин)

caching:
  enabled: true                    # Включить кеширование
  user_pulses_ttl: 600            # TTL для пульсов пользователя (сек)
  personalized_feed_ttl: 300      # TTL для персонализированной ленты (сек)
  max_size: 5000                  # Максимальный размер кеша
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `APP_ENV` | Окружение (development/production) | development |
| `APP_PORT` | Порт HTTP сервера | 8083 |
| `POSTGRES_HOST` | Хост PostgreSQL | localhost |
| `POSTGRES_PASSWORD` | Пароль PostgreSQL | - |
| `API_MAX_PULSES_PER_USER` | Максимум пульсов на пользователя | 10 |
| `API_MAX_NEWS_PER_FEED` | Максимум новостей в ленте | 100 |
| `PULSE_DEFAULT_REFRESH_INTERVAL` | Интервал обновления по умолчанию | 30 |
| `CACHE_ENABLED` | Включить кеширование | true |
| `LOG_LEVEL` | Уровень логирования | info |

## Модели данных

### UserPulse - Пульс пользователя
```json
{
  "id": 1,
  "user_id": 123,
  "name": "Технологии и ИИ",
  "description": "Новости о технологиях и искусственном интеллекте",
  "refresh_interval_min": 30,
  "is_active": true,
  "is_default": false,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T12:45:00Z",
  "last_refreshed_at": "2024-01-15T12:30:00Z",
  "sources": [
    {
      "source_id": 1,
      "source_name": "TechCrunch",
      "source_domain": "techcrunch.com"
    }
  ],
  "categories": [
    {
      "category_id": 4,
      "category_name": "Технологии",
      "category_slug": "tech"
    }
  ],
  "news_count": 150
}
```

### PersonalizedFeed - Персонализированная лента
```json
{
  "pulse_id": 1,
  "pulse_name": "Технологии и ИИ",
  "news": [
    {
      "id": 12345,
      "title": "OpenAI представила новую модель GPT-5",
      "description": "Революционные возможности ИИ...",
      "url": "https://example.com/news/12345",
      "published_at": "2024-01-15T14:30:00Z",
      "source_name": "TechCrunch",
      "category_name": "Технологии",
      "relevance_score": 0.95,
      "personal_score": 0.89,
      "match_reason": "источник: TechCrunch, категория: Технологии"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  },
  "generated_at": "2024-01-15T15:00:00Z",
  "stats": {
    "total_news": 20,
    "news_sources": 3,
    "news_categories": 2,
    "average_score": 0.82
  }
}
```

## Примеры использования API

### Создание пульса

```bash
curl -X POST http://localhost:8083/api/pulses \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "name": "Российские технологии",
    "description": "Новости российского IT и технологий",
    "refresh_interval_min": 15,
    "source_ids": [1, 5, 8],
    "category_ids": [4, 6],
    "is_active": true
  }'
```

### Получение пульсов пользователя

```bash
curl http://localhost:8083/api/pulses?is_active=true&page=1&page_size=10 \
  -H "X-User-ID: 1"
```

### Получение персонализированной ленты

```bash
curl "http://localhost:8083/api/pulses/1/feed?page=1&page_size=20&sort_by=personal_score" \
  -H "X-User-ID: 1"
```

### Поиск пульсов

```bash
curl "http://localhost:8083/api/pulses?keywords=технологии&sort_by=updated_at&sort_order=desc" \
  -H "X-User-ID: 1"
```

## Персональный скор новостей

Сервис вычисляет персональный скор для каждой новости в ленте по формуле:

```
PersonalScore = BaseRelevance × FreshnessCoeff × PopularityCoeff
```

Где:
- **BaseRelevance** - базовая релевантность новости (0.0-1.0)
- **FreshnessCoeff** - коэффициент свежести:
  - ≤ 1 час: 1.2
  - ≤ 6 часов: 1.1
  - ≤ 24 часа: 1.0
  - ≤ 72 часа: 0.9
  - > 72 часов: 0.8
- **PopularityCoeff** - коэффициент популярности:
  - > 1000 просмотров: 1.2
  - > 500 просмотров: 1.1
  - > 100 просмотров: 1.05
  - ≤ 100 просмотров: 1.0

## Кеширование

Сервис использует многоуровневое кеширование:

- **Пульсы пользователя**: 10 минут
- **Персонализированные ленты**: 5 минут  
- **Статистика пульсов**: 15 минут
- **Дефолтные пульсы**: 10 минут
- **Последние/трендовые новости**: 5 минут

### Инвалидация кеша

Кеш автоматически инвалидируется при:
- Создании/обновлении/удалении пульса
- Изменении источников или категорий пульса
- Очистке кеша через API

```bash
# Очистить весь кеш
curl -X POST http://localhost:8083/api/cache/clear
```

## Мониторинг

### Health Check

```bash
curl http://localhost:8093/health
```

Ответ содержит:
- Статус сервиса
- Время работы
- Статистику кеша
- Информацию о подключении к БД
- Количество активных пульсов

### Метрики

Базовые метрики доступны на порту 9093:

```bash
curl http://localhost:9093/metrics
```

### Логирование

Все HTTP запросы логируются с подробной информацией:

```json
{
  "level": "info",
  "msg": "HTTP request",
  "method": "GET",
  "url": "/api/pulses/1/feed?page=1",
  "status_code": 200,
  "duration": "125.3ms",
  "user_id": 1,
  "time": "2024-01-15T15:30:00Z"
}
```

## Ограничения и лимиты

### По умолчанию
- **Максимум пульсов на пользователя**: 10
- **Максимум источников в пульсе**: 50
- **Максимум категорий в пульсе**: 10
- **Максимум новостей в ленте**: 100
- **Минимальный интервал обновления**: 5 минут
- **Максимальный интервал обновления**: 24 часа

### Размеры данных
- **Название пульса**: до 100 символов
- **Описание пульса**: до 500 символов
- **Поисковый запрос**: до 100 символов

## Безопасность

- Запуск от непривилегированного пользователя
- Валидация всех входящих данных
- Проверка принадлежности пульсов пользователю
- Защита от SQL injection через prepared statements
- Ограничение количества пульсов на пользователя
- Таймауты для предотвращения DoS

## Интеграция с другими сервисами

### News Management Service
Получает новости для персонализированных лент из общей базы данных новостей.

### API Gateway
Все запросы проходят через API Gateway для аутентификации и авторизации.

### Frontend
Предоставляет полный API для управления пульсами и получения персонализированных лент.

## Разработка

### Добавление нового endpoint

1. Добавьте метод в `services/pulse_service.go`
2. Создайте handler в `handlers/handlers.go`
3. Зарегистрируйте маршрут в `SetupRoutes()`
4. Обновите документацию

### Структура репозиториев

```
internal/repository/
├── pulse_repository.go     # Управление пульсами
└── feed_repository.go      # Персонализированные ленты
```

### Модели валидации

Все входящие данные проходят валидацию:

```go
func (r *PulseRequest) Validate() error {
    if len(r.Name) == 0 || len(r.Name) > 100 {
        return fmt.Errorf("pulse name must be 1-100 characters")
    }
    // ... другие проверки
}
```

## Troubleshooting

### Частые проблемы

1. **Медленная генерация лент**
   - Проверьте индексы БД
   - Включите кеширование
   - Уменьшите размер страницы

2. **Ошибки "Pulse not found"**
   - Проверьте X-User-ID заголовок
   - Убедитесь, что пульс принадлежит пользователю
   - Проверьте активность пульса

3. **Превышение лимитов**
   - Проверьте количество пульсов пользователя
   - Уменьшите количество источников в пульсе
   - Настройте лимиты в конфигурации

### Логи для диагностики

```bash
# Просмотр логов в Docker
docker logs news_pulse_pulse_service

# Ошибки создания пульсов
docker logs news_pulse_pulse_service | grep "Failed to create pulse"

# Медленные запросы
docker logs news_pulse_pulse_service | grep "duration.*[5-9][0-9][0-9]ms"
```

## Производительность

### Оптимизации

- **Специализированные индексы** для быстрого поиска пульсов и новостей
- **In-memory кеширование** часто запрашиваемых данных
- **Пагинация** для ограничения размера ответов
- **Ленивая загрузка** связанных данных
- **Батчинг** операций с БД

### Рекомендации по масштабированию

1. **Горизонтальное масштабирование**: Несколько экземпляров за load balancer
2. **Read replicas**: Использование read-only реплик для чтения
3. **Распределенный кеш**: Переход на Redis для кеширования
4. **Партиционирование**: Разделение данных по пользователям

## Лицензия

MIT License - см. файл LICENSE для деталей.
