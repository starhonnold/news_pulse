package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/config"
	"news-parsing-service/internal/models"
	"news-parsing-service/internal/repository"
)

// ParsingService представляет сервис парсинга новостей
type ParsingService struct {
	rssParser            *RSSParser
	newsSourceRepo       *repository.NewsSourceRepository
	newsRepo             *repository.NewsRepository
	parsingLogRepo       *repository.ParsingLogRepository
	config               *config.ParsingConfig
	logger               *logrus.Logger
	cron                 *cron.Cron
	isRunning            bool
	mu                   sync.RWMutex
	semaphore            chan struct{}
	countryDetector      *CountryDetector
	contentExtractor     *ContentExtractor
	simpleNewsClassifier *WeightedNewsClassifier
	fastTextClient       *FastTextClassifierClient // Основной FastText классификатор
}

// NewParsingService создает новый сервис парсинга
func NewParsingService(
	rssParser *RSSParser,
	newsSourceRepo *repository.NewsSourceRepository,
	newsRepo *repository.NewsRepository,
	parsingLogRepo *repository.ParsingLogRepository,
	parsingConfig *config.ParsingConfig,
	fullConfig *config.Config,
	logger *logrus.Logger,
) *ParsingService {
	// Создаем семафор для ограничения количества одновременных парсингов
	semaphore := make(chan struct{}, parsingConfig.MaxConcurrentParsers)

	// Создаем cron для планирования задач
	cronScheduler := cron.New(cron.WithSeconds())

	// Создаем детектор стран
	countryDetector := NewCountryDetector(logger)

	// Создаем извлекатель контента
	contentExtractor, err := NewContentExtractor(logger, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to create content extractor, but continuing")
		contentExtractor = nil
	}

	// Создаем классификатор новостей с щадящими параметрами
	simpleNewsClassifier, err := NewWeightedNewsClassifier(logger, WeightedClassifierConfig{
		TitleWeight:   1.6,
		SummaryWeight: 1.0,
		ContentWeight: 1.0,
		MinConfidence: 0.18, // Снижаем порог уверенности
		MinMargin:     0.03, // Снижаем минимальный отрыв
		UseStemming:   true,
		URLPriorBoost: 0.30, // Приор по URL
		BatchTimeout:  30 * time.Second,

		// Щадящая классификация
		AllowUnknown:        true,
		MinScoreForFallback: 0.30,
		FallbackCategory:    CatSociety,
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to create news classifier")
	}

	// Создаем FastText клиент (NEW - основной классификатор)
	var fastTextClient *FastTextClassifierClient
	if fullConfig != nil && fullConfig.FastText.Enabled {
		serviceURL := fullConfig.FastText.ServiceURL
		timeout := fullConfig.FastText.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		fastTextClient = NewFastTextClassifierClient(serviceURL, timeout, logger, true)

		// Проверяем доступность сервиса
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := fastTextClient.HealthCheck(ctx); err != nil {
			logger.WithError(err).Warn("FastText service unavailable, will use fallback classifier")
			fastTextClient.SetEnabled(false)
		} else {
			logger.WithFields(logrus.Fields{
				"url":     serviceURL,
				"timeout": timeout,
			}).Info("✅ FastText classifier initialized and available")
		}
	} else {
		logger.Info("FastText classifier is disabled in config")
	}

	return &ParsingService{
		rssParser:            rssParser,
		newsSourceRepo:       newsSourceRepo,
		newsRepo:             newsRepo,
		parsingLogRepo:       parsingLogRepo,
		config:               parsingConfig,
		logger:               logger,
		cron:                 cronScheduler,
		semaphore:            semaphore,
		countryDetector:      countryDetector,
		contentExtractor:     contentExtractor,
		simpleNewsClassifier: simpleNewsClassifier,
		fastTextClient:       fastTextClient,
	}
}

// Start запускает сервис парсинга
func (s *ParsingService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("parsing service is already running")
	}

	// Запускаем cron задачу для парсинга
	_, err := s.cron.AddFunc("@every 10m", s.ParseAllSources)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.cron.Start()
	s.isRunning = true

	s.logger.Info("Parsing service started")

	// Запускаем немедленный парсинг
	go s.ParseAllSources()

	return nil
}

