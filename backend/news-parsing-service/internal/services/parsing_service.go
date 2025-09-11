package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/config"
	"news-parsing-service/internal/models"
	"news-parsing-service/internal/repository"
)

// ParsingService представляет сервис парсинга новостей
type ParsingService struct {
	rssParser                *RSSParser
	newsSourceRepo           *repository.NewsSourceRepository
	newsRepo                 *repository.NewsRepository
	parsingLogRepo           *repository.ParsingLogRepository
	config                   *config.ParsingConfig
	logger                   *logrus.Logger
	cron                     *cron.Cron
	isRunning                bool
	mu                       sync.RWMutex
	semaphore                chan struct{}
	countryDetector          *CountryDetector
	contentExtractor         *ContentExtractor
	deepSeekContentExtractor *DeepSeekContentExtractor
	deepSeekNewsClassifier   *DeepSeekNewsClassifier
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

	// Создаем извлекатель контента (упрощенный)
	contentExtractor := NewContentExtractor(logger, nil)

	// Создаем DeepSeek-извлекатель контента
	deepSeekContentExtractor := NewDeepSeekContentExtractor(fullConfig.DeepSeekAPIKey, logger)

	// Создаем DeepSeek-классификатор новостей
	deepSeekNewsClassifier := NewDeepSeekNewsClassifier(fullConfig.DeepSeekAPIKey, logger)

	return &ParsingService{
		rssParser:                rssParser,
		newsSourceRepo:           newsSourceRepo,
		newsRepo:                 newsRepo,
		parsingLogRepo:           parsingLogRepo,
		config:                   parsingConfig,
		logger:                   logger,
		cron:                     cronScheduler,
		semaphore:                semaphore,
		countryDetector:          countryDetector,
		contentExtractor:         contentExtractor,
		deepSeekContentExtractor: deepSeekContentExtractor,
		deepSeekNewsClassifier:   deepSeekNewsClassifier,
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
		logger.WithError(fmt.Errorf(result.Error)).Error("Failed to parse RSS feed")
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
	var itemsForDeepSeekClassification []DeepSeekNewsItem
	var itemsForContentExtraction []DeepSeekContentExtractionItem

	// Сначала обрабатываем все элементы и собираем те, которые нуждаются в обработке
	for i, item := range items {
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

		// Собираем элементы для DeepSeek-классификации
		itemsForDeepSeekClassification = append(itemsForDeepSeekClassification, DeepSeekNewsItem{
			Index:       i,
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Categories:  item.Categories,
		})

		// Собираем элементы для извлечения полного контента через DeepSeek
		if s.deepSeekContentExtractor.IsValidURL(item.Link) {
			itemsForContentExtraction = append(itemsForContentExtraction, DeepSeekContentExtractionItem{
				URL:   item.Link,
				Index: i,
			})
		}
	}

	// Выполняем batch-классификацию для элементов через DeepSeek
	deepSeekResults := make(map[int]int)
	if len(itemsForDeepSeekClassification) > 0 {
		s.logger.WithField("items_count", len(itemsForDeepSeekClassification)).Info("Performing batch DeepSeek classification")

		// Ограничиваем количество элементов для batch-классификации (максимум 10)
		batchSize := 10
		if len(itemsForDeepSeekClassification) > batchSize {
			itemsForDeepSeekClassification = itemsForDeepSeekClassification[:batchSize]
		}

		results, err := s.deepSeekNewsClassifier.ClassifyNewsBatch(ctx, itemsForDeepSeekClassification)
		if err != nil {
			s.logger.WithError(err).Warn("Batch DeepSeek classification failed")
		} else {
			// Создаем карту результатов DeepSeek-классификации
			for _, result := range results {
				if result.Error == nil {
					deepSeekResults[result.Index] = result.CategoryID
					s.logger.WithFields(logrus.Fields{
						"index":       result.Index,
						"category_id": result.CategoryID,
						"confidence":  result.Confidence,
					}).Info("DeepSeek classified news category")
				}
			}
		}
	}

	// Выполняем batch-извлечение контента через AI
	contentResults := make(map[int]DeepSeekContentExtractionResult)
	if len(itemsForContentExtraction) > 0 {
		s.logger.WithField("items_count", len(itemsForContentExtraction)).Info("Performing batch AI content extraction")

		results, err := s.deepSeekContentExtractor.ExtractContentBatch(ctx, itemsForContentExtraction)
		if err != nil {
			s.logger.WithError(err).Warn("Batch AI content extraction failed")
		} else {
			// Создаем карту результатов извлечения контента
			for _, result := range results {
				if result.Error == nil {
					contentResults[result.Index] = result
					s.logger.WithFields(logrus.Fields{
						"index":          result.Index,
						"title":          truncateForLog(result.Title, 50),
						"content_length": len(result.Content),
					}).Info("DeepSeek extracted content")
				}
			}
		}
	}

	// Теперь создаем новости с правильными категориями и извлеченным контентом
	for i, item := range items {
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

		// Используем результат DeepSeek-классификации
		var categoryID *int
		if deepSeekCategoryID, exists := deepSeekResults[i]; exists {
			categoryID = &deepSeekCategoryID
		} else {
			// Если DeepSeek-классификация не сработала, используем категорию по умолчанию
			defaultCategory := models.CategoryPolitics // "Политика" как fallback
			categoryID = &defaultCategory
		}

		// Определяем полный контент новости
		fullContent := item.Content
		contentSource := "rss_content"

		// Сначала пробуем использовать контент, извлеченный через AI
		if contentResult, exists := contentResults[i]; exists {
			fullContent = contentResult.Content
			contentSource = "ai_extraction"
			s.logger.WithFields(logrus.Fields{
				"url":            item.Link,
				"content_length": len(fullContent),
				"source":         contentSource,
			}).Debug("Using AI extracted content")
		} else if fullContent == "" {
			// Если AI-извлечение не удалось, пробуем извлечь с веб-страницы (fallback)
			if s.contentExtractor.IsValidURL(item.Link) {
				if extractedContent, err := s.contentExtractor.ExtractFullContent(ctx, item.Link); err == nil && extractedContent != "" {
					fullContent = extractedContent
					contentSource = "web_extraction"
					s.logger.WithFields(logrus.Fields{
						"url":            item.Link,
						"content_length": len(extractedContent),
						"source":         contentSource,
					}).Debug("Extracted full content from web page")
				} else {
					s.logger.WithFields(logrus.Fields{
						"url":    item.Link,
						"error":  err,
						"source": "web_extraction",
					}).Debug("Failed to extract full content from web page")
				}
			}
		}

		// Если не удалось извлечь с веб-страницы, используем описание из RSS
		if fullContent == "" {
			fullContent = item.Description
			contentSource = "rss_description"
			s.logger.WithFields(logrus.Fields{
				"url":            item.Link,
				"content_length": len(fullContent),
				"source":         contentSource,
			}).Debug("Using description from RSS feed")
		}

		// Если и описания нет, используем контент из RSS
		if fullContent == "" {
			fullContent = s.contentExtractor.ExtractContentFromRSS(item.Description, item.Content)
			contentSource = "rss_content"
			s.logger.WithFields(logrus.Fields{
				"url":            item.Link,
				"content_length": len(fullContent),
				"source":         contentSource,
			}).Debug("Using content from RSS feed")
		}

		// Определяем страну по контенту
		var detectedCountry *string
		if fullContent != "" {
			detectedCountry = s.countryDetector.DetectCountry(item.Title, item.Description, fullContent)
		}

		// Создаем объект новости с очисткой UTF-8
		news := models.News{
			Title:          cleanUTF8(item.Title),
			Description:    cleanUTF8(item.Description),
			Content:        cleanUTF8(fullContent),
			URL:            item.Link,
			ImageURL:       item.ImageURL,
			Author:         cleanUTF8(item.Author),
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

// shouldUseAIClassifier определяет, нужно ли использовать AI-классификатор
func (s *ParsingService) shouldUseAIClassifier(categoryID *int, title, description string) bool {
	// Если категория не определена, используем AI
	if categoryID == nil {
		return true
	}

	// Если категория "Наука и техника" (ID=4), но заголовок содержит политические ключевые слова
	if *categoryID == 4 {
		politicalKeywords := []string{
			"президент", "премьер", "министр", "правительство", "парламент", "выборы",
			"политика", "дипломатия", "санкции", "соглашение", "договор", "переговоры",
			"кремль", "белый дом", "конгресс", "сенат", "нато", "ес", "оон",
			"война", "конфликт", "мир", "безопасность", "оборона", "военные",
			"экономика", "торговля", "бизнес", "финансы", "инвестиции",
		}

		text := strings.ToLower(title + " " + description)
		for _, keyword := range politicalKeywords {
			if strings.Contains(text, keyword) {
				s.logger.WithFields(logrus.Fields{
					"title":       title,
					"category_id": *categoryID,
					"keyword":     keyword,
				}).Info("Detected political content in science category, will use AI classifier")
				return true
			}
		}
	}

	// Если категория "Спорт" (ID=13), но заголовок не содержит спортивных ключевых слов
	if *categoryID == 13 {
		sportKeywords := []string{
			"футбол", "хоккей", "баскетбол", "теннис", "бокс", "олимпиада", "чемпионат",
			"спорт", "игра", "матч", "команда", "игрок", "тренер", "стадион",
			"нхл", "нба", "фнл", "уефа", "фифа", "молодежка", "юниор",
		}

		text := strings.ToLower(title + " " + description)
		hasSportKeyword := false
		for _, keyword := range sportKeywords {
			if strings.Contains(text, keyword) {
				hasSportKeyword = true
				break
			}
		}

		if !hasSportKeyword {
			s.logger.WithFields(logrus.Fields{
				"title":       title,
				"category_id": *categoryID,
			}).Info("Sport category assigned but no sport keywords found, will use AI classifier")
			return true
		}
	}

	return false
}

// extractImageFromFeed извлекает URL изображения из RSS ленты
func extractImageFromFeed(feed *gofeed.Feed) string {
	if feed.Image != nil && feed.Image.URL != "" {
		return feed.Image.URL
	}

	// Можно добавить логику для извлечения favicon
	return ""
}

// cleanUTF8 очищает текст от невалидных UTF-8 символов
func cleanUTF8(text string) string {
	if text == "" {
		return text
	}

	// Проверяем, является ли строка валидной UTF-8
	if utf8.ValidString(text) {
		return text
	}

	// Если строка невалидна, очищаем её
	var result strings.Builder
	for _, r := range text {
		if r == utf8.RuneError {
			// Заменяем невалидные символы на пробел
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}
