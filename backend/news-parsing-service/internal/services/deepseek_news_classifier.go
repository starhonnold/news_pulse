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

	"news-parsing-service/internal/models"
)

// DeepSeekNewsClassifier классификатор новостей с помощью DeepSeek API
type DeepSeekNewsClassifier struct {
	logger      *logrus.Logger
	httpClient  *http.Client
	apiKey      string
	lastRequest time.Time
	model       string
}

// DeepSeekClassificationRequest структура запроса к DeepSeek API
type DeepSeekClassificationRequest struct {
	Model       string                          `json:"model"`
	Messages    []DeepSeekClassificationMessage `json:"messages"`
	MaxTokens   int                             `json:"max_tokens,omitempty"`
	Temperature float64                         `json:"temperature,omitempty"`
}

// DeepSeekClassificationMessage структура сообщения для DeepSeek API
type DeepSeekClassificationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekClassificationResponse структура ответа от DeepSeek API
type DeepSeekClassificationResponse struct {
	Choices []DeepSeekClassificationChoice `json:"choices"`
	Error   *DeepSeekClassificationError   `json:"error,omitempty"`
}

// DeepSeekClassificationChoice структура выбора из ответа
type DeepSeekClassificationChoice struct {
	Message DeepSeekClassificationMessage `json:"message"`
}

// DeepSeekClassificationError структура ошибки
type DeepSeekClassificationError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// DeepSeekNewsItem элемент для batch-классификации новостей
type DeepSeekNewsItem struct {
	Index       int
	Title       string
	Description string
	Content     string
	Categories  []string
}

// DeepSeekClassificationResult результат классификации новости
type DeepSeekClassificationResult struct {
	Index      int
	CategoryID int
	Confidence float64
	Error      error
}

// DeepSeekBatchClassificationResponse структура для парсинга batch-ответа
type DeepSeekBatchClassificationResponse struct {
	Classifications []DeepSeekClassificationItem `json:"classifications"`
}

// DeepSeekClassificationItem элемент классификации
type DeepSeekClassificationItem struct {
	Index      int     `json:"index"`
	CategoryID int     `json:"category_id"`
	Confidence float64 `json:"confidence"`
}

// NewDeepSeekNewsClassifier создает новый DeepSeek-классификатор новостей
func NewDeepSeekNewsClassifier(apiKey string, logger *logrus.Logger) *DeepSeekNewsClassifier {
	return &DeepSeekNewsClassifier{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: apiKey,
		model:  "deepseek-chat",
	}
}

// ClassifyNewsBatch классифицирует множество новостей одним запросом
func (c *DeepSeekNewsClassifier) ClassifyNewsBatch(ctx context.Context, items []DeepSeekNewsItem) ([]DeepSeekClassificationResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to classify")
	}

	// Ограничиваем количество элементов для batch-классификации
	batchSize := 10
	if len(items) > batchSize {
		items = items[:batchSize]
	}

	// Формируем batch-промпт
	prompt := c.buildBatchClassificationPrompt(items)

	// Отправляем запрос к DeepSeek API
	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		c.logger.WithError(err).Error("Failed to send batch classification request to DeepSeek API")
		return nil, err
	}

	// Парсим batch-ответ
	results, err := c.parseBatchClassificationResponse(response, items)
	if err != nil {
		c.logger.WithError(err).Error("Failed to parse batch DeepSeek classification response")
		return nil, err
	}

	c.logger.WithFields(logrus.Fields{
		"items_count":   len(items),
		"results_count": len(results),
		"model":         c.model,
	}).Info("DeepSeek batch classified news")

	return results, nil
}

// ClassifySingleNews классифицирует одну новость (для обратной совместимости)
func (c *DeepSeekNewsClassifier) ClassifySingleNews(ctx context.Context, title, description, content string, categories []string) (categoryID int, confidence float64, err error) {
	items := []DeepSeekNewsItem{
		{
			Index:       0,
			Title:       title,
			Description: description,
			Content:     content,
			Categories:  categories,
		},
	}

	results, err := c.ClassifyNewsBatch(ctx, items)
	if err != nil {
		return 0, 0, err
	}

	if len(results) == 0 || results[0].Error != nil {
		return 0, 0, fmt.Errorf("news classification failed")
	}

	return results[0].CategoryID, results[0].Confidence, nil
}