// Stop останавливает сервис парсинга
func (s *ParsingService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("parsing service is not running")
	}

	s.cron.Stop()
	s.isRunning = false

	s.logger.Info("Parsing service stopped")
	return nil
}

// IsRunning возвращает статус работы сервиса
func (s *ParsingService) IsRunning() bool {
	return s.isRunning
}

// GetStats возвращает статистику парсинга
func (s *ParsingService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"is_running": s.isRunning,
	}
}

// ValidateSource валидирует источник новостей
func (s *ParsingService) ValidateSource(source models.NewsSource) error {
	// Простая валидация
	if source.RSSURL == "" {
		return fmt.Errorf("RSS URL is required")
	}
	return nil
}

// GetFeedInfo получает информацию о фиде
func (s *ParsingService) GetFeedInfo(ctx context.Context, url string) (map[string]interface{}, error) {
	// Заглушка
	return map[string]interface{}{
		"url":         url,
		"title":       "Feed Title",
		"description": "Feed Description",
	}, nil
}

// ExtractContent извлекает контент из статьи
func (s *ParsingService) ExtractContent(ctx context.Context, url string) (string, error) {
	// Заглушка
	return "Extracted content", nil
}

// ParseAllSources парсит все источники новостей
func (s *ParsingService) ParseAllSources() {
	s.logger.Info("Starting to parse all sources")

	sources, err := s.newsSourceRepo.GetActive(context.Background())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active sources")
		return
	}

	if len(sources) == 0 {
		s.logger.Warn("No active sources found")
		return
	}

	s.logger.WithField("sources_count", len(sources)).Info("Found sources to parse")

	// Парсим источники параллельно с ограничением
	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go func(src models.NewsSource) {
			defer wg.Done()
			s.ParseSource(context.Background(), src)
		}(source)
	}

	wg.Wait()
	s.logger.Info("Finished parsing all sources")
}

// ParseSource парсит один источник новостей
func (s *ParsingService) ParseSource(ctx context.Context, source models.NewsSource) {
	// Получаем семафор
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	startTime := time.Now()
	s.logger.WithField("source", source.Name).Info("Starting to parse source")

	// Парсим RSS ленту
	result := s.rssParser.ParseFeed(ctx, source)
	if !result.Success {
		s.logger.WithFields(logrus.Fields{
			"source":  source.Name,
			"rss_url": source.RSSURL,
			"error":   result.Error,
		}).Error("Failed to parse RSS feed")
		return
	}
	items := result.Items

	if len(items) == 0 {
		s.logger.WithField("source", source.Name).Warn("No items found in RSS feed")
		return
	}

	// Обрабатываем элементы
	newsList, err := s.processItems(ctx, items, source)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"source": source.Name,
			"error":  err,
		}).Error("Failed to process items")
		return
	}

	// Сохраняем новости в базу данных пакетом (с защитой от дубликатов)
	itemsFoundCount := len(items)

	if err := s.newsRepo.CreateBatch(ctx, newsList); err != nil {
		s.logger.WithFields(logrus.Fields{
			"source": source.Name,
			"error":  err,
		}).Error("Failed to create news batch")

		// Логируем ошибку парсинга
		parsingLog := models.ParsingLog{
			SourceID:        source.ID,
			Status:          "error",
			ErrorMessage:    err.Error(),
			ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
		}
		if err := s.parsingLogRepo.Create(ctx, &parsingLog); err != nil {
			s.logger.WithError(err).Warn("Failed to create parsing log")
		}
		return
	}

	// Подсчитываем сколько новостей было создано (не пропущено как дубликаты)
	createdCount := 0
	for _, news := range newsList {
		if news.ID > 0 {
			createdCount++
		}
	}

	// Логируем результат парсинга
	parsingLog := models.ParsingLog{
		SourceID:        source.ID,
		Status:          "success",
		NewsCount:       createdCount,
		ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
	}

	if err := s.parsingLogRepo.Create(ctx, &parsingLog); err != nil {
		s.logger.WithError(err).Warn("Failed to create parsing log")
	}

	s.logger.WithFields(logrus.Fields{
		"source":        source.Name,
		"items_found":   itemsFoundCount,
		"items_created": createdCount,
		"parse_time":    time.Since(startTime),
	}).Info("Successfully parsed source")
}

