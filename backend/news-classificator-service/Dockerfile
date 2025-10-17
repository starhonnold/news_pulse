# FastText News Classificator Service Dockerfile

FROM python:3.11-slim

# Установка рабочей директории
WORKDIR /app

# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Копирование requirements
COPY requirements.txt .

# Установка Python зависимостей
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Копирование файлов приложения
COPY . .

# Создание директории для модели
RUN mkdir -p /app/model_cache

# Expose порт
EXPOSE 8085

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8085/health')" || exit 1

# Запуск приложения
CMD ["python3", "main.py"]

