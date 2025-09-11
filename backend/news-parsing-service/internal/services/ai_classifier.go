package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"news-parsing-service/internal/config"

	"github.com/sirupsen/logrus"
)

// OpenRouterRequest структура запроса к OpenRouter API
type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message структура сообщения для OpenRouter
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse структура ответа от OpenRouter API
type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
	Error   *Error   `json:"error,omitempty"`
}

// Choice структура выбора из ответа
type Choice struct {
	Message Message `json:"message"`
}

// Error структура ошибки
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// AIClassifier сервис для классификации новостей с помощью AI
type AIClassifier struct {
	config      *config.Config
	logger      *logrus.Logger
	httpClient  *http.Client
	apiKey      string
	lastRequest time.Time
}

// NewAIClassifier создает новый экземпляр AI-классификатора
func NewAIClassifier(config *config.Config, logger *logrus.Logger) *AIClassifier {
	return &AIClassifier{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: config.OpenAIAPIKey,
	}
}

// NewsItem представляет новость для batch-классификации
type NewsItem struct {
	Title       string
	Description string
	Content     string
	Index       int // Индекс для сопоставления ответа
}

// BatchClassificationResult результат batch-классификации
type BatchClassificationResult struct {
	Index      int
	CategoryID int
	Error      error
}

// ClassifyNews классифицирует новость с помощью AI (для обратной совместимости)
func (ai *AIClassifier) ClassifyNews(ctx context.Context, title, description, content string) (*int, error) {
	// Для обратной совместимости используем batch-классификацию с одной новостью
	items := []NewsItem{
		{
			Title:       title,
			Description: description,
			Content:     content,
			Index:       0,
		},
	}

	results, err := ai.ClassifyNewsBatch(ctx, items)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 || results[0].Error != nil {
		return nil, fmt.Errorf("batch classification failed")
	}

	return &results[0].CategoryID, nil
}

// ClassifyNewsBatch классифицирует множество новостей одним запросом
func (ai *AIClassifier) ClassifyNewsBatch(ctx context.Context, items []NewsItem) ([]BatchClassificationResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to classify")
	}

	// Получаем список доступных категорий
	categories := ai.getAvailableCategories()

	// Формируем batch-промпт
	prompt := ai.buildBatchPrompt(items, categories)

	// Отправляем запрос к OpenRouter
	response, err := ai.sendRequest(ctx, prompt)
	if err != nil {
		ai.logger.WithError(err).Error("Failed to send batch request to OpenRouter")
		return nil, err
	}

	// Парсим batch-ответ
	results, err := ai.parseBatchResponse(response, items, categories)
	if err != nil {
		ai.logger.WithError(err).Error("Failed to parse batch AI response")
		return nil, err
	}

	ai.logger.WithFields(logrus.Fields{
		"items_count":   len(items),
		"results_count": len(results),
		"ai_response":   response,
	}).Info("AI batch classified news categories")

	return results, nil
}

// getAvailableCategories возвращает список доступных категорий
func (ai *AIClassifier) getAvailableCategories() map[string]int {
	return map[string]int{
		"Россия":            1,
		"Моя страна":        2,
		"Бывший СССР":       3,
		"Наука и техника":   4,
		"Путешествия":       5,
		"Интернет и СМИ":    6,
		"Из жизни":          7,
		"Силовые структуры": 8,
		"Ценности":          9,
		"Забота о себе":     10,
		"Мир":               11,
		"Бизнес":            12,
		"Спорт":             13,
	}
}

