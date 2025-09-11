# News Management Service

Сервис управления новостями для проекта "Пульс Новостей". Предоставляет REST API для получения, фильтрации и поиска новостей, а также управления справочными данными.

## Функциональность

- **CRUD операции** с новостями
- **Расширенная фильтрация** по источникам, категориям, странам, датам
- **Полнотекстовый поиск** по заголовкам и описаниям новостей
- **Пагинация** результатов с настраиваемым размером страницы
- **In-memory кеширование** для повышения производительности
- **Статистика** по новостям, категориям, источникам
- **API для фронтенда** с CORS поддержкой
- **Health checks** и метрики

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

### Новости

#### Получить список новостей
```
GET /api/news?page=1&page_size=20&sort_by=published_at&sort_order=desc
```

**Параметры фильтрации:**
- `sources` - ID источников через запятую (1,2,3)
- `categories` - ID категорий через запятую (1,2,3)  
- `countries` - ID стран через запятую (1,2,3)
- `keywords` - ключевые слова для поиска
- `date_from` - дата от (YYYY-MM-DD)
- `date_to` - дата до (YYYY-MM-DD)
- `min_relevance` - минимальная релевантность (0.0-1.0)
- `page` - номер страницы (по умолчанию 1)
- `page_size` - размер страницы (по умолчанию 20)
- `sort_by` - поле сортировки (published_at, relevance_score, view_count)
- `sort_order` - порядок сортировки (asc, desc)

#### Получить новость по ID
```
GET /api/news/{id}
```

#### Получить последние новости
```
GET /api/news/latest?limit=20
```

#### Получить трендовые новости
```
GET /api/news/trending?limit=20
```

#### Поиск новостей
```
GET /api/news/search?q=технологии&page=1&page_size=20
```

### Справочные данные

#### Получить категории
```
GET /api/categories
```

#### Получить источники новостей
```
GET /api/sources
```

#### Получить страны
```
GET /api/countries
```

### Статистика

#### Получить общую статистику
```
GET /api/stats
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
docker build -t news-management-service .
```

2. **Запуск контейнера:**
```bash
docker run -p 8082:8082 \
  -e POSTGRES_HOST=host.docker.internal \
  -e POSTGRES_PASSWORD=your_password \
  news-management-service
```

### Docker Compose

Сервис уже добавлен в `docker-compose.yml`:

```yaml
services:
  news-management-service:
    build: ./news-management-service
    ports:
      - "8082:8082"
      - "8092:8092"  # Health check
      - "9092:9092"  # Metrics
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    depends_on:
      - postgres
```

## Конфигурация

### Основные параметры

```yaml
server:
  port: 8082
  host: "0.0.0.0"

database:
  host: "postgres"
  port: 5432
  user: "news_pulse_user"
  password: "news_pulse_secure_password_2024"
  dbname: "news_pulse"

api:
  max_page_size: 100           # Максимальный размер страницы
  default_page_size: 20        # Размер страницы по умолчанию
  max_search_length: 500       # Максимальная длина поискового запроса
  enable_fulltext_search: true # Включить полнотекстовый поиск

caching:
  enabled: true                # Включить кеширование
  news_ttl: 300               # TTL для кеша новостей (секунды)
  categories_ttl: 3600        # TTL для кеша категорий
  max_size: 10000             # Максимальный размер кеша
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `APP_ENV` | Окружение (development/production) | development |
| `APP_PORT` | Порт HTTP сервера | 8082 |
| `POSTGRES_HOST` | Хост PostgreSQL | localhost |
| `POSTGRES_PASSWORD` | Пароль PostgreSQL | - |
| `API_DEFAULT_PAGE_SIZE` | Размер страницы по умолчанию | 20 |
| `API_MAX_PAGE_SIZE` | Максимальный размер страницы | 100 |
| `CACHE_ENABLED` | Включить кеширование | true |
| `LOG_LEVEL` | Уровень логирования | info |

## Примеры использования API

### Получение новостей с фильтрацией

```bash
# Новости из России и Беларуси по категории "Технологии"
curl "http://localhost:8082/api/news?countries=1,2&categories=4&page_size=10"

# Поиск новостей за последнюю неделю
curl "http://localhost:8082/api/news?date_from=2024-01-08&keywords=искусственный интеллект"

# Трендовые новости с высокой релевантностью
curl "http://localhost:8082/api/news?min_relevance=0.8&sort_by=view_count&sort_order=desc"
```

### Полнотекстовый поиск

```bash
# Поиск по ключевым словам
curl "http://localhost:8082/api/news/search?q=блокчейн криптовалюта&page=1&page_size=20"

