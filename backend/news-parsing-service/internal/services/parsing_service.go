package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
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
	simpleNewsClassifier *SimpleNewsClassifier
	aiClassifier         *AIClassifier
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
		logger.WithError(err).Error("Failed to create content extractor")
		return nil
	}

	// Создаем классификатор новостей
	simpleNewsClassifier := NewSimpleNewsClassifier(logger)

	// Создаем AI классификатор
	ollamaURL := "http://ollama:11434"
	model := "hf.co/Vikhrmodels/Vikhr-Llama-3.2-1B-instruct-GGUF:Q4_K_M"
	timeout := 60 * time.Second

	if fullConfig != nil {
		if fullConfig.AI.OllamaURL != "" {
			ollamaURL = fullConfig.AI.OllamaURL
		}
		if fullConfig.AI.Model != "" {
			model = fullConfig.AI.Model
		}
		if fullConfig.AI.Timeout > 0 {
			timeout = fullConfig.AI.Timeout
		}
	}
	aiClassifier := NewAIClassifier(ollamaURL, model, timeout, fullConfig.AI.Temperature, logger)

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
		aiClassifier:         aiClassifier,
	}
}

// Start запускает сервис парсинга
func (s *ParsingService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("parsing service is already running")
	}

	s.logger.Info("Starting parsing service")

	// Добавляем задачу в cron
	cronExpr := fmt.Sprintf("@every %s", s.config.Interval.String())
	_, err := s.cron.AddFunc(cronExpr, func() {
		if err := s.ParseAllSources(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to parse all sources")
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Запускаем cron
	s.cron.Start()
	s.isRunning = true

	s.logger.WithField("interval", s.config.Interval).Info("Parsing service started")

	// Запускаем первый парсинг
	go func() {
		if err := s.ParseAllSources(ctx); err != nil {
			s.logger.WithError(err).Error("Failed initial parsing")
		}
	}()

	return nil
}

// Stop останавливает сервис парсинга
func (s *ParsingService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.logger.Info("Stopping parsing service")

	// Останавливаем cron
	s.cron.Stop()
	s.isRunning = false

	s.logger.Info("Parsing service stopped")
	return nil
}

// ParseAllSources парсит все активные источники
func (s *ParsingService) ParseAllSources(ctx context.Context) error {
	s.logger.Info("Starting to parse all sources")

	// Получаем источники, которые нужно парсить
	sources, err := s.newsSourceRepo.GetSourcesToParse(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sources to parse: %w", err)
	}

	if len(sources) == 0 {
		s.logger.Debug("No sources to parse")
		return nil
	}

	s.logger.WithField("sources_count", len(sources)).Info("Found sources to parse")

	// Парсим источники параллельно
	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go func(src models.NewsSource) {
			defer wg.Done()
			s.parseSource(ctx, src)
		}(source)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	s.logger.Info("Finished parsing all sources")
	return nil
}

// parseSource парсит один источник
func (s *ParsingService) parseSource(ctx context.Context, source models.NewsSource) {
	// Ограничиваем количество одновременных парсингов
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	logger := s.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"source_name": source.Name,
		"rss_url":     source.RSSURL,
	})

	logger.Debug("Starting to parse source")

	// Создаем контекст с таймаутом
	parseCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout*2)
	defer cancel()

	// Парсим RSS ленту
	result := s.rssParser.ParseFeed(parseCtx, source)

	// Логируем результат парсинга
	if err := s.parsingLogRepo.LogParsingResult(ctx, result); err != nil {
		logger.WithError(err).Error("Failed to log parsing result")
	}

	if !result.Success {
		logger.WithError(fmt.Errorf("RSS parsing failed: %s", result.Error)).Error("Failed to parse RSS feed")
		return
	}

	if len(result.Items) == 0 {
		logger.Debug("No new items found in RSS feed")
		// Обновляем время последнего парсинга даже если новостей нет
		if err := s.newsSourceRepo.UpdateLastParsedAt(ctx, source.ID, result.ParsedAt); err != nil {
			logger.WithError(err).Error("Failed to update last parsed time")
		}
		return
	}

	// Обрабатываем и сохраняем новости
	newsList, err := s.processItems(ctx, result.Items, source)
	if err != nil {
		logger.WithError(err).Error("Failed to process items")
		return
	}

	if len(newsList) == 0 {
		logger.Debug("No new news to save after processing")
		// Обновляем время последнего парсинга
		if err := s.newsSourceRepo.UpdateLastParsedAt(ctx, source.ID, result.ParsedAt); err != nil {
			logger.WithError(err).Error("Failed to update last parsed time")
		}
		return
	}

	// Сохраняем новости в базу данных
	if err := s.newsRepo.CreateBatch(ctx, newsList); err != nil {
		logger.WithError(err).Error("Failed to save news batch")
		return
	}

	// Обновляем время последнего парсинга
	if err := s.newsSourceRepo.UpdateLastParsedAt(ctx, source.ID, result.ParsedAt); err != nil {
		logger.WithError(err).Error("Failed to update last parsed time")
	}

	logger.WithFields(logrus.Fields{
		"items_parsed": len(result.Items),
		"news_saved":   len(newsList),
		"parse_time":   result.ExecutionTime,
	}).Info("Successfully parsed source")
}