// buildBatchPrompt создает batch-промпт для AI
func (ai *AIClassifier) buildBatchPrompt(items []NewsItem, categories map[string]int) string {
	var categoryList strings.Builder
	for category := range categories {
		categoryList.WriteString(fmt.Sprintf("- %s\n", category))
	}

	var newsList strings.Builder
	for i, item := range items {
		// Берем первые 200 символов контента для экономии токенов
		contentPreview := item.Content
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "..."
		}

		newsList.WriteString(fmt.Sprintf("%d. Заголовок: %s\n   Описание: %s\n   Содержание: %s\n\n",
			i+1, item.Title, item.Description, contentPreview))
	}

	prompt := fmt.Sprintf(`Ты - эксперт по классификации новостей. Определи категорию для каждой новости из предложенного списка.

Доступные категории:
%s

Новости для классификации:
%s

Правила классификации:
- "Россия" - новости о России, политике, экономике, обществе
- "Моя страна" - новости о стране пользователя (если не Россия)
- "Бывший СССР" - новости о странах бывшего СССР (кроме России)
- "Наука и техника" - технологии, IT, научные открытия, инновации
- "Путешествия" - туризм, путешествия, достопримечательности
- "Интернет и СМИ" - новости о медиа, интернете, социальных сетях
- "Из жизни" - бытовые новости, происшествия, социальные темы
- "Силовые структуры" - военные, полиция, спецслужбы, безопасность
- "Ценности" - культура, религия, мораль, традиции
- "Забота о себе" - здоровье, медицина, спорт, фитнес
- "Мир" - международные новости, геополитика
- "Бизнес" - экономика, финансы, бизнес, инвестиции
- "Спорт" - спортивные новости, соревнования, спортсмены

Ответь в формате JSON:
{
  "classifications": [
    {"index": 1, "category": "Россия"},
    {"index": 2, "category": "Спорт"},
    ...
  ]
}`,
		categoryList.String(), newsList.String())

	return prompt
}

// buildPrompt создает промпт для AI (для обратной совместимости)
func (ai *AIClassifier) buildPrompt(title, description, content string, categories map[string]int) string {
	var categoryList strings.Builder
	for category := range categories {
		categoryList.WriteString(fmt.Sprintf("- %s\n", category))
	}

	// Берем первые 500 символов контента для экономии токенов
	contentPreview := content
	if len(content) > 500 {
		contentPreview = content[:500] + "..."
	}

	prompt := fmt.Sprintf(`Ты - эксперт по классификации новостей. Определи категорию для следующей новости из предложенного списка.

Доступные категории:
%s

Заголовок: %s
Описание: %s
Содержание: %s

Правила классификации:
- "Россия" - новости о России, политике, экономике, обществе
- "Моя страна" - новости о стране пользователя (если не Россия)
- "Бывший СССР" - новости о странах бывшего СССР (кроме России)
- "Наука и техника" - технологии, IT, научные открытия, инновации
- "Путешествия" - туризм, путешествия, достопримечательности
- "Интернет и СМИ" - новости о медиа, интернете, социальных сетях
- "Из жизни" - бытовые новости, происшествия, социальные темы
- "Силовые структуры" - военные, полиция, спецслужбы, безопасность
- "Ценности" - культура, религия, мораль, традиции
- "Забота о себе" - здоровье, медицина, спорт, фитнес
- "Мир" - международные новости, геополитика
- "Бизнес" - экономика, финансы, бизнес, инвестиции
- "Спорт" - спортивные новости, соревнования, спортсмены

Ответь ТОЛЬКО названием категории без дополнительных объяснений.`,
		categoryList.String(), title, description, contentPreview)

	return prompt
}

