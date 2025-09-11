package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DeepSeekContentExtractor извлекатель контента с помощью DeepSeek API
type DeepSeekContentExtractor struct {
	logger     *logrus.Logger
	httpClient *http.Client
	apiKey     string
	lastRequest time.Time
	model      string // Модель для использования
}

// DeepSeekRequest структура запроса к DeepSeek API
type DeepSeekRequest struct {
	Model       string                `json:"model"`
	Messages    []DeepSeekMessage     `json:"messages"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	Stream      bool                  `json:"stream,omitempty"`
}

// DeepSeekMessage структура сообщения для DeepSeek API
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse структура ответа от DeepSeek API
type DeepSeekResponse struct {
	Choices []DeepSeekChoice `json:"choices"`
	Error   *DeepSeekError   `json:"error,omitempty"`
	Usage   *DeepSeekUsage   `json:"usage,omitempty"`
}

// DeepSeekChoice структура выбора из ответа
type DeepSeekChoice struct {
	Message DeepSeekMessage `json:"message"`
}

// DeepSeekError структура ошибки
type DeepSeekError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// DeepSeekUsage структура использования токенов
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeepSeekContentExtractionItem элемент для batch-извлечения контента
type DeepSeekContentExtractionItem struct {
	URL   string
	Index int
}

// DeepSeekContentExtractionResult результат извлечения контента
type DeepSeekContentExtractionResult struct {
	Index   int
	Title   string
	Content string
	Error   error
	Usage   *DeepSeekUsage // Информация об использовании токенов
}

// DeepSeekBatchResponse структура для парсинга batch-ответа
type DeepSeekBatchResponse struct {
	Extractions []DeepSeekExtractionItem `json:"extractions"`
}

// DeepSeekExtractionItem элемент извлечения
type DeepSeekExtractionItem struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// NewDeepSeekContentExtractor создает новый DeepSeek-извлекатель контента
func NewDeepSeekContentExtractor(apiKey string, logger *logrus.Logger) *DeepSeekContentExtractor {
	return &DeepSeekContentExtractor{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: apiKey,
		model:  "deepseek-chat", // Используем основную модель DeepSeek
	}
}

// ExtractContentBatch извлекает контент множества новостей одним запросом
func (e *DeepSeekContentExtractor) ExtractContentBatch(ctx context.Context, items []DeepSeekContentExtractionItem) ([]DeepSeekContentExtractionResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to extract")
	}

	// Ограничиваем количество элементов для batch-извлечения
	batchSize := 3
	if len(items) > batchSize {
		items = items[:batchSize]
	}

	// Формируем batch-промпт
	prompt := e.buildBatchPrompt(items)

	// Отправляем запрос к DeepSeek API
	response, usage, err := e.sendRequest(ctx, prompt)
	if err != nil {
		e.logger.WithError(err).Error("Failed to send batch request to DeepSeek API")
		return nil, err
	}

	// Парсим batch-ответ
	results, err := e.parseBatchResponse(response, items, usage)
	if err != nil {
		e.logger.WithError(err).Error("Failed to parse batch DeepSeek response")
		return nil, err
	}

	e.logger.WithFields(logrus.Fields{
		"items_count":   len(items),
		"results_count": len(results),
		"model":         e.model,
		"usage":         usage,
	}).Info("DeepSeek batch extracted content")

	return results, nil
}

// ExtractSingleContent извлекает контент одной новости (для обратной совместимости)
func (e *DeepSeekContentExtractor) ExtractSingleContent(ctx context.Context, url string) (title, content string, err error) {
	items := []DeepSeekContentExtractionItem{
		{
			URL:   url,
			Index: 0,
		},
	}

	results, err := e.ExtractContentBatch(ctx, items)
	if err != nil {
		return "", "", err
	}

	if len(results) == 0 || results[0].Error != nil {
		return "", "", fmt.Errorf("content extraction failed")
	}

	return results[0].Title, results[0].Content, nil
}

// buildBatchPrompt создает batch-промпт для DeepSeek API
func (e *DeepSeekContentExtractor) buildBatchPrompt(items []DeepSeekContentExtractionItem) string {
	var urlList strings.Builder
	for i, item := range items {
		urlList.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.URL))
	}

	prompt := fmt.Sprintf(`Ты - эксперт по извлечению контента новостей. Мне нужно извлечь полный текст новостей по следующим ссылкам:

Ссылки для обработки:
%s

ТРЕБОВАНИЯ для каждой новости:
1. Перейди по ссылке и найди основную новость
2. Извлеки ТОЛЬКО основной текст новости (без рекламы, навигации, комментариев, подписей к фото)
3. Сохрани структуру текста (абзацы, заголовки)
4. Удали все HTML-теги и оставь только чистый текст
5. Если новость недоступна или не найдена, верни "Новость недоступна"
6. Ограничь текст до 2000 символов
7. Убедись, что текст в правильной кодировке (UTF-8) и без специальных символов

Ответь в формате JSON:
{
  "extractions": [
    {
      "index": 1,
      "title": "Заголовок новости",
      "content": "Полный текст новости без HTML-тегов"
    },
    {
      "index": 2,
      "title": "Заголовок новости",
      "content": "Полный текст новости без HTML-тегов"
    }
  ]
}

Если новость недоступна, верни:
{
  "index": 1,
  "title": "Новость недоступна",
  "content": "Новость недоступна"
}`, urlList.String())

	return prompt
}

// sendRequest отправляет запрос к DeepSeek API
func (e *DeepSeekContentExtractor) sendRequest(ctx context.Context, prompt string) (string, *DeepSeekUsage, error) {
	// Добавляем задержку между запросами для соблюдения rate limits
	timeSinceLastRequest := time.Since(e.lastRequest)
	if timeSinceLastRequest < 1*time.Second {
		delay := 1*time.Second - timeSinceLastRequest
		e.logger.WithField("delay", delay).Debug("Rate limiting delay")
		time.Sleep(delay)
	}
	e.lastRequest = time.Now()

	request := DeepSeekRequest{
		Model: e.model,
		Messages: []DeepSeekMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000, // Ограничиваем для экономии
		Temperature: 0.1,  // Низкая температура для точности
		Stream:      false, // Не используем streaming
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		e.logger.Warn("Rate limited by DeepSeek API, will retry with longer delay")
		// Увеличиваем задержку для следующего запроса
		e.lastRequest = time.Now().Add(-5 * time.Second)
		return "", nil, fmt.Errorf("rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var response DeepSeekResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", nil, fmt.Errorf("API error: %s (type: %s, code: %s)", response.Error.Message, response.Error.Type, response.Error.Code)
	}

	if len(response.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, response.Usage, nil
}

// parseBatchResponse парсит batch-ответ DeepSeek API
func (e *DeepSeekContentExtractor) parseBatchResponse(response string, items []DeepSeekContentExtractionItem, usage *DeepSeekUsage) ([]DeepSeekContentExtractionResult, error) {
	// Очищаем ответ от лишних символов
	response = strings.TrimSpace(response)

	// Если ответ пустой, возвращаем ошибку
	if response == "" {
		return nil, fmt.Errorf("empty DeepSeek response")
	}

	// Пытаемся найти JSON в ответе
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var batchResponse DeepSeekBatchResponse
	if err := json.Unmarshal([]byte(jsonStr), &batchResponse); err != nil {
		e.logger.WithFields(logrus.Fields{
			"response": response,
			"json_str": jsonStr,
			"error":    err,
		}).Error("Failed to parse JSON response")
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Создаем результаты
	results := make([]DeepSeekContentExtractionResult, len(items))

	// Инициализируем все результаты как ошибки
	for i := range results {
		results[i] = DeepSeekContentExtractionResult{
			Index:   i,
			Title:   "Не удалось извлечь",
			Content: "",
			Error:   fmt.Errorf("no extraction found"),
			Usage:   usage,
		}
	}

	// Обрабатываем извлечения
	for _, extraction := range batchResponse.Extractions {
		index := extraction.Index - 1 // Индексы в JSON начинаются с 1
		if index < 0 || index >= len(items) {
			e.logger.WithFields(logrus.Fields{
				"index":       extraction.Index,
				"items_count": len(items),
			}).Warn("Invalid index in DeepSeek response")
			continue
		}

		// Проверяем, доступна ли новость
		if extraction.Title == "Новость недоступна" || extraction.Content == "Новость недоступна" {
			results[index] = DeepSeekContentExtractionResult{
				Index:   index,
				Title:   "Новость недоступна",
				Content: "",
				Error:   fmt.Errorf("news not available"),
				Usage:   usage,
			}
		} else {
			results[index] = DeepSeekContentExtractionResult{
				Index:   index,
				Title:   extraction.Title,
				Content: extraction.Content,
				Error:   nil,
				Usage:   usage,
			}
		}
	}

	return results, nil
}

// IsValidURL проверяет, является ли URL валидным для извлечения контента
func (e *DeepSeekContentExtractor) IsValidURL(url string) bool {
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

// GetModelInfo возвращает информацию о используемой модели
func (e *DeepSeekContentExtractor) GetModelInfo() string {
	return fmt.Sprintf("DeepSeek API с моделью %s (стоимость: $0.07/1M входных токенов, $1.10/1M выходных токенов)", e.model)
}

// GetUsageInfo возвращает информацию об использовании токенов
func (e *DeepSeekContentExtractor) GetUsageInfo(usage *DeepSeekUsage) string {
	if usage == nil {
		return "Информация об использовании токенов недоступна"
	}
	
	cost := float64(usage.PromptTokens)*0.07/1000000 + float64(usage.CompletionTokens)*1.10/1000000
	return fmt.Sprintf("Использовано токенов: %d входных, %d выходных, %d всего. Примерная стоимость: $%.4f", 
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, cost)
}
