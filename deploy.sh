#!/bin/bash

# News Pulse - Скрипт развертывания
# Автор: News Pulse Team
# Версия: 1.0

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
print_message() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверка системных требований
check_requirements() {
    print_message "Проверка системных требований..."
    
    # Проверка Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker не установлен. Установите Docker и повторите попытку."
        exit 1
    fi
    
    # Проверка Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose не установлен. Установите Docker Compose и повторите попытку."
        exit 1
    fi
    
    # Проверка RAM
    TOTAL_RAM=$(free -m | awk 'NR==2{printf "%.0f", $2/1024}')
    if [ "$TOTAL_RAM" -lt 8 ]; then
        print_warning "Недостаточно RAM: ${TOTAL_RAM}GB. Рекомендуется минимум 8GB"
        read -p "Продолжить развертывание? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        print_success "RAM: ${TOTAL_RAM}GB - достаточно"
    fi
    
    # Проверка свободного места на диске
    FREE_SPACE=$(df -BG . | awk 'NR==2{print $4}' | sed 's/G//')
    if [ "$FREE_SPACE" -lt 20 ]; then
        print_warning "Недостаточно свободного места: ${FREE_SPACE}GB. Рекомендуется минимум 20GB"
        read -p "Продолжить развертывание? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        print_success "Свободное место: ${FREE_SPACE}GB - достаточно"
    fi
}

# Создание .env файла
create_env_file() {
    if [ ! -f .env ]; then
        print_message "Создание .env файла..."
        if [ -f backend/env.example ]; then
            cp backend/env.example .env
            print_success ".env файл создан из шаблона"
            print_warning "Не забудьте настроить переменные в .env файле!"
        else
            print_error "Файл backend/env.example не найден"
            exit 1
        fi
    else
        print_message ".env файл уже существует"
    fi
}

# Остановка старых контейнеров
stop_old_containers() {
    print_message "Остановка старых контейнеров..."
    if [ -f backend/docker-compose.yml ]; then
        cd backend
        docker-compose down 2>/dev/null || true
        cd ..
        print_success "Старые контейнеры остановлены"
    else
        print_error "Файл backend/docker-compose.yml не найден"
        exit 1
    fi
}

# Сборка и запуск сервисов
build_and_start() {
    print_message "Сборка и запуск сервисов..."
    
    cd backend
    
    # Сборка образов
    print_message "Сборка Docker образов..."
    docker-compose build --no-cache
    
    # Запуск PostgreSQL
    print_message "Запуск PostgreSQL..."
    docker-compose up -d postgres
    sleep 10
    
    # Проверка подключения к БД
    print_message "Проверка подключения к базе данных..."
    for i in {1..30}; do
        if docker-compose exec -T postgres pg_isready -U news_pulse_user -d news_pulse >/dev/null 2>&1; then
            print_success "PostgreSQL готов к работе"
            break
        fi
        if [ $i -eq 30 ]; then
            print_error "Не удалось подключиться к PostgreSQL"
            exit 1
        fi
        sleep 2
    done
    
    # Запуск остальных сервисов
    print_message "Запуск основных сервисов..."
    docker-compose up -d news-parsing-service news-management-service pulse-service notification-service api-gateway
    
    # Запуск frontend (если есть)
    if [ -f ../frontend/package.json ]; then
        print_message "Запуск frontend..."
        docker-compose up -d frontend 2>/dev/null || print_warning "Frontend не запущен (возможно, не настроен)"
    fi
    
    cd ..
}

# Проверка состояния сервисов
check_services() {
    print_message "Проверка состояния сервисов..."
    
    cd backend
    
    # Список сервисов для проверки
    SERVICES=("postgres" "api-gateway" "news-parsing-service" "news-management-service" "pulse-service" "notification-service")
    
    sleep 30  # Даем время сервисам запуститься
    
    for service in "${SERVICES[@]}"; do
        if docker-compose ps $service | grep -q "Up"; then
            print_success "$service - работает"
        else
            print_error "$service - ошибка"
            print_message "Логи $service:"
            docker-compose logs --tail=20 $service
        fi
    done
    
    cd ..
}

# Инициализация Ollama
init_ollama() {
    print_message "Инициализация Ollama AI сервиса..."
    
    cd backend
    
    # Проверка доступности Ollama
    if docker-compose ps ollama | grep -q "Up"; then
        print_message "Загрузка модели для классификации новостей..."
        docker exec news_pulse_ollama ollama pull llama3.2:3b || print_warning "Не удалось загрузить модель Ollama"
        
        # Проверка модели
        if curl -f http://localhost:11434/api/tags >/dev/null 2>&1; then
            print_success "Ollama готов к работе"
        else
            print_warning "Ollama не отвечает"
        fi
    else
        print_warning "Ollama не запущен"
    fi
    
    cd ..
}

# Проверка API
check_api() {
    print_message "Проверка API endpoints..."
    
    # Проверка API Gateway
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        print_success "API Gateway доступен"
    else
        print_error "API Gateway недоступен"
    fi
    
    # Проверка парсинга
    if curl -f http://localhost:8080/api/parsing/status >/dev/null 2>&1; then
        print_success "News Parsing Service доступен"
    else
        print_warning "News Parsing Service недоступен"
    fi
}

# Запуск парсинга новостей
start_parsing() {
    print_message "Запуск парсинга новостей..."
    
    if curl -X POST http://localhost:8080/api/parsing/parse-all >/dev/null 2>&1; then
        print_success "Парсинг новостей запущен"
    else
        print_warning "Не удалось запустить парсинг новостей"
    fi
}

# Вывод информации о развертывании
show_deployment_info() {
    print_success "Развертывание завершено!"
    echo
    echo "🌐 Доступные сервисы:"
    echo "  - Frontend: http://localhost:3000"
    echo "  - API Gateway: http://localhost:8080"
    echo "  - API Health: http://localhost:8080/health"
    echo "  - News Parsing: http://localhost:8085"
    echo "  - News Management: http://localhost:8082"
    echo "  - Pulse Service: http://localhost:8083"
    echo "  - Notification Service: http://localhost:8084"
    echo "  - Ollama AI: http://localhost:11434"
    echo
    echo "📊 Управление сервисами:"
    echo "  - Просмотр логов: docker-compose logs -f"
    echo "  - Остановка: docker-compose down"
    echo "  - Перезапуск: docker-compose restart"
    echo
    echo "📚 Документация:"
    echo "  - README.md - основная документация"
    echo "  - documentation/ - подробная документация"
    echo
    print_warning "Не забудьте настроить переменные в .env файле!"
}

# Основная функция
main() {
    echo "🚀 News Pulse - Развертывание системы"
    echo "======================================"
    echo
    
    check_requirements
    create_env_file
    stop_old_containers
    build_and_start
    check_services
    init_ollama
    check_api
    start_parsing
    show_deployment_info
}

# Обработка аргументов командной строки
case "${1:-}" in
    --help|-h)
        echo "News Pulse - Скрипт развертывания"
        echo
        echo "Использование: $0 [опции]"
        echo
        echo "Опции:"
        echo "  --help, -h     Показать эту справку"
        echo "  --no-parsing   Не запускать парсинг новостей"
        echo "  --no-ollama    Не инициализировать Ollama"
        echo
        exit 0
        ;;
    --no-parsing)
        SKIP_PARSING=true
        ;;
    --no-ollama)
        SKIP_OLLAMA=true
        ;;
esac

# Запуск основной функции
main