// sendRequest отправляет запрос к OpenRouter API
func (ai *AIClassifier) sendRequest(ctx context.Context, prompt string) (string, error) {
	// Добавляем небольшую задержку между запросами (собственный API ключ)
	timeSinceLastRequest := time.Since(ai.lastRequest)
	if timeSinceLastRequest < 2*time.Second {
		delay := 2*time.Second - timeSinceLastRequest
		ai.logger.WithField("delay", delay).Debug("Rate limiting delay")
		time.Sleep(delay)
	}
	ai.lastRequest = time.Now()
	request := OpenRouterRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1, // Низкая температура для более точных ответов
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)
	req.Header.Set("HTTP-Referer", "https://news-pulse.local")
	req.Header.Set("X-Title", "News Pulse AI Classifier")

	resp, err := ai.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		ai.logger.Warn("Rate limited by OpenRouter API, will retry with longer delay")
		// Увеличиваем задержку для следующего запроса
		ai.lastRequest = time.Now().Add(-10 * time.Second)
		return "", fmt.Errorf("rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var response OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}

// BatchResponse структура для парсинга batch-ответа
type BatchResponse struct {
	Classifications []ClassificationItem `json:"classifications"`
}

// ClassificationItem элемент классификации
type ClassificationItem struct {
	Index    int    `json:"index"`
	Category string `json:"category"`
}

// parseBatchResponse парсит batch-ответ AI
func (ai *AIClassifier) parseBatchResponse(response string, items []NewsItem, categories map[string]int) ([]BatchClassificationResult, error) {
	// Очищаем ответ от лишних символов
	response = strings.TrimSpace(response)

	// Если ответ пустой, возвращаем ошибку
	if response == "" {
		return nil, fmt.Errorf("empty AI response")
	}

	// Пытаемся найти JSON в ответе
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var batchResponse BatchResponse
	if err := json.Unmarshal([]byte(jsonStr), &batchResponse); err != nil {
		ai.logger.WithFields(logrus.Fields{
			"response": response,
			"json_str": jsonStr,
			"error":    err,
		}).Error("Failed to parse JSON response")
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Создаем результаты
	results := make([]BatchClassificationResult, len(items))

	// Инициализируем все результаты как ошибки
	for i := range results {
		results[i] = BatchClassificationResult{
			Index:      i,
			CategoryID: 7, // "Из жизни" по умолчанию
			Error:      fmt.Errorf("no classification found"),
		}
	}

	// Обрабатываем классификации
	for _, classification := range batchResponse.Classifications {
		index := classification.Index - 1 // Индексы в JSON начинаются с 1
		if index < 0 || index >= len(items) {
			ai.logger.WithFields(logrus.Fields{
				"index":       classification.Index,
				"items_count": len(items),
			}).Warn("Invalid index in AI response")
			continue
		}

		// Ищем категорию
		if categoryID, exists := categories[classification.Category]; exists {
			results[index] = BatchClassificationResult{
				Index:      index,
				CategoryID: categoryID,
				Error:      nil,
			}
		} else {
			// Ищем частичное совпадение
			found := false
			for category, categoryID := range categories {
				if strings.Contains(strings.ToLower(classification.Category), strings.ToLower(category)) {
					results[index] = BatchClassificationResult{
						Index:      index,
						CategoryID: categoryID,
						Error:      nil,
					}
					found = true
					break
				}
			}

			if !found {
				ai.logger.WithFields(logrus.Fields{
					"index":    index,
					"category": classification.Category,
				}).Warn("Unknown category in AI response")
			}
		}
	}

	return results, nil
}

// parseResponse парсит ответ AI и возвращает ID категории (для обратной совместимости)
func (ai *AIClassifier) parseResponse(response string, categories map[string]int) (*int, error) {
	// Очищаем ответ от лишних символов
	response = strings.TrimSpace(response)
	response = strings.Trim(response, "\"'")

	// Если ответ пустой, возвращаем ошибку
	if response == "" {
		return nil, fmt.Errorf("empty AI response")
	}

	// Ищем точное совпадение
	if categoryID, exists := categories[response]; exists {
		return &categoryID, nil
	}

	// Ищем частичное совпадение
	for category, categoryID := range categories {
		if strings.Contains(strings.ToLower(response), strings.ToLower(category)) {
			ai.logger.WithFields(logrus.Fields{
				"ai_response":      response,
				"matched_category": category,
				"category_id":      categoryID,
			}).Info("Partial match found for AI response")
			return &categoryID, nil
		}
	}

	// Если ничего не найдено, возвращаем категорию "Из жизни" по умолчанию
	defaultCategory := 7 // "Из жизни"
	ai.logger.WithFields(logrus.Fields{
		"ai_response":         response,
		"default_category_id": defaultCategory,
	}).Warn("No category match found, using default")

	return &defaultCategory, nil
}