// processItems обрабатывает элементы из RSS ленты и конвертирует их в новости
func (s *ParsingService) processItems(ctx context.Context, items []models.ParsedFeedItem, source models.NewsSource) ([]models.News, error) {
	var newsList []models.News
	var itemsForClassification []UnifiedNewsItem
	var validItems []models.ParsedFeedItem

	// Сначала фильтруем элементы и собираем те, которые нуждаются в обработке
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

		// Добавляем элемент в список валидных
		validItems = append(validItems, item)

		// Собираем элементы для классификации
		itemsForClassification = append(itemsForClassification, UnifiedNewsItem{
			Index:       len(validItems) - 1,
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			URL:         item.Link,
			Categories:  item.Categories,
		})
	}

	// Выполняем классификацию новостей с повторными попытками
	classificationResults := make(map[int]UnifiedProcessingResult)
	if len(itemsForClassification) > 0 {
		s.logger.WithField("items_count", len(itemsForClassification)).Info("Performing news classification")

		// Сначала пробуем AI классификатор
		var results []UnifiedProcessingResult
		var err error

		if s.aiClassifier != nil {
			s.logger.Info("Using AI classifier (Ollama)")
			results, err = s.aiClassifier.ProcessNewsBatch(ctx, itemsForClassification)
			if err != nil {
				s.logger.WithError(err).Warn("AI classification failed, trying simple classifier")
				// Если AI не работает, пробуем простой классификатор
				if s.simpleNewsClassifier != nil {
					results, err = s.simpleNewsClassifier.ProcessNewsBatch(ctx, itemsForClassification)
				}
			} else {
				// Проверяем результаты AI классификации
				var retryItems []UnifiedNewsItem
				for i, result := range results {
					if result.Error != nil || result.CategoryID == 0 {
						s.logger.WithFields(logrus.Fields{
							"index": result.Index,
							"title": truncateForLog(result.Title, 50),
							"error": result.Error,
						}).Warn("AI classification failed for item, will retry with simple classifier")
						retryItems = append(retryItems, itemsForClassification[i])
					} else {
						classificationResults[result.Index] = result
						s.logger.WithFields(logrus.Fields{
							"index":       result.Index,
							"title":       truncateForLog(result.Title, 50),
							"category_id": result.CategoryID,
							"confidence":  result.Confidence,
						}).Info("News classified with AI")
					}
				}

				// Если есть элементы для повторной попытки, пробуем простой классификатор
				if len(retryItems) > 0 && s.simpleNewsClassifier != nil {
					s.logger.WithField("retry_count", len(retryItems)).Info("Retrying failed items with simple classifier")
					retryResults, retryErr := s.simpleNewsClassifier.ProcessNewsBatch(ctx, retryItems)
					if retryErr != nil {
						s.logger.WithError(retryErr).Warn("Simple classifier retry also failed")
					} else {
						// Добавляем успешные результаты повторной попытки
						for _, result := range retryResults {
							if result.Error == nil && result.CategoryID > 0 {
								classificationResults[result.Index] = result
								s.logger.WithFields(logrus.Fields{
									"index":       result.Index,
									"title":       truncateForLog(result.Title, 50),
									"category_id": result.CategoryID,
									"confidence":  result.Confidence,
								}).Info("News classified with simple classifier retry")
							} else {
								s.logger.WithFields(logrus.Fields{
									"index": result.Index,
									"title": truncateForLog(result.Title, 50),
									"error": result.Error,
								}).Warn("Simple classifier retry also failed for item")
							}
						}
					}
				}
			}
		} else if s.simpleNewsClassifier != nil {
			s.logger.Info("Using simple classifier")
			results, err = s.simpleNewsClassifier.ProcessNewsBatch(ctx, itemsForClassification)
			if err != nil {
				s.logger.WithError(err).Warn("Simple classification failed")
			} else {
				// Проверяем результаты простой классификации
				for _, result := range results {
					if result.Error == nil && result.CategoryID > 0 {
						classificationResults[result.Index] = result
						s.logger.WithFields(logrus.Fields{
							"index":       result.Index,
							"title":       truncateForLog(result.Title, 50),
							"category_id": result.CategoryID,
							"confidence":  result.Confidence,
						}).Info("News classified with simple classifier")
					} else {
						s.logger.WithFields(logrus.Fields{
							"index": result.Index,
							"title": truncateForLog(result.Title, 50),
							"error": result.Error,
						}).Warn("Simple classification failed for item")
					}
				}
			}
		}
	}

	// Теперь создаем новости только с успешно определенными категориями
	for i, item := range validItems {
		// Проверяем, есть ли результат классификации для этого элемента
		result, exists := classificationResults[i]
		if !exists || result.CategoryID == 0 {
			s.logger.WithFields(logrus.Fields{
				"index": i,
				"title": truncateForLog(item.Title, 50),
				"url":   item.Link,
			}).Warn("Skipping news item - no valid category determined")
			continue
		}

		categoryID := &result.CategoryID

		// Извлекаем полный контент
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

		// Если извлечение с веб-страницы не удалось, используем описание из RSS
		if fullContent == "" {
			fullContent = item.Description
			contentSource = "rss_description"
		}

		// Если и описания нет, используем контент из RSS
		if fullContent == "" {
			if item.Content != "" {
				fullContent = item.Content
			} else {
				fullContent = item.Description
			}
			contentSource = "rss_content"
		}

		// Определяем страну по контенту
		var detectedCountry *string
		if fullContent != "" {
			detectedCountry = s.countryDetector.DetectCountry(item.Title, item.Description, fullContent)
		}

		// Создаем объект новости
		news := models.News{
			Title:          item.Title,
			Description:    item.Description,
			Content:        fullContent,
			URL:            item.Link,
			ImageURL:       item.ImageURL,
			Author:         item.Author,
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

		newsList = append(newsList, news)
	}

	return newsList, nil
}

