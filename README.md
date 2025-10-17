# News Pulse - Система персонализированных новостей

## 📋 Описание проекта

News Pulse - это современная система для сбора, анализа и персонализации новостей из различных источников. Система использует микросервисную архитектуру на Go и современный frontend на Quasar Framework.

### 🚀 Основные возможности

- **Автоматический парсинг** RSS лент из 200+ источников
- **AI классификация** новостей с помощью Google AI и Ollama
- **Персонализация** через пользовательские пульсы
- **Real-time уведомления** через WebSocket
- **Многоязычная поддержка** (русский, английский)
- **Адаптивный дизайн** для всех устройств

## 🏗️ Архитектура системы

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Host (8GB RAM)                   │
├─────────────────────────────────────────────────────────────┤
│  Frontend (Nginx)     │  Backend Services                  │
│  - Quasar Static      │  - API Gateway (8080)              │
│  - Nginx (80/443)     │  - News Parsing (8081)             │
│                       │  - News Management (8082)          │
│                       │  - Pulse Service (8083)            │
│                       │  - Notification Service (8084)     │
│                       │  - Ollama AI (11434)               │
├─────────────────────────────────────────────────────────────┤
│  Database & Storage                                        │
│  - PostgreSQL 15 (5432)                                    │
│  - Docker Volumes                                          │
└─────────────────────────────────────────────────────────────┘
```

## 📋 Требования к системе

### Минимальные требования

- **ОС**: Ubuntu 20.04+ / CentOS 8+ / Debian 11+
- **RAM**: 8GB (минимум 6GB свободной памяти)
- **CPU**: 4 ядра (рекомендуется 8 ядер)
- **Диск**: 50GB SSD (минимум 20GB свободного места)
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### Рекомендуемые требования

- **RAM**: 16GB+
- **CPU**: 8+ ядер
- **Диск**: 100GB+ SSD
- **GPU**: NVIDIA GPU для Ollama (опционально)

## 🚀 Быстрый старт

### 1. Клонирование репозитория

```bash
git clone <repository-url> news_pulse
cd news_pulse
```

### 2. Настройка окружения

```bash
# Копирование файла конфигурации
cp backend/env.example .env

# Редактирование переменных окружения
nano .env
```

### 3. Запуск системы

```bash
# Переход в директорию backend
cd backend

# Сборка и запуск всех сервисов
docker-compose up -d --build

# Проверка статуса
docker-compose ps
```

### 4. Проверка работоспособности

```bash
# Проверка API Gateway
curl http://localhost:8080/health

# Проверка frontend
curl http://localhost:3000
```

## ⚙️ Подробная настройка

### Переменные окружения (.env)

Создайте файл `.env` в корне проекта со следующими настройками:

```bash
# Основные настройки
APP_ENV=production
LOG_LEVEL=info

# База данных
POSTGRES_PASSWORD=news_pulse_secure_password_2024

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# AI сервисы
DEEPSEEK_API_KEY=your-deepseek-api-key
GOOGLE_AI_API_KEY=your-google-ai-api-key

# Прокси (опционально)
PROXY_ENABLED=false
PROXY_URL=http://195.96.157.223:9765
PROXY_USERNAME=user322683
PROXY_PASSWORD=97fgob

# Парсинг RSS
RSS_PARSE_INTERVAL_MINUTES=10
RSS_MAX_CONCURRENT_PARSERS=5

# API настройки
API_DEFAULT_PAGE_SIZE=20
API_MAX_PAGE_SIZE=100
API_MAX_PULSES_PER_USER=10
API_MAX_NEWS_PER_FEED=100

# Кэширование
CACHE_ENABLED=true

# Безопасность
AUTH_ENABLED=false
RATE_LIMITING_ENABLED=false
CORS_ENABLED=true

# WebSocket
WEBSOCKET_ENABLED=true
```

### Порты сервисов

| Сервис         | Порт | Описание                        |
| -------------------- | -------- | --------------------------------------- |
| API Gateway          | 8080     | Основной API                    |
| News Parsing         | 8085     | Парсинг новостей         |
| News Management      | 8082     | Управление новостями |
| Pulse Service        | 8083     | Пульсы пользователей |
| Notification Service | 8084     | Уведомления                  |
| PostgreSQL           | 5433     | База данных                   |
| Ollama               | 11434    | AI сервис                         |
| Frontend             | 3000     | Web интерфейс                  |

## 🔧 Управление сервисами

### Основные команды

```bash
# Запуск всех сервисов
docker-compose up -d

# Остановка всех сервисов
docker-compose down

# Перезапуск сервиса
docker-compose restart service_name

# Просмотр логов
docker-compose logs -f service_name

# Обновление сервисов
docker-compose pull
docker-compose up -d --build
```

### Управление отдельными сервисами

```bash
# Запуск только базы данных
docker-compose up -d postgres

# Запуск backend сервисов
docker-compose up -d news-parsing-service news-management-service pulse-service notification-service api-gateway

# Запуск frontend
docker-compose up -d frontend
```

## 📊 Мониторинг и диагностика

### Проверка состояния сервисов

```bash
# Статус всех контейнеров
docker-compose ps

# Использование ресурсов
docker stats

# Логи всех сервисов
docker-compose logs -f

# Логи конкретного сервиса
docker-compose logs -f api-gateway
```

### Health checks

```bash
# API Gateway
curl http://localhost:8080/health

# News Management
curl http://localhost:8082/health

