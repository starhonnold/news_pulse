"""
FastAPI сервис классификации новостей
"""

import logging
import time
from typing import List, Dict, Any
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import yaml

from classifier import FastTextNewsClassifier


# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Глобальный классификатор
classifier = None


# Pydantic модели
class ClassifyRequest(BaseModel):
    """Запрос на классификацию одного текста"""
    text: str = Field(..., description="Текст новости для классификации")


class BatchItem(BaseModel):
    """Элемент для пакетной классификации"""
    index: int = Field(..., description="Индекс элемента")
    title: str = Field(default="", description="Заголовок новости")
    description: str = Field(default="", description="Описание новости")


class BatchClassifyRequest(BaseModel):
    """Запрос на пакетную классификацию"""
    items: List[BatchItem] = Field(..., description="Список новостей для классификации")


class ClassifyResponse(BaseModel):
    """Ответ классификации"""
    original_category: str = Field(..., description="Оригинальная категория FastText")
    original_score: float = Field(..., description="Оригинальный скор FastText")
    category_id: int = Field(..., description="ID категории проекта")
    category_name: str = Field(..., description="Название категории проекта")
    confidence: float = Field(..., description="Уверенность классификации (0-1)")


class BatchClassifyResponse(BaseModel):
    """Ответ пакетной классификации"""
    results: List[Dict[str, Any]] = Field(..., description="Результаты классификации")


class HealthResponse(BaseModel):
    """Ответ health check"""
    status: str
    model_loaded: bool
    model_info: Dict[str, Any]
    uptime: float


# Lifespan для инициализации
@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan события приложения"""
    global classifier
    
    # Startup
    logger.info("🚀 Starting News Classificator Service...")
    
    try:
        classifier = FastTextNewsClassifier()
        classifier.load_model()
        logger.info("✅ Classifier initialized successfully")
    except Exception as e:
        logger.error(f"❌ Failed to initialize classifier: {e}")
        raise
    
    yield
    
    # Shutdown
    logger.info("👋 Shutting down News Classificator Service...")


# Создание приложения
app = FastAPI(
    title="News Classificator Service",
    description="FastText-based русскоязычный классификатор новостей",
    version="1.0.0",
    lifespan=lifespan
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Время запуска для uptime
START_TIME = time.time()


@app.get("/", tags=["Root"])
async def root():
    """Корневой endpoint"""
    return {
        "service": "News Classificator Service",
        "version": "1.0.0",
        "model": "data-silence/fasttext-rus-news-classifier",
        "status": "running"
    }


@app.get("/health", response_model=HealthResponse, tags=["Health"])
async def health_check():
    """Health check endpoint"""
    if classifier is None:
        raise HTTPException(status_code=503, detail="Classifier not initialized")
    
    uptime = time.time() - START_TIME
    
    return HealthResponse(
        status="healthy" if classifier.is_loaded() else "unhealthy",
        model_loaded=classifier.is_loaded(),
        model_info=classifier.get_model_info(),
        uptime=uptime
    )


@app.post("/classify", response_model=ClassifyResponse, tags=["Classification"])
async def classify_text(request: ClassifyRequest):
    """
    Классификация одного текста
    
    Args:
        request: Запрос с текстом для классификации
        
    Returns:
        Результат классификации
    """
    if classifier is None or not classifier.is_loaded():
        raise HTTPException(status_code=503, detail="Classifier not loaded")
    
    try:
        result = classifier.classify_text(request.text)
        return ClassifyResponse(**result)
    except Exception as e:
        logger.error(f"Classification error: {e}")
        raise HTTPException(status_code=500, detail=f"Classification failed: {str(e)}")


@app.post("/classify/batch", response_model=BatchClassifyResponse, tags=["Classification"])
async def classify_batch(request: BatchClassifyRequest):
    """
    Пакетная классификация новостей
    
    Args:
        request: Запрос со списком новостей
        
    Returns:
        Список результатов классификации
    """
    if classifier is None or not classifier.is_loaded():
        raise HTTPException(status_code=503, detail="Classifier not loaded")
    
    try:
        # Конвертируем Pydantic модели в словари
        items = [item.model_dump() for item in request.items]
        
        results = classifier.classify_batch(items)
        
        return BatchClassifyResponse(results=results)
    except Exception as e:
        logger.error(f"Batch classification error: {e}")
        raise HTTPException(status_code=500, detail=f"Batch classification failed: {str(e)}")


@app.get("/categories", tags=["Info"])
async def get_categories():
    """Получение списка категорий"""
    if classifier is None:
        raise HTTPException(status_code=503, detail="Classifier not initialized")
    
    return {
        "fasttext_categories": list(classifier.category_mapping.keys()),
        "project_categories": classifier.category_names,
        "mapping": classifier.category_mapping
    }


if __name__ == "__main__":
    import uvicorn
    
    # Загружаем конфигурацию
    with open("config.yaml", "r", encoding="utf-8") as f:
        config = yaml.safe_load(f)
    
    host = config['server']['host']
    port = config['server']['port']
    workers = config['server']['workers']
    
    logger.info(f"Starting server on {host}:{port}")
    
    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        workers=workers,
        log_level="info"
    )

