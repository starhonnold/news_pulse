package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// ContentExtractor представляет извлекатель контента с веб-страниц
type ContentExtractor struct {
	logger *logrus.Logger
	client *http.Client
	config *ContentExtractorConfig
}

// ContentExtractorConfig представляет конфигурацию извлекателя контента
type ContentExtractorConfig struct {
	RequestTimeout   time.Duration
	MaxContentSize   int64
	UserAgent        string
	EnableFullText   bool
	ContentSelectors []string // CSS селекторы для извлечения контента
	ExcludeSelectors []string // CSS селекторы для исключения контента
}

// NewContentExtractor создает новый извлекатель контента
func NewContentExtractor(logger *logrus.Logger, config *ContentExtractorConfig) *ContentExtractor {
	if config == nil {
		config = &ContentExtractorConfig{
			RequestTimeout:   30 * time.Second,
			MaxContentSize:   5 * 1024 * 1024, // 5MB
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			EnableFullText:   true,
			ContentSelectors: []string{"article", ".article", ".content", ".post", ".news-content", "main", ".main-content"},
			ExcludeSelectors: []string{"script", "style", "nav", "header", "footer", ".advertisement", ".ads", ".sidebar"},
		}
	}

	client := &http.Client{
		Timeout: config.RequestTimeout,
	}

	return &ContentExtractor{
		logger: logger,
		client: client,
		config: config,
	}
}

// ExtractFullContent извлекает полный текст статьи с веб-страницы
func (e *ContentExtractor) ExtractFullContent(ctx context.Context, url string) (string, error) {
	if !e.config.EnableFullText {
		return "", nil
	}

	e.logger.WithField("url", url).Debug("Extracting full content from URL")

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("User-Agent", e.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Выполняем запрос
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Проверяем размер контента
	if resp.ContentLength > e.config.MaxContentSize {
		return "", fmt.Errorf("content too large: %d bytes", resp.ContentLength)
	}

	// Ограничиваем размер читаемых данных
	limitedReader := io.LimitReader(resp.Body, e.config.MaxContentSize)

	// Парсим HTML
	doc, err := html.Parse(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Извлекаем контент
	content := e.extractTextFromNode(doc)

	// Очищаем и форматируем текст
	content = e.cleanText(content)

	e.logger.WithFields(logrus.Fields{
		"url":            url,
		"content_length": len(content),
	}).Debug("Extracted full content")

	return content, nil
}

// extractTextFromNode извлекает текст из HTML узла
func (e *ContentExtractor) extractTextFromNode(n *html.Node) string {
	var result strings.Builder

	// Проверяем, нужно ли исключить этот узел
	if e.shouldExcludeNode(n) {
		return ""
	}

	// Если это текстовый узел, добавляем его содержимое
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			result.WriteString(text)
			result.WriteString(" ")
		}
	}

	// Рекурсивно обрабатываем дочерние узлы
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		childText := e.extractTextFromNode(c)
		if childText != "" {
			result.WriteString(childText)
		}
	}

	return result.String()
}

// shouldExcludeNode проверяет, нужно ли исключить узел
func (e *ContentExtractor) shouldExcludeNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	// Проверяем теги для исключения
	excludeTags := map[string]bool{
		"script": true, "style": true, "nav": true, "header": true,
		"footer": true, "aside": true, "noscript": true, "iframe": true,
	}

	if excludeTags[n.Data] {
		return true
	}

	// Проверяем CSS классы для исключения
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			class := strings.ToLower(attr.Val)
			for _, excludeClass := range e.config.ExcludeSelectors {
				if strings.Contains(class, strings.ToLower(excludeClass)) {
					return true
				}
			}
		}
	}

	return false
}

// cleanText очищает и форматирует извлеченный текст
func (e *ContentExtractor) cleanText(text string) string {
	// Удаляем лишние пробелы и переносы строк
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Удаляем HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Удаляем повторяющиеся знаки препинания
	text = regexp.MustCompile(`[.]{2,}`).ReplaceAllString(text, ".")
	text = regexp.MustCompile(`[!]{2,}`).ReplaceAllString(text, "!")
	text = regexp.MustCompile(`[?]{2,}`).ReplaceAllString(text, "?")

	// Ограничиваем длину контента
	maxLength := 50000 // 50KB текста
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	return text
}

// ExtractContentFromRSS извлекает контент из RSS элемента
func (e *ContentExtractor) ExtractContentFromRSS(description, content string) string {
	// Если есть полный контент, используем его
	if content != "" {
		return e.cleanText(content)
	}

	// Иначе используем описание
	if description != "" {
		return e.cleanText(description)
	}

	return ""
}

// IsValidURL проверяет, является ли URL валидным для извлечения контента
func (e *ContentExtractor) IsValidURL(url string) bool {
	// Простая проверка URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// Исключаем некоторые типы URL
	excludePatterns := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".wmv",
		".zip", ".rar", ".7z", ".tar", ".gz",
	}

	urlLower := strings.ToLower(url)
	for _, pattern := range excludePatterns {
		if strings.Contains(urlLower, pattern) {
			return false
		}
	}

	return true
}

// OpenRouterContentRequest структура запроса к OpenRouter API для извлечения контента
type OpenRouterContentRequest struct {
	Model       string           `json:"model"`
	Messages    []ContentMessage `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

// ContentMessage структура сообщения для OpenRouter
type ContentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterContentResponse структура ответа от OpenRouter API
type OpenRouterContentResponse struct {
	Choices []ContentChoice `json:"choices"`
	Error   *ContentError   `json:"error,omitempty"`
}

// ContentChoice структура выбора из ответа
type ContentChoice struct {
	Message ContentMessage `json:"message"`
}

// ContentError структура ошибки
type ContentError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ExtractFullContentWithAI извлекает полный текст статьи с помощью OpenRouter AI
func (e *ContentExtractor) ExtractFullContentWithAI(ctx context.Context, url string) (string, error) {
	e.logger.WithField("url", url).Debug("Extracting full content using AI")

	// API ключ OpenAI
	apiKey := e.config.OpenAIAPIKey

	// Формируем промпт для AI
	prompt := fmt.Sprintf(`Пожалуйста, извлеки полный текст новости по ссылке: %s

Требования:
1. Извлеки только основной текст новости, без рекламы, навигации, комментариев
2. Сохрани структуру текста (абзацы, заголовки)
3. Удали все HTML-теги и оставь только чистый текст
4. Если новость недоступна или не найдена, верни "Новость недоступна"
5. Ограничь текст до 5000 символов

Ответь только текстом новости, без дополнительных комментариев.`, url)

	// Создаем запрос к OpenRouter
	request := OpenRouterContentRequest{
		Model: "openai/gpt-4o-mini:online",
		Messages: []ContentMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1,
	}

	// Сериализуем запрос
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-OpenAI-API-Key", apiKey)
	req.Header.Set("HTTP-Referer", "https://news-pulse.local")
	req.Header.Set("X-Title", "News Pulse Content Extractor")

	// Выполняем запрос
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Парсим ответ
	var response OpenRouterContentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Проверяем на ошибки
	if response.Error != nil {
		return "", fmt.Errorf("OpenRouter error: %s", response.Error.Message)
	}

	// Извлекаем контент
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	// Если AI вернул "Новость недоступна", возвращаем пустую строку
	if strings.Contains(strings.ToLower(content), "новость недоступна") {
		return "", nil
	}

	e.logger.WithFields(logrus.Fields{
		"url":            url,
		"content_length": len(content),
	}).Debug("Extracted full content using AI")

	return content, nil
}