// calculateRelevanceScore вычисляет оценку релевантности новости
func (s *ParsingService) calculateRelevanceScore(item models.ParsedFeedItem) float64 {
	score := 0.5 // Базовая оценка

	// Учитываем актуальность (свежесть)
	age := time.Since(item.Published)
	if age < time.Hour {
		score += 0.3 // Очень свежие новости
	} else if age < 6*time.Hour {
		score += 0.2 // Свежие новости
	} else if age < 24*time.Hour {
		score += 0.1 // Новости за день
	}

	// Учитываем длину заголовка (оптимальная длина 50-100 символов)
	titleLen := len(item.Title)
	if titleLen >= 50 && titleLen <= 100 {
		score += 0.1
	}

	// Учитываем наличие описания
	if len(item.Description) > 100 {
		score += 0.05
	}

	// Учитываем наличие изображения
	if item.ImageURL != "" {
		score += 0.05
	}

	// Учитываем наличие автора
	if item.Author != "" {
		score += 0.05
	}

	// Ограничиваем оценку диапазоном [0.0, 1.0]
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// ParseSource парсит конкретный источник (для API)
func (s *ParsingService) ParseSource(ctx context.Context, sourceID int) error {
	source, err := s.newsSourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source: %w", err)
	}

	if !source.IsActive {
		return fmt.Errorf("source %d is not active", sourceID)
	}

	s.parseSource(ctx, *source)
	return nil
}

