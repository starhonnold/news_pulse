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
	rssParser          *RSSParser
	newsSourceRepo     *repository.NewsSourceRepository
	newsRepo           *repository.NewsRepository
	parsingLogRepo     *repository.ParsingLogRepository
	config             *config.ParsingConfig
	logger             *logrus.Logger
	cron               *cron.Cron
	isRunning          bool
	mu                 sync.RWMutex
	semaphore          chan struct{}
	categoryClassifier *CategoryClassifier
	aiClassifier       *AIClassifier
	countryDetector    *CountryDetector
	contentExtractor   *ContentExtractor
}

// NewParsingService создает новый сервис парсинга
func NewParsingService(
	rssParser *RSSParser,
	newsSourceRepo *repository.NewsSourceRepository,
	newsRepo *repository.NewsRepository,
	parsingLogRepo *repository.ParsingLogRepository,
	config *config.ParsingConfig,
	logger *logrus.Logger,
) *ParsingService {
	// Создаем семафор для ограничения количества одновременных парсингов
	semaphore := make(chan struct{}, config.MaxConcurrentParsers)

	// Создаем cron для планирования задач
	cronScheduler := cron.New(cron.WithSeconds())

	// Создаем классификатор категорий
	classifier := NewCategoryClassifier(logger)

	// Создаем AI-классификатор (передаем nil, так как API ключ захардкожен)
	aiClassifier := NewAIClassifier(nil, logger)

	// Создаем детектор стран
	countryDetector := NewCountryDetector(logger)

	// Создаем извлекатель контента (упрощенный)
	contentExtractor := NewContentExtractor(logger, nil)

	return &ParsingService{
		rssParser:          rssParser,
		newsSourceRepo:     newsSourceRepo,
		newsRepo:           newsRepo,
		parsingLogRepo:     parsingLogRepo,
		config:             config,
		logger:             logger,
		cron:               cronScheduler,
		semaphore:          semaphore,
		categoryClassifier: classifier,
		aiClassifier:       aiClassifier,
		countryDetector:    countryDetector,
		contentExtractor:   contentExtractor,
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
	var itemsForAIClassification []NewsItem

	// Сначала обрабатываем все элементы и собираем те, которые нуждаются в AI-классификации
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

		// Все новости классифицируем через AI для точности
		itemsForAIClassification = append(itemsForAIClassification, NewsItem{
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Index:       i,
		})
	}

	// Выполняем batch-классификацию для элементов, которые нуждаются в AI-классификации
	aiResults := make(map[int]int)
	if len(itemsForAIClassification) > 0 {
		s.logger.WithField("items_count", len(itemsForAIClassification)).Info("Performing batch AI classification")

		// Ограничиваем количество элементов для batch-классификации (максимум 50)
		batchSize := 50
		if len(itemsForAIClassification) > batchSize {
			itemsForAIClassification = itemsForAIClassification[:batchSize]
		}

		results, err := s.aiClassifier.ClassifyNewsBatch(ctx, itemsForAIClassification)
		if err != nil {
			s.logger.WithError(err).Warn("Batch AI classification failed")
		} else {
			// Создаем карту результатов AI-классификации
			for _, result := range results {
				if result.Error == nil {
					aiResults[result.Index] = result.CategoryID
					s.logger.WithFields(logrus.Fields{
						"index":       result.Index,
						"category_id": result.CategoryID,
					}).Info("AI classified news category")
				}
			}
		}
	}

	// Теперь создаем новости с правильными категориями
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

		// Используем результат AI-классификации
		var categoryID *int
		if aiCategoryID, exists := aiResults[i]; exists {
			categoryID = &aiCategoryID
		} else {
			// Если AI-классификация не сработала, используем категорию по умолчанию
			defaultCategory := 7 // "Из жизни"
			categoryID = &defaultCategory
		}

		// Сначала пробуем извлечь полный контент с помощью AI
		fullContent := item.Content

		// Если контента нет в RSS, пробуем извлечь с веб-страницы (без AI для экономии API запросов)
		if fullContent == "" && s.contentExtractor.IsValidURL(item.Link) {
			if extractedContent, err := s.contentExtractor.ExtractFullContent(ctx, item.Link); err == nil && extractedContent != "" {
				fullContent = extractedContent
				s.logger.WithFields(logrus.Fields{
					"url":            item.Link,
					"content_length": len(extractedContent),
					"source":         "web",
				}).Debug("Extracted full content from web page")
			} else {
				s.logger.WithFields(logrus.Fields{
					"url":    item.Link,
					"error":  err,
					"source": "web",
				}).Debug("Failed to extract full content from web page")
			}
		}

		// Если не удалось извлечь с веб-страницы, используем описание из RSS
		if fullContent == "" {
			fullContent = item.Description
			s.logger.WithFields(logrus.Fields{
				"url":            item.Link,
				"content_length": len(fullContent),
				"source":         "rss_description",
			}).Debug("Using description from RSS feed")
		}

		// Если и описания нет, используем контент из RSS
		if fullContent == "" {
			fullContent = s.contentExtractor.ExtractContentFromRSS(item.Description, item.Content)
			s.logger.WithFields(logrus.Fields{
				"url":            item.Link,
				"content_length": len(fullContent),
				"source":         "rss_content",
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