# Поиск с фразой
curl "http://localhost:8082/api/news/search?q=\"машинное обучение\""
```

### Получение справочных данных

```bash
# Все категории
curl "http://localhost:8082/api/categories"

# Все источники новостей
curl "http://localhost:8082/api/sources"

# Все страны
curl "http://localhost:8082/api/countries"
```

## Кеширование

Сервис использует in-memory кеш для повышения производительности:

- **Новости**: кешируются на 5 минут
- **Категории**: кешируются на 1 час
- **Источники**: кешируются на 1 час
- **Страны**: кешируются на 1 час
- **Результаты поиска**: кешируются на 5 минут

### Управление кешем

```bash
# Получить статистику кеша
curl "http://localhost:8082/health"

# Очистить кеш
curl -X POST "http://localhost:8082/api/cache/clear"
```

## Мониторинг

### Health Check

```bash
curl http://localhost:8092/health
```

Ответ содержит:
- Статус сервиса
- Время работы
- Статистику кеша
- Информацию о подключении к БД

### Метрики

Базовые метрики доступны на порту 9092:

```bash
curl http://localhost:9092/metrics
```

### Логирование

Сервис поддерживает структурированное логирование:

```json
{
  "level": "info",
  "msg": "HTTP request",
  "method": "GET",
  "url": "/api/news?page=1",
  "status_code": 200,
  "duration": "45.2ms",
  "time": "2024-01-15T10:30:00Z"
}
```

## Производительность

### Оптимизации

- **In-memory кеширование** часто запрашиваемых данных
- **Connection pooling** для БД (до 25 соединений)
- **Индексы БД** для быстрого поиска и фильтрации
- **Пагинация** для ограничения размера ответов
- **Полнотекстовый поиск** PostgreSQL с русскоязычной поддержкой

### Ограничения ресурсов

- **Память**: ~512MB
- **CPU**: Умеренное использование
- **БД соединения**: До 25 одновременных
- **Размер страницы**: Максимум 100 элементов

## Разработка

### Структура проекта

```
internal/
├── cache/           # In-memory кеширование
├── config/          # Конфигурация приложения
├── database/        # Подключение к БД
├── handlers/        # HTTP обработчики
├── models/          # Структуры данных
├── repository/      # Слой доступа к данным
└── services/        # Бизнес-логика
```

### Добавление нового endpoint

1. Добавьте метод в `services/news_service.go`
2. Создайте handler в `handlers/handlers.go`
3. Зарегистрируйте маршрут в `SetupRoutes()`
4. Обновите документацию

### Тестирование

```bash
# Запуск тестов
go test ./...

# Тесты с покрытием
go test -cover ./...

# Линтер
golangci-lint run
```

## Интеграция с другими сервисами

### News Parsing Service

Сервис читает новости, которые парсит News Parsing Service из PostgreSQL базы данных.

### API Gateway

Все запросы к сервису должны проходить через API Gateway для:
- Аутентификации и авторизации
- Rate limiting
- Логирования запросов
- Load balancing

### Frontend

Предоставляет API для фронтенда с поддержкой:
- CORS заголовков
- JSON ответов
- Стандартизированных ошибок
- Пагинации

## Безопасность

- Запуск от непривилегированного пользователя
- Валидация всех входящих параметров
- Защита от SQL injection через prepared statements
- Ограничение размера запросов и ответов
- Таймауты для предотвращения DoS

## Troubleshooting

### Частые проблемы

1. **Медленные запросы**
   - Проверьте индексы в БД
   - Включите кеширование
   - Уменьшите размер страницы

2. **Ошибки подключения к БД**
   - Проверьте параметры подключения
   - Убедитесь, что PostgreSQL доступен
   - Проверьте количество соединений

3. **Проблемы с поиском**
   - Убедитесь, что установлено расширение pg_trgm
   - Проверьте настройки полнотекстового поиска
   - Валидируйте поисковые запросы

### Логи для диагностики

```bash
# Просмотр логов в Docker
docker logs news_pulse_management_service

# Логи API запросов
docker logs news_pulse_management_service | grep "HTTP request"

# Ошибки
docker logs news_pulse_management_service | grep "error"
```

## Масштабирование

Для увеличения производительности:

1. **Горизонтальное масштабирование**: Запуск нескольких экземпляров за load balancer
2. **Кеширование**: Добавление Redis для распределенного кеша
3. **Read replicas**: Использование read-only реплик PostgreSQL
4. **CDN**: Кеширование статических ответов через CDN

## Лицензия

MIT License - см. файл LICENSE для деталей.