// GetStats возвращает статистику парсинга
func (s *ParsingService) GetStats(ctx context.Context) (*models.ParsingStats, error) {
	// Получаем статистику за последние 24 часа
	since := time.Now().Add(-24 * time.Hour)

	// Статистика из логов парсинга
	parsingStats, err := s.parsingLogRepo.GetStats(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get parsing stats: %w", err)
	}

	// Статистика источников
	sourceStats, err := s.newsSourceRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get source stats: %w", err)
	}

	// Статистика новостей
	newsStats, err := s.newsRepo.GetNewsStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get news stats: %w", err)
	}

	// Объединяем статистики
	stats := &models.ParsingStats{
		TotalSources:   sourceStats.TotalSources,
		ActiveSources:  sourceStats.ActiveSources,
		SuccessfulRuns: parsingStats.SuccessfulRuns,
		FailedRuns:     parsingStats.FailedRuns,
		TotalNews:      newsStats.TotalNews,
		NewsToday:      newsStats.NewsToday,
		AvgParseTime:   parsingStats.AvgParseTime,
		LastParseTime:  parsingStats.LastParseTime,
	}

	return stats, nil
}

// IsRunning возвращает статус работы сервиса
func (s *ParsingService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// ValidateSource проверяет корректность источника RSS
func (s *ParsingService) ValidateSource(ctx context.Context, rssURL string) error {
	return s.rssParser.ValidateFeed(ctx, rssURL)
}

// GetFeedInfo возвращает информацию о RSS ленте
func (s *ParsingService) GetFeedInfo(ctx context.Context, rssURL string) (*models.NewsSource, error) {
	feed, err := s.rssParser.GetFeedInfo(ctx, rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed info: %w", err)
	}

	// Создаем объект источника на основе информации из RSS
	source := &models.NewsSource{
		Name:                 feed.Title,
		Domain:               extractDomainFromURL(rssURL),
		RSSURL:               rssURL,
		WebsiteURL:           feed.Link,
		Language:             "ru", // По умолчанию русский
		Description:          feed.Description,
		LogoURL:              extractImageFromFeed(feed),
		IsActive:             true,
		ParseIntervalMinutes: 10, // По умолчанию 10 минут
	}

	return source, nil
}

// extractDomainFromURL извлекает домен из URL
func extractDomainFromURL(url string) string {
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
}

// extractImageFromFeed извлекает URL изображения из RSS ленты
func extractImageFromFeed(feed *gofeed.Feed) string {
	if feed.Image != nil && feed.Image.URL != "" {
		return feed.Image.URL
	}

	// Можно добавить логику для извлечения favicon
	return ""
}

// ExtractContent извлекает контент с веб-страницы
func (s *ParsingService) ExtractContent(url string) (string, error) {
	if s.contentExtractor == nil {
		return "", fmt.Errorf("content extractor not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	content, err := s.contentExtractor.ExtractFullContent(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return content, nil
}
