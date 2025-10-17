"""
FastText классификатор новостей
Модель: data-silence/fasttext-rus-news-classifier
"""

import os
import logging
from typing import Dict, List, Any, Optional
from huggingface_hub import hf_hub_download
import fasttext
import yaml


logger = logging.getLogger(__name__)


class FastTextNewsClassifier:
    """FastText классификатор русскоязычных новостей"""
    
    def __init__(self, config_path: str = "config.yaml"):
        """
        Инициализация классификатора
        
        Args:
            config_path: Путь к файлу конфигурации
        """
        self.model = None
        self.model_path = None
        self.config = self._load_config(config_path)
        
        # Маппинг категорий
        self.category_mapping = self.config['category_mapping']
        self.category_names = {
            int(k): v for k, v in self.config['category_names'].items()
        }
        
    def _load_config(self, config_path: str) -> Dict:
        """Загрузка конфигурации из YAML"""
        try:
            with open(config_path, 'r', encoding='utf-8') as f:
                return yaml.safe_load(f)
        except Exception as e:
            logger.error(f"Failed to load config: {e}")
            raise
    
    def load_model(self) -> None:
        """Загрузка модели с Hugging Face Hub"""
        logger.info("Loading FastText model from Hugging Face Hub...")
        
        try:
            repo_id = self.config['model']['repo_id']
            filename = self.config['model']['filename']
            cache_dir = self.config['model'].get('cache_dir', './model_cache')
            
            # Создаем директорию для кэша если не существует
            os.makedirs(cache_dir, exist_ok=True)
            
            # Загружаем модель
            self.model_path = hf_hub_download(
                repo_id=repo_id,
                filename=filename,
                cache_dir=cache_dir
            )
            
            logger.info(f"Model downloaded to: {self.model_path}")
            
            # Загружаем модель FastText
            # Подавляем warning от FastText
            fasttext.FastText.eprint = lambda x: None
            self.model = fasttext.load_model(self.model_path)
            
            logger.info("✅ Model loaded successfully")
            
        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            raise
    
    def classify_text(self, text: str) -> Dict[str, Any]:
        """
        Классификация одного текста
        
        Args:
            text: Текст новости (заголовок + описание)
            
        Returns:
            Словарь с результатом классификации:
            {
                'original_category': str,      # Категория FastText
                'original_score': float,       # Уверенность FastText
                'category_id': int,            # ID категории проекта
                'category_name': str,          # Название категории проекта
                'confidence': float            # Уверенность (0-1)
            }
        """
        if not self.model:
            raise RuntimeError("Model not loaded. Call load_model() first.")
        
        # Очистка текста
        text = text.strip().replace('\n', ' ').replace('\r', '')
        
        if not text:
            logger.warning("Empty text provided for classification")
            return {
                'original_category': 'unknown',
                'original_score': 0.0,
                'category_id': 5,  # Общество по умолчанию
                'category_name': 'Общество',
                'confidence': 0.0,
            }
        
        # Классификация
        try:
            prediction = self.model.predict(text, k=1)
            
            # Извлечение результата
            original_label = prediction[0][0].replace("__label__", "")
            original_score = float(prediction[1][0])
            
            # Маппинг на категории проекта
            mapped_id = self.category_mapping.get(original_label, 5)
            mapped_name = self.category_names[mapped_id]
            
            return {
                'original_category': original_label,
                'original_score': original_score,
                'category_id': mapped_id,
                'category_name': mapped_name,
                'confidence': original_score,
            }
            
        except Exception as e:
            logger.error(f"Classification error: {e}")
            # Возвращаем Общество в случае ошибки
            return {
                'original_category': 'error',
                'original_score': 0.0,
                'category_id': 5,
                'category_name': 'Общество',
                'confidence': 0.0,
            }
    
    def classify_batch(
        self,
        items: List[Dict[str, Any]]
    ) -> List[Dict[str, Any]]:
        """
        Пакетная классификация текстов
        
        Args:
            items: Список элементов для классификации
                   Каждый элемент: {'index': int, 'title': str, 'description': str}
            
        Returns:
            Список результатов классификации
        """
        results = []
        
        for item in items:
            index = item.get('index', 0)
            title = item.get('title', '')
            description = item.get('description', '')
            
            # Объединяем заголовок и описание
            text = f"{title}. {description}".strip()
            
            # Классифицируем
            result = self.classify_text(text)
            result['index'] = index
            
            results.append(result)
        
        return results
    
    def is_loaded(self) -> bool:
        """Проверка загрузки модели"""
        return self.model is not None
    
    def get_model_info(self) -> Dict[str, Any]:
        """Получение информации о модели"""
        return {
            'repo_id': self.config['model']['repo_id'],
            'filename': self.config['model']['filename'],
            'model_path': self.model_path,
            'is_loaded': self.is_loaded(),
            'categories': len(self.category_mapping),
            'target_categories': len(self.category_names),
        }