// processItems обрабатывает элементы из RSS ленты и конвертирует их в новости
func (s *ParsingService) processItems(ctx context.Context, items []models.ParsedFeedItem, source models.NewsSource) ([]models.News, error) {
	var newsList []models.News

	// Обрабатываем каждую новость по отдельности
	for _, item := range items {
		// Проверяем, существует ли уже такая новость
		if s.config.EnableDeduplication {
			exists, err := s.newsRepo.ExistsByURL(ctx, item.Link, source.ID)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to check news existence")
				continue
			}
			if exists {
				continue
			}
		}

		// СНАЧАЛА извлекаем полный контент
		var fullContent string
		var contentSource string

		// Пробуем извлечь контент с веб-страницы
		if s.contentExtractor != nil && s.contentExtractor.IsValidURL(item.Link) {
			s.logger.WithField("url", item.Link).Debug("Attempting to extract content with go-readability")
			if extractedContent, err := s.contentExtractor.ExtractFullContent(ctx, item.Link); err == nil && extractedContent != "" {
				fullContent = extractedContent
				contentSource = "web_extraction"
				s.logger.WithFields(logrus.Fields{
					"url":            item.Link,
					"content_length": len(extractedContent),
					"source":         contentSource,
				}).Info("Successfully extracted full content from web page")
			} else {
				s.logger.WithFields(logrus.Fields{
					"url":   item.Link,
					"error": err,
				}).Warn("Failed to extract content from web page")
			}
		}

		// Если не удалось извлечь контент, используем то, что пришло из RSS
		if fullContent == "" {
			fullContent = item.Description
			contentSource = "rss_description"
		}

		// СТРОГАЯ ВАЛИДАЦИЯ: требуем заголовок + описание + контент для качественной классификации
		// Пропускаем новости, которые не соответствуют критериям:
		// 1. Должен быть заголовок (уже проверено в RSS парсере)
		// 2. Должно быть описание (не пустое)
		// 3. Должен быть полный контент (минимум 500 символов)

		if item.Description == "" {
			s.logger.WithFields(logrus.Fields{
				"title": truncateForLog(item.Title, 50),
				"url":   item.Link,
			}).Debug("Skipping news without description")
			continue
		}

		if len(fullContent) < 500 {
			s.logger.WithFields(logrus.Fields{
				"title":          truncateForLog(item.Title, 50),
				"url":            item.Link,
				"content_length": len(fullContent),
			}).Debug("Skipping news with insufficient content (< 500 chars)")
			continue
		}

		// Проверяем общую длину для дополнительной уверенности
		totalContentLength := len(item.Description) + len(fullContent)
		if totalContentLength < 700 {
			s.logger.WithFields(logrus.Fields{
				"title":                truncateForLog(item.Title, 50),
				"url":                  item.Link,
				"description_length":   len(item.Description),
				"content_length":       len(fullContent),
				"total_content_length": totalContentLength,
			}).Debug("Skipping news with insufficient total content (< 700 chars)")
			continue
		}

		// КЛАССИФИКАЦИЯ с полным контекстом
		var categoryID *int

		// 1) FastText классификация с ПОЛНЫМ контентом
		if s.fastTextClient != nil && s.fastTextClient.IsEnabled() {
			// Объединяем заголовок + описание + полный контент
			// Используем всё доступное для максимальной точности
			textForClassification := item.Title + ". " + item.Description

			// Добавляем полный контент (если он отличается от описания)
			if fullContent != "" && fullContent != item.Description {
				textForClassification += ". " + fullContent
			}

			// Ограничиваем текст для классификации (FastText может работать медленно на очень длинных текстах)
			// Берем первые 5000 символов - этого достаточно для точной классификации
			if len(textForClassification) > 5000 {
				textForClassification = textForClassification[:5000]
			}

			// Снижаем порог confidence до 30% для лучшей классификации
			minConfidence := 0.30

			s.logger.WithFields(logrus.Fields{
				"title":              truncateForLog(item.Title, 50),
				"description_length": len(item.Description),
				"content_length":     len(fullContent),
				"total_length":       len(textForClassification),
				"min_confidence":     minConfidence,
			}).Debug("Classifying with FastText (full content)")

			resp, err := s.fastTextClient.Classify(ctx, item.Title, textForClassification)

			if err == nil && resp.CategoryID > 0 && resp.Confidence >= minConfidence {
				categoryID = &resp.CategoryID
				s.logger.WithFields(logrus.Fields{
					"title":              truncateForLog(item.Title, 50),
					"category_id":        resp.CategoryID,
					"category_name":      resp.CategoryName,
					"confidence":         resp.Confidence,
					"original_category":  resp.OriginalCategory,
					"description_len":    len(item.Description),
					"content_len":        len(fullContent),
					"classification_len": len(textForClassification),
				}).Info("✅ News classified with FastText (full content)")
			} else if err != nil {
				s.logger.WithFields(logrus.Fields{
					"title": truncateForLog(item.Title, 50),
					"error": err,
				}).Warn("FastText classification failed, using fallback")
			} else if resp.Confidence < minConfidence {
				s.logger.WithFields(logrus.Fields{
					"title":          truncateForLog(item.Title, 50),
					"confidence":     resp.Confidence,
					"min_confidence": minConfidence,
					"reason":         "confidence too low",
				}).Debug("FastText confidence too low, using fallback")
			}
		}

		// 2) Fallback на "Общество" если FastText не уверен
		// WeightedClassifier ОТКЛЮЧЕН - он работает хуже FastText
		if categoryID == nil {
			fallbackCategory := CatSociety // 5 — Общество
			categoryID = &fallbackCategory
			s.logger.WithFields(logrus.Fields{
				"title":       truncateForLog(item.Title, 50),
				"fallback_id": fallbackCategory,
				"reason":      "FastText confidence too low or error, using fallback",
			}).Info("📋 Using fallback category (Общество)")
		}

		// Детектируем страну
		var detectedCountry *string
		if fullContent != "" {
			detectedCountry = s.countryDetector.DetectCountry(item.Title, item.Description, fullContent)
		}

		// Создаем объект новости
		news := models.News{
			Title:          item.Title,
			Description:    s.ensureString(item.Description),
			Content:        s.ensureString(fullContent),
			URL:            item.Link,
			ImageURL:       s.ensureString(item.ImageURL),
			Author:         s.ensureString(item.Author),
			SourceID:       source.ID,
			CategoryID:     categoryID,
			PublishedAt:    item.Published,
			RelevanceScore: s.calculateRelevanceScore(item),
			IsActive:       true,
		}

		// Логируем результаты анализа
		s.logger.WithFields(logrus.Fields{
			"title":            truncateForLog(item.Title, 50),
			"category_id":      categoryID,
			"detected_country": detectedCountry,
			"content_length":   len(fullContent),
			"content_source":   contentSource,
			"source_id":        source.ID,
		}).Debug("Processed news item")

		// Валидируем новость
		if !news.IsValid() {
			s.logger.WithFields(logrus.Fields{
				"title":     item.Title,
				"url":       item.Link,
				"source_id": source.ID,
			}).Debug("Skipping invalid news")
			continue
		}

		// Добавляем в список
		newsList = append(newsList, news)
	}

	return newsList, nil
}

// calculateRelevanceScore вычисляет релевантность новости
func (s *ParsingService) calculateRelevanceScore(item models.ParsedFeedItem) float64 {
	score := 0.5 // Базовый балл

	// Увеличиваем балл за наличие изображения
	if item.ImageURL != "" {
		score += 0.1
	}

	// Увеличиваем балл за длину контента
	if len(item.Content) > 100 {
		score += 0.1
	}
	if len(item.Content) > 500 {
		score += 0.1
	}

	// Увеличиваем балл за наличие автора
	if item.Author != "" {
		score += 0.1
	}

	// Ограничиваем балл до 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// extractFullContent извлекает полный контент с веб-страницы
func (s *ParsingService) extractFullContent(ctx context.Context, url string) (string, error) {
	if s.contentExtractor == nil {
		return "", fmt.Errorf("content extractor not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	content, err := s.contentExtractor.ExtractFullContent(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return content, nil
}

// ensureString гарантирует, что строка не пустая (заменяет пустую строку на пустую строку)
func (s *ParsingService) ensureString(str string) string {
	if str == "" {
		return ""
	}
	return str
}

// truncateForLog обрезает строку для логирования
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
