# News Classificator Service

FastText-based русскоязычный классификатор новостей для проекта "Пульс Новостей".

## Описание

Микросервис на Python для классификации новостей с использованием модели [data-silence/fasttext-rus-news-classifier](https://huggingface.co/data-silence/fasttext-rus-news-classifier) от Hugging Face.

## Особенности

- ✅ **CPU-only** - работает без GPU
- 🚀 **Быстрая классификация** - FastText один из самых быстрых алгоритмов
- 📊 **Точность 86.91%** на тестовом датасете
- 🔄 **REST API** - интеграция через HTTP
- 📝 **11 → 5 категорий** - маппинг на категории проекта

## Категории

### FastText модель (11 категорий)
1. sports (спорт)
2. science (наука)
3. politics (политика)
4. economy (экономика)
5. society (общество)
6. culture (культура)
7. health (здоровье)
8. travel (путешествия)
9. conflicts (конфликты)
10. climate (климат)
11. gloss (глянец)

### Маппинг на News Pulse (5 категорий)
| FastText | News Pulse |
|----------|-----------|
| sports | 1 - Спорт |
| science | 2 - Технологии |
| politics | 3 - Политика |
| economy | 4 - Экономика и финансы |
| society | 5 - Общество |
| culture | 5 - Общество |
| health | 5 - Общество |
| travel | 5 - Общество |
| conflicts | 3 - Политика |
| climate | 5 - Общество |
| gloss | 5 - Общество |

## Установка

```bash
cd /root/project/news_pulse/backend/news-classificator-service

# Создание виртуального окружения
python3 -m venv venv
source venv/bin/activate

# Установка зависимостей
pip install --upgrade pip
pip install -r requirements.txt
```

## Запуск

### Разработка
```bash
source venv/bin/activate
python3 main.py
```

### Production (Docker)
```bash
docker build -t news-classificator-service .
docker run -p 8085:8085 news-classificator-service
```

## API Endpoints

### Health Check
```bash
GET /health
```

### Классификация одной новости
```bash
POST /classify
Content-Type: application/json

{
  "text": "Футбольный матч завершился со счетом 2:1"
}
```

**Ответ:**
```json
{
  "original_category": "sports",
  "original_score": 0.9999,
  "category_id": 1,
  "category_name": "Спорт",
  "confidence": 0.9999
}
```

### Пакетная классификация
```bash
POST /classify/batch
Content-Type: application/json

{
  "items": [
    {
      "index": 0,
      "title": "Футбольный матч",
      "description": "Завершился со счетом 2:1"
    },
    {
      "index": 1,
      "title": "iPhone 15 Pro",
      "description": "Apple представила новый смартфон"
    }
  ]
}
```

**Ответ:**
```json
{
  "results": [
    {
      "index": 0,
      "original_category": "sports",
      "category_id": 1,
      "category_name": "Спорт",
      "confidence": 0.9999
    },
    {
      "index": 1,
      "original_category": "science",
      "category_id": 2,
      "category_name": "Технологии",
      "confidence": 0.9856
    }
  ]
}
```

## Тестирование

```bash
# Запуск тестов классификатора
python3 test_classifier.py

# Тест производительности
python3 benchmark.py
```

## Производительность

- **CPU**: 100-200 классификаций/сек
- **Память**: ~100-150MB
- **Латентность**: 5-10ms на классификацию

## Интеграция с News Pulse

Сервис интегрируется с `news-parsing-service` через HTTP API:

```go
// Go код интеграции
type FastTextClient struct {
    baseURL string
    client  *http.Client
}

func (c *FastTextClient) Classify(title, description string) (int, float64, error) {
    // HTTP запрос к сервису
}
```

## Конфигурация

```yaml
server:
  host: "0.0.0.0"
  port: 8085

model:
  repo_id: "data-silence/fasttext-rus-news-classifier"
  filename: "fasttext_news_classifier.bin"
  
logging:
  level: "INFO"
  format: "json"
```

## Docker

```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8085
CMD ["python3", "main.py"]
```

## Лицензия

MIT License

