# База данных News Pulse

Этот каталог содержит всё необходимое для настройки и управления базой данных PostgreSQL для проекта "Пульс Новостей".

## Структура файлов

```
database/
├── README.md              # Этот файл с инструкциями
├── schema.sql            # Схема базы данных (таблицы, индексы, функции)
├── seed_data.sql         # Начальные данные (страны, источники, категории)
├── postgresql.conf       # Конфигурация PostgreSQL
├── pgadmin_servers.json  # Предконфигурированные серверы для pgAdmin
└── redis.conf           # Конфигурация Redis для кеширования
```

## Быстрый старт

### 1. Запуск базы данных через Docker Compose

```bash
# Перейти в каталог backend
cd backend

# Скопировать файл переменных окружения
cp env.example .env

# Запустить только PostgreSQL
docker-compose up -d postgres

# Или запустить с pgAdmin для разработки
docker-compose --profile dev up -d

# Или запустить с Redis для кеширования
docker-compose --profile cache up -d
```

### 2. Проверка работоспособности

```bash
# Проверить статус контейнеров
docker-compose ps

# Проверить логи PostgreSQL
docker-compose logs postgres

# Подключиться к базе данных
docker-compose exec postgres psql -U news_pulse_user -d news_pulse
```

### 3. Доступ к pgAdmin (в dev режиме)

- URL: http://localhost:8080
- Email: admin@newspulse.local
- Password: admin123

## Схема базы данных

### Основные таблицы

#### `countries` - Страны
Содержит список стран с их кодами и флагами для фильтрации новостей.

#### `categories` - Категории новостей
Предопределенные категории новостей (политика, экономика, спорт, и т.д.).

#### `news_sources` - Источники новостей
RSS ленты новостных сайтов с информацией о парсинге.

#### `news` - Новости
Основная таблица с новостями, спарсенными из RSS лент.

#### `tags` - Теги
Теги для классификации новостей.

#### `users` - Пользователи
Пользователи системы (для будущей SMS авторизации).

#### `user_pulses` - Пульсы пользователей
Персонализированные новостные ленты пользователей.

### Связующие таблицы

- `news_tags` - связь новостей с тегами
- `pulse_countries` - связь пульсов со странами
- `pulse_sources` - связь пульсов с источниками
- `pulse_categories` - связь пульсов с категориями

## Начальные данные

После инициализации база данных содержит:

- **12 стран** СНГ и близлежащих регионов
- **12 категорий** новостей
- **30+ источников** RSS лент
- **20 базовых тегов**
- **Тестовый пользователь** для разработки
- **Тестовый пульс** "Технологические новости"

## Источники новостей по странам

### Россия (7 источников)
- РИА Новости, ТАСС, Интерфакс
- Lenta.ru, Газета.Ru, РБК
- Коммерсант, Ведомости, RT, Sputnik

### Беларусь (3 источника)
- БЕЛТА, Советская Беларусь, Белорусские новости

### Казахстан (3 источника)
- Казинформ, Tengrinews, Nur.kz

### Другие страны СНГ
- По 2-3 источника для каждой страны

## Оптимизация производительности

### Индексы
- Полнотекстовый поиск по заголовкам и содержимому новостей
- Индексы по датам публикации и релевантности
- Составные индексы для фильтрации

### Конфигурация PostgreSQL
- Оптимизировано для 8GB сервера
- 512MB shared_buffers
- Настроен русскоязычный полнотекстовый поиск
- Автоматический VACUUM для частых INSERT операций

### Кеширование (Redis)
- Кеширование популярных запросов
- TTL для списков новостей
- Кеширование метаданных источников

## Функции и триггеры

### `count_pulse_news(pulse_uuid)`
Подсчитывает количество новостей, соответствующих критериям пульса.

### `cleanup_expired_sms_codes()`
Очищает истекшие SMS коды (для будущей авторизации).

### Автоматические триггеры
- Обновление `updated_at` при изменении записей
- Логирование изменений

## Мониторинг

### Таблица `parsing_logs`
Логирует результаты парсинга RSS лент:
- Статус парсинга (success/error/timeout)
- Количество спарсенных новостей
- Время выполнения
- Сообщения об ошибках

### Полезные запросы для мониторинга

```sql
-- Статистика по источникам
SELECT 
    ns.name,
    COUNT(n.id) as news_count,
    MAX(n.published_at) as latest_news,
    ns.last_parsed_at
FROM news_sources ns
LEFT JOIN news n ON ns.id = n.source_id
WHERE ns.is_active = true
GROUP BY ns.id, ns.name, ns.last_parsed_at
ORDER BY news_count DESC;

-- Ошибки парсинга за последние 24 часа
SELECT 
    ns.name,
    pl.status,
    pl.error_message,
    pl.created_at
FROM parsing_logs pl
JOIN news_sources ns ON pl.source_id = ns.id
WHERE pl.created_at > NOW() - INTERVAL '24 hours'
AND pl.status != 'success'
ORDER BY pl.created_at DESC;

-- Топ категорий по количеству новостей
SELECT 
    c.name,
    COUNT(n.id) as news_count
FROM categories c
LEFT JOIN news n ON c.id = n.category_id
GROUP BY c.id, c.name
ORDER BY news_count DESC;
```

## Резервное копирование

### Автоматическое резервное копирование
```bash
# Создать резервную копию
docker-compose exec postgres pg_dump -U news_pulse_user news_pulse > backup_$(date +%Y%m%d_%H%M%S).sql

# Восстановить из резервной копии
docker-compose exec -T postgres psql -U news_pulse_user news_pulse < backup_file.sql
```

### Скрипт для регулярного резервного копирования
```bash
#!/bin/bash
# backup_database.sh
BACKUP_DIR="/path/to/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
docker-compose exec postgres pg_dump -U news_pulse_user news_pulse > "${BACKUP_DIR}/news_pulse_${TIMESTAMP}.sql"
# Удалить резервные копии старше 7 дней
find "${BACKUP_DIR}" -name "news_pulse_*.sql" -mtime +7 -delete
```

## Миграции

При изменении схемы базы данных создавайте файлы миграций:

```
migrations/
├── 001_initial_schema.sql
├── 002_add_user_preferences.sql
└── 003_optimize_indexes.sql
```

## Troubleshooting

### Проблемы с подключением
```bash
# Проверить статус PostgreSQL
docker-compose exec postgres pg_isready -U news_pulse_user

# Проверить логи
docker-compose logs postgres
```

### Проблемы с производительностью
```sql
-- Проверить медленные запросы
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Проверить размер таблиц
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Очистка старых данных
```sql
-- Удалить новости старше 90 дней
DELETE FROM news 
WHERE published_at < NOW() - INTERVAL '90 days';

-- Очистить старые логи парсинга
DELETE FROM parsing_logs 
WHERE created_at < NOW() - INTERVAL '30 days';
```

## Безопасность

### Production настройки
1. Изменить пароли в `.env` файле
2. Включить SSL соединения
3. Настроить файрволл для ограничения доступа
4. Регулярно обновлять PostgreSQL
5. Мониторить подозрительную активность

### Рекомендуемые настройки для production
```bash
# В .env файле
POSTGRES_PASSWORD=очень_сложный_пароль_123!@#
PGADMIN_PASSWORD=другой_сложный_пароль_456$%^
```

## Контакты

При возникновении проблем с базой данных обращайтесь к команде разработки или создавайте issue в репозитории проекта.