# Pulse Service
curl http://localhost:8083/health

# Notification Service
curl http://localhost:8084/health
```

### Проверка базы данных

```bash
# Подключение к PostgreSQL
docker exec -it news_pulse_postgres psql -U news_pulse_user -d news_pulse

# Проверка таблиц
docker exec news_pulse_postgres psql -U news_pulse_user -d news_pulse -c "\dt"

# Количество новостей
docker exec news_pulse_postgres psql -U news_pulse_user -d news_pulse -c "SELECT COUNT(*) FROM news;"
```

## 🔄 Инициализация и настройка

### Первоначальная настройка Ollama

```bash
# Загрузка модели для классификации
docker exec news_pulse_ollama ollama pull llama3.2:3b

# Проверка доступности модели
curl http://localhost:11434/api/tags
```

### Запуск парсинга новостей

```bash
# Проверка статуса парсинга
curl http://localhost:8080/api/parsing/status

# Ручной запуск парсинга всех источников
curl -X POST http://localhost:8080/api/parsing/parse-all

# Парсинг конкретного источника
curl -X POST http://localhost:8080/api/parsing/parse-source/1
```

## 🛠️ Разработка

### Backend разработка

```bash
# Переход в директорию backend
cd backend

# Запуск в режиме разработки
docker-compose -f docker-compose.dev.yml up -d

# Просмотр логов
docker-compose logs -f
```

### Frontend разработка

```bash
# Переход в директорию frontend
cd frontend

# Установка зависимостей
npm install

# Запуск в режиме разработки
npm run dev

# Сборка для продакшена
npm run build
```

## 📦 Резервное копирование

### Создание резервной копии

```bash
# Создание бэкапа базы данных
docker exec news_pulse_postgres pg_dump -U news_pulse_user news_pulse > backup_$(date +%Y%m%d_%H%M%S).sql

# Создание архива конфигураций
tar -czf configs_$(date +%Y%m%d_%H%M%S).tar.gz docker-compose.yml .env
```

### Восстановление из резервной копии

```bash
# Восстановление базы данных
docker exec -i news_pulse_postgres psql -U news_pulse_user news_pulse < backup_file.sql

# Восстановление конфигураций
tar -xzf configs_file.tar.gz
```

## 🔒 Безопасность

### Рекомендации по безопасности

1. **Измените пароли по умолчанию**:

   ```bash
   # В .env файле
   POSTGRES_PASSWORD=strong_random_password
   JWT_SECRET=strong_random_jwt_secret
   ```
2. **Настройте файрвол**:

   ```bash
   # Открыть только необходимые порты
   sudo ufw allow 80
   sudo ufw allow 443
   sudo ufw allow 8080
   sudo ufw enable
   ```
3. **Используйте HTTPS**:

   - Настройте reverse proxy (Nginx)
   - Получите SSL сертификат (Let's Encrypt)
4. **Регулярные обновления**:

   ```bash
   # Обновление Docker образов
   docker-compose pull
   docker-compose up -d
   ```

## 🚨 Устранение неполадок

### Частые проблемы

1. **Сервис не запускается**:

   ```bash
   # Проверка логов
   docker-compose logs service_name

   # Проверка конфигурации
   docker-compose config
   ```
2. **База данных недоступна**:

   ```bash
   # Проверка статуса PostgreSQL
   docker exec news_pulse_postgres pg_isready

   # Проверка подключения
   docker exec news_pulse_postgres psql -U news_pulse_user -d news_pulse
   ```
3. **Парсинг не работает**:

   ```bash
   # Проверка статуса парсинга
   curl http://localhost:8080/api/parsing/status

   # Ручной запуск парсинга
   curl -X POST http://localhost:8080/api/parsing/parse-all
   ```
4. **Недостаточно памяти**:

   ```bash
   # Мониторинг использования памяти
   docker stats

   # Остановка неиспользуемых сервисов
   docker-compose stop service_name
   ```

### Логи и диагностика

```bash
# Все логи
docker-compose logs -f

# Логи с фильтрацией
docker-compose logs -f | grep ERROR

# Логи за последний час
docker-compose logs --since 1h

# Логи конкретного сервиса
docker-compose logs -f api-gateway --tail=100
```

## 📈 Оптимизация производительности

### Настройки для 8GB сервера

1. **Увеличить количество парсеров**:

   ```bash
   # В .env файле
   RSS_MAX_CONCURRENT_PARSERS=10
   ```
2. **Настроить кэширование**:

   ```bash
   # В .env файле
   CACHE_ENABLED=true
   ```
3. **Оптимизировать PostgreSQL**:

   - Увеличить `shared_buffers`
   - Настроить `work_mem`
   - Добавить индексы

### Мониторинг ресурсов

```bash
# Использование памяти
docker stats --no-stream

# Использование диска
df -h

# Нагрузка на CPU
top
```

## 📚 Дополнительная документация

- [Архитектура системы](documentation/architecture.md)
- [API документация](documentation/api_documentation.md)
- [Структура базы данных](documentation/database_structure.md)
- [Конфигурация развертывания](documentation/deployment_config.md)

## 🤝 Поддержка

При возникновении проблем:

1. Проверьте логи сервисов
2. Убедитесь в корректности конфигурации
3. Проверьте доступность ресурсов
4. Обратитесь к документации

## 📄 Лицензия

Проект распространяется под лицензией MIT.

---

**News Pulse** - современная система персонализированных новостей с AI-классификацией и real-time уведомлениями.