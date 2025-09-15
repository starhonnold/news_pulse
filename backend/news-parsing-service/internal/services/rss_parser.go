package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/config"
	"news-parsing-service/internal/models"
)

// RSSParser представляет парсер RSS лент
type RSSParser struct {
	client *http.Client
	parser *gofeed.Parser
	config *config.ParsingConfig
	logger *logrus.Logger
}

// NewRSSParser создает новый парсер RSS лент
func NewRSSParser(cfg *config.ParsingConfig, proxyConfig *config.ProxyConfig, logger *logrus.Logger) *RSSParser {
	// Настройка HTTP клиента
	var transport *http.Transport

	if proxyConfig != nil && proxyConfig.Enabled && proxyConfig.URL != "" {
		// Настройка прокси
		proxyURL, err := url.Parse(proxyConfig.URL)
		if err != nil {
			logger.WithError(err).Error("Failed to parse proxy URL for RSS parser")
			transport = &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 2,
			}
		} else {
			// Добавляем аутентификацию, если указана
			if proxyConfig.Username != "" && proxyConfig.Password != "" {
				proxyURL.User = url.UserPassword(proxyConfig.Username, proxyConfig.Password)
			}

			transport = &http.Transport{
				Proxy:               http.ProxyURL(proxyURL),
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 2,
			}

			logger.WithField("proxy_url", proxyConfig.URL).Info("Using proxy for RSS parsing")
		}
	} else {
		transport = &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 2,
		}
	}

	client := &http.Client{
		Timeout:   cfg.RequestTimeout,
		Transport: transport,
	}

	// Создание парсера gofeed
	parser := gofeed.NewParser()

	return &RSSParser{
		client: client,
		parser: parser,
		config: cfg,
		logger: logger,
	}
}

// ParseFeed парсит RSS ленту из указанного источника
func (p *RSSParser) ParseFeed(ctx context.Context, source models.NewsSource) models.FeedParseResult {
	startTime := time.Now()

	result := models.FeedParseResult{
		SourceID: source.ID,
		ParsedAt: startTime,
		Success:  false,
		Items:    []models.ParsedFeedItem{},
	}

	p.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"source_name": source.Name,
		"rss_url":     source.RSSURL,
	}).Debug("Starting RSS feed parsing")

	// Создание HTTP запроса
	req, err := http.NewRequestWithContext(ctx, "GET", source.RSSURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Установка заголовков
	req.Header.Set("User-Agent", p.config.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("Cache-Control", "no-cache")

	// Выполнение HTTP запроса
	resp, err := p.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch RSS feed: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}
	defer resp.Body.Close()

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Проверка размера ответа
	if resp.ContentLength > p.config.MaxFeedSize {
		result.Error = fmt.Sprintf("feed size too large: %d bytes", resp.ContentLength)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Ограничение размера читаемых данных
	limitedReader := io.LimitReader(resp.Body, p.config.MaxFeedSize)

	// Парсинг RSS ленты
	feed, err := p.parser.Parse(limitedReader)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse RSS feed: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Обработка элементов ленты
	items := p.processFeedItems(feed, source)

	result.Items = items
	result.Success = true
	result.ExecutionTime = time.Since(startTime)

	p.logger.WithFields(logrus.Fields{
		"source_id":      source.ID,
		"source_name":    source.Name,
		"items_count":    len(items),
		"execution_time": result.ExecutionTime,
	}).Info("Successfully parsed RSS feed")

	return result
}

// processFeedItems обрабатывает элементы RSS ленты
func (p *RSSParser) processFeedItems(feed *gofeed.Feed, source models.NewsSource) []models.ParsedFeedItem {
	var items []models.ParsedFeedItem

	for _, item := range feed.Items {
		// Пропускаем элементы без заголовка или ссылки
		if item.Title == "" || item.Link == "" {
			continue
		}

		// Валидация длины заголовка
		title := strings.TrimSpace(item.Title)
		if len(title) < p.config.MinTitleLength || len(title) > p.config.MaxTitleLength {
			p.logger.WithFields(logrus.Fields{
				"source_id": source.ID,
				"title":     title,
				"length":    len(title),
			}).Debug("Skipping item with invalid title length")
			continue
		}

		// Определение времени публикации
		publishedTime := time.Now()
		if item.PublishedParsed != nil {
			publishedTime = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedTime = *item.UpdatedParsed
		}

		// Извлечение описания
		description := ""
		if item.Description != "" {
			description = p.cleanHTMLContent(item.Description)
		}

		// Извлечение контента
		content := ""
		if item.Content != "" {
			content = p.cleanHTMLContent(item.Content)
		}

		// Извлечение автора
		author := ""
		if item.Author != nil && item.Author.Name != "" {
			author = item.Author.Name
		}

		// Извлечение изображения
		imageURL := ""
		if item.Image != nil && item.Image.URL != "" {
			imageURL = item.Image.URL
		} else if item.Enclosures != nil {
			for _, enclosure := range item.Enclosures {
				if strings.HasPrefix(enclosure.Type, "image/") {
					imageURL = enclosure.URL
					break
				}
			}
		}

		// Извлечение категорий
		var categories []string
		if item.Categories != nil {
			categories = item.Categories
		}

		// GUID для дедупликации
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}

		parsedItem := models.ParsedFeedItem{
			Title:       title,
			Description: description,
			Content:     content,
			Link:        item.Link,
			Author:      author,
			Published:   publishedTime,
			ImageURL:    imageURL,
			GUID:        guid,
			Categories:  categories,
		}

		items = append(items, parsedItem)

		// Ограничиваем количество обрабатываемых элементов
		if len(items) >= p.config.BatchSize {
			break
		}
	}

	p.logger.WithFields(logrus.Fields{
		"source_id":    source.ID,
		"total_items":  len(feed.Items),
		"parsed_items": len(items),
	}).Debug("Processed feed items")

	return items
}

// cleanHTMLContent очищает HTML контент от тегов
func (p *RSSParser) cleanHTMLContent(content string) string {
	// Простая очистка HTML тегов
	// В production рекомендуется использовать более продвинутую библиотеку
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")
	content = strings.ReplaceAll(content, "</p>", "\n\n")

	// Удаление HTML тегов (простая реализация)
	for strings.Contains(content, "<") && strings.Contains(content, ">") {
		start := strings.Index(content, "<")
		end := strings.Index(content[start:], ">")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+1:]
	}

	// Очистка от лишних пробелов и переносов
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	result := strings.Join(cleanLines, "\n")

	// Ограничиваем длину контента
	if len(result) > 10000 {
		result = result[:10000] + "..."
	}

	return result
}

// ValidateFeed проверяет доступность и корректность RSS ленты
func (p *RSSParser) ValidateFeed(ctx context.Context, rssURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", rssURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", p.config.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Проверяем, что это валидный RSS/XML
	limitedReader := io.LimitReader(resp.Body, 1024*1024) // 1MB для проверки
	_, err = p.parser.Parse(limitedReader)
	if err != nil {
		return fmt.Errorf("invalid RSS feed format: %w", err)
	}

	return nil
}

// GetFeedInfo возвращает информацию о RSS ленте
func (p *RSSParser) GetFeedInfo(ctx context.Context, rssURL string) (*gofeed.Feed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", p.config.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	limitedReader := io.LimitReader(resp.Body, p.config.MaxFeedSize)
	feed, err := p.parser.Parse(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	return feed, nil
}

// Close закрывает ресурсы парсера
func (p *RSSParser) Close() {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
}
