#!/bin/bash
# Скрипт тестирования классификатора

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🧪 News Classificator Test Runner"
echo "=================================="

# Проверка виртуального окружения
if [ ! -d "venv" ]; then
    echo "❌ Виртуальное окружение не найдено"
    echo "Запустите: ./run.sh сначала"
    exit 1
fi

# Активация окружения
source venv/bin/activate

# Запуск тестов
echo "▶️  Запуск тестов..."
echo ""

python3 test_classifier.py

echo ""
echo "=================================="
echo "✅ Тестирование завершено"

