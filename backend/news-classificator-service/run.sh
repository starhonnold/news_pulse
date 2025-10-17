#!/bin/bash
# Скрипт запуска News Classificator Service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 News Classificator Service Launcher"
echo "======================================="

# Проверка Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 не найден. Установите Python 3.8+"
    exit 1
fi

# Проверка виртуального окружения
if [ ! -d "venv" ]; then
    echo "📦 Создание виртуального окружения..."
    python3 -m venv venv
fi

# Активация окружения
echo "🔧 Активация виртуального окружения..."
source venv/bin/activate

# Проверка зависимостей
if [ ! -f "venv/bin/uvicorn" ]; then
    echo "📥 Установка зависимостей..."
    pip install --upgrade pip
    pip install -r requirements.txt
fi

# Запуск сервиса
echo "✅ Запуск сервиса на http://localhost:8085"
echo "📚 Документация API: http://localhost:8085/docs"
echo ""
echo "Для остановки нажмите Ctrl+C"
echo "======================================="

python3 main.py