// buildBatchClassificationPrompt создает batch-промпт для DeepSeek API
func (c *DeepSeekNewsClassifier) buildBatchClassificationPrompt(items []DeepSeekNewsItem) string {
	var newsList strings.Builder
	for i, item := range items {
		newsList.WriteString(fmt.Sprintf("%d. Заголовок: %s\n", i+1, item.Title))
		if item.Description != "" {
			newsList.WriteString(fmt.Sprintf("   Описание: %s\n", truncateForLog(item.Description, 200)))
		}
		if item.Content != "" {
			newsList.WriteString(fmt.Sprintf("   Контент: %s\n", truncateForLog(item.Content, 300)))
		}
		if len(item.Categories) > 0 {
			newsList.WriteString(fmt.Sprintf("   RSS категории: %s\n", strings.Join(item.Categories, ", ")))
		}
		newsList.WriteString("\n")
	}

	prompt := fmt.Sprintf(`Классифицируй следующие новости по категориям:

%s

Доступные категории:
1 - Политика (правительство, президент, выборы, парламент, санкции, дипломатия, политические партии)
2 - Экономика (финансы, бизнес, рынки, валюта, банки, инвестиции, ВВП, торговля)
3 - Спорт (футбол, хоккей, баскетбол, теннис, олимпиады, чемпионаты, соревнования)
4 - Технологии (IT, интернет, ИИ, роботы, смартфоны, блокчейн, криптовалюты, стартапы)
5 - Культура (искусство, кино, музыка, театр, литература, выставки, музеи)
6 - Наука (исследования, открытия, медицина, космос, экология, образование)
7 - Общество (социальные вопросы, семья, образование, транспорт, экология, быт)
8 - Происшествия (аварии, катастрофы, преступления, ДТП, пожары, чрезвычайные ситуации)

Требования:
1. Проанализируй заголовок, описание и контент каждой новости
2. Определи наиболее подходящую категорию из 8 доступных
3. Оцени уверенность от 0.0 до 1.0
4. НЕ используй категорию 1 (Политика) как fallback - выбирай наиболее подходящую категорию
5. Для спортивных новостей используй категорию 3 (Спорт)
6. Для экономических новостей используй категорию 2 (Экономика)
7. Для технологических новостей используй категорию 4 (Технологии)
8. Для культурных новостей используй категорию 5 (Культура)
9. Для научных новостей используй категорию 6 (Наука)
10. Для общественных новостей используй категорию 7 (Общество)
11. Для происшествий используй категорию 8 (Происшествия)

Ответ в JSON формате:
{
  "classifications": [
    {"index": 1, "category_id": 2, "confidence": 0.85},
    {"index": 2, "category_id": 3, "confidence": 0.92}
  ]
}`, newsList.String())

	return prompt
}

// sendRequest отправляет запрос к DeepSeek API
func (c *DeepSeekNewsClassifier) sendRequest(ctx context.Context, prompt string) (string, error) {
	// Добавляем задержку между запросами для соблюдения rate limits
	timeSinceLastRequest := time.Since(c.lastRequest)
	if timeSinceLastRequest < 1*time.Second {
		delay := 1*time.Second - timeSinceLastRequest
		c.logger.WithField("delay", delay).Debug("Rate limiting delay")
		time.Sleep(delay)
	}
	c.lastRequest = time.Now()

	request := DeepSeekClassificationRequest{
		Model: c.model,
		Messages: []DeepSeekClassificationMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   1500, // Ограничиваем для экономии
		Temperature: 0.1,  // Низкая температура для точности
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		c.logger.Warn("Rate limited by DeepSeek API, will retry with longer delay")
		// Увеличиваем задержку для следующего запроса
		c.lastRequest = time.Now().Add(-5 * time.Second)
		return "", fmt.Errorf("rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var response DeepSeekClassificationResponse
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

// parseBatchClassificationResponse парсит batch-ответ DeepSeek API
func (c *DeepSeekNewsClassifier) parseBatchClassificationResponse(response string, items []DeepSeekNewsItem) ([]DeepSeekClassificationResult, error) {
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

	var batchResponse DeepSeekBatchClassificationResponse
	if err := json.Unmarshal([]byte(jsonStr), &batchResponse); err != nil {
		c.logger.WithFields(logrus.Fields{
			"response": response,
			"json_str": jsonStr,
			"error":    err,
		}).Error("Failed to parse JSON response")
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Создаем результаты
	results := make([]DeepSeekClassificationResult, len(items))

	// Инициализируем все результаты как ошибки
	for i := range results {
		results[i] = DeepSeekClassificationResult{
			Index:      i,
			CategoryID: models.CategoryPolitics, // Fallback категория
			Confidence: 0.1,
			Error:      fmt.Errorf("no classification found"),
		}
	}

	// Обрабатываем классификации
	for _, classification := range batchResponse.Classifications {
		index := classification.Index - 1 // Индексы в JSON начинаются с 1
		if index < 0 || index >= len(items) {
			c.logger.WithFields(logrus.Fields{
				"index":       classification.Index,
				"items_count": len(items),
			}).Warn("Invalid index in DeepSeek response")
			continue
		}

		// Проверяем валидность category_id
		validCategories := []int{1, 2, 3, 4, 5, 6, 7, 8}
		isValid := false
		for _, validID := range validCategories {
			if classification.CategoryID == validID {
				isValid = true
				break
			}
		}
		if !isValid {
			c.logger.WithFields(logrus.Fields{
				"index":       classification.Index,
				"category_id": classification.CategoryID,
			}).Warn("Invalid category_id in DeepSeek response")
			classification.CategoryID = models.CategoryPolitics // Fallback
		}

		// Проверяем валидность confidence
		if classification.Confidence < 0.0 || classification.Confidence > 1.0 {
			classification.Confidence = 0.5 // Fallback
		}

		results[index] = DeepSeekClassificationResult{
			Index:      index,
			CategoryID: classification.CategoryID,
			Confidence: classification.Confidence,
			Error:      nil,
		}
	}

	return results, nil
}

// GetAvailableCategories возвращает список доступных категорий
func (c *DeepSeekNewsClassifier) GetAvailableCategories() map[int]string {
	return map[int]string{
		models.CategoryPolitics:   "Политика",
		models.CategoryEconomics:  "Экономика",
		models.CategorySports:     "Спорт",
		models.CategoryTechnology: "Технологии",
		models.CategoryCulture:    "Культура",
		models.CategoryScience:    "Наука",
		models.CategorySociety:    "Общество",
		models.CategoryIncidents:  "Происшествия",
	}
}
