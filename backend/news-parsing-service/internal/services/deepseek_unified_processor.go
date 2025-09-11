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
	"news-parsing-service/internal/utils"
)

// DeepSeekUnifiedProcessor объединенный процессор для классификации и извлечения контента
type DeepSeekUnifiedProcessor struct {
	logger      *logrus.Logger
	httpClient  *http.Client
	apiKey      string
	lastRequest time.Time
	model       string
}

// UnifiedNewsItem элемент для объединенной обработки новостей
type UnifiedNewsItem struct {
	Index       int
	Title       string
	Description string
	Content     string
	URL         string
	Categories  []string
}

// UnifiedProcessingResult результат объединенной обработки
type UnifiedProcessingResult struct {
	Index      int
	Title      string
	Content    string
	CategoryID int
	Confidence float64
	Error      error
}

// DeepSeekUnifiedRequest структура запроса к DeepSeek API
type DeepSeekUnifiedRequest struct {
	Model       string                   `json:"model"`
	Messages    []DeepSeekUnifiedMessage `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
}

// DeepSeekUnifiedMessage структура сообщения для DeepSeek API
type DeepSeekUnifiedMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekUnifiedResponse структура ответа от DeepSeek API
type DeepSeekUnifiedResponse struct {
	Choices []DeepSeekUnifiedChoice `json:"choices"`
	Error   *DeepSeekUnifiedError   `json:"error,omitempty"`
	Usage   *DeepSeekUnifiedUsage   `json:"usage,omitempty"`
}

// DeepSeekUnifiedChoice структура выбора из ответа
type DeepSeekUnifiedChoice struct {
	Message DeepSeekUnifiedMessage `json:"message"`
}

// DeepSeekUnifiedError структура ошибки
type DeepSeekUnifiedError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// DeepSeekUnifiedUsage структура использования токенов
type DeepSeekUnifiedUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeepSeekUnifiedBatchResponse структура для парсинга batch-ответа
type DeepSeekUnifiedBatchResponse struct {
	Processings []DeepSeekUnifiedProcessingItem `json:"processings"`
}

// DeepSeekUnifiedProcessingItem элемент обработки
type DeepSeekUnifiedProcessingItem struct {
	Index      int     `json:"index"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	CategoryID int     `json:"category_id"`
	Confidence float64 `json:"confidence"`
}

// NewDeepSeekUnifiedProcessor создает новый объединенный процессор
func NewDeepSeekUnifiedProcessor(apiKey string, logger *logrus.Logger) *DeepSeekUnifiedProcessor {
	return &DeepSeekUnifiedProcessor{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Увеличиваем таймаут для сложных запросов
		},
		apiKey: apiKey,
		model:  "deepseek-chat",
	}
}

// ProcessNewsBatch обрабатывает новости (классификация + извлечение контента) одним запросом
func (p *DeepSeekUnifiedProcessor) ProcessNewsBatch(ctx context.Context, items []UnifiedNewsItem) ([]UnifiedProcessingResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to process")
	}

	// Ограничиваем количество элементов для batch-обработки
	batchSize := 5 // Уменьшаем размер batch для стабильности
	if len(items) > batchSize {
		items = items[:batchSize]
	}

	// Формируем batch-промпт
	prompt := p.buildUnifiedPrompt(items)

	// Отправляем запрос к DeepSeek API
	response, usage, err := p.sendRequest(ctx, prompt)
	if err != nil {
		p.logger.WithError(err).Error("Failed to send unified processing request to DeepSeek API")
		return nil, err
	}

	// Парсим batch-ответ
	results, err := p.parseUnifiedResponse(response, items, usage)
	if err != nil {
		p.logger.WithError(err).Error("Failed to parse unified DeepSeek response")
		return nil, err
	}

	p.logger.WithFields(logrus.Fields{
		"items_count":   len(items),
		"results_count": len(results),
		"model":         p.model,
		"usage":         usage,
	}).Info("DeepSeek unified processed news")

	return results, nil
}

// buildUnifiedPrompt создает объединенный промпт для DeepSeek API
func (p *DeepSeekUnifiedProcessor) buildUnifiedPrompt(items []UnifiedNewsItem) string {
	var newsList strings.Builder
	for i, item := range items {
		newsList.WriteString(fmt.Sprintf("%d. Заголовок: %s\n", i+1, item.Title))
		if item.Description != "" {
			newsList.WriteString(fmt.Sprintf("   Описание: %s\n", truncateForLog(item.Description, 200)))
		}
		if item.URL != "" {
			newsList.WriteString(fmt.Sprintf("   URL: %s\n", item.URL))
		}
		if len(item.Categories) > 0 {
			newsList.WriteString(fmt.Sprintf("   RSS категории: %s\n", strings.Join(item.Categories, ", ")))
		}
		newsList.WriteString("\n")
	}

	prompt := fmt.Sprintf(`Ты - эксперт по обработке новостей. Мне нужно для каждой новости:
1. Извлечь полный текст новости по ссылке (если URL доступен)
2. Классифицировать новость по категориям

Новости для обработки:
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

ТРЕБОВАНИЯ для каждой новости:
1. Перейди по ссылке и извлеки полный текст новости (без рекламы, навигации, комментариев)
2. Сохрани структуру текста (абзацы, заголовки)
3. Удали все HTML-теги и оставь только чистый текст
4. Ограничь текст до 2000 символов
5. ВАЖНО: Используй только печатаемые символы UTF-8, без управляющих символов
6. Проанализируй заголовок, описание и извлеченный контент
7. Определи наиболее подходящую категорию из 8 доступных
8. Оцени уверенность от 0.0 до 1.0
9. НЕ используй категорию 1 (Политика) как fallback - выбирай наиболее подходящую категорию

Если новость недоступна или не найдена, верни:
- title: "Новость недоступна"
- content: "Новость недоступна"
- category_id: 1 (Политика как fallback для недоступных новостей)
- confidence: 0.1

Ответ в JSON формате:
{
  "processings": [
    {
      "index": 1,
      "title": "Заголовок новости",
      "content": "Полный текст новости без HTML-тегов и специальных символов",
      "category_id": 2,
      "confidence": 0.85
    },
    {
      "index": 2,
      "title": "Заголовок новости",
      "content": "Полный текст новости без HTML-тегов и специальных символов",
      "category_id": 3,
      "confidence": 0.92
    }
  ]
}`, newsList.String())

	return prompt
}

// sendRequest отправляет запрос к DeepSeek API
func (p *DeepSeekUnifiedProcessor) sendRequest(ctx context.Context, prompt string) (string, *DeepSeekUnifiedUsage, error) {
	// Добавляем задержку между запросами для соблюдения rate limits
	timeSinceLastRequest := time.Since(p.lastRequest)
	if timeSinceLastRequest < 2*time.Second {
		delay := 2*time.Second - timeSinceLastRequest
		p.logger.WithField("delay", delay).Debug("Rate limiting delay")
		time.Sleep(delay)
	}
	p.lastRequest = time.Now()

	request := DeepSeekUnifiedRequest{
		Model: p.model,
		Messages: []DeepSeekUnifiedMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   4000, // Увеличиваем лимит для сложных запросов
		Temperature: 0.1,  // Низкая температура для точности
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		p.logger.Warn("Rate limited by DeepSeek API, will retry with longer delay")
		// Увеличиваем задержку для следующего запроса
		p.lastRequest = time.Now().Add(-10 * time.Second)
		return "", nil, fmt.Errorf("rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("API request failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	var response DeepSeekUnifiedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", nil, fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, response.Usage, nil
}

// parseUnifiedResponse парсит объединенный ответ DeepSeek API
func (p *DeepSeekUnifiedProcessor) parseUnifiedResponse(response string, items []UnifiedNewsItem, usage *DeepSeekUnifiedUsage) ([]UnifiedProcessingResult, error) {
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

	var batchResponse DeepSeekUnifiedBatchResponse
	if err := json.Unmarshal([]byte(jsonStr), &batchResponse); err != nil {
		p.logger.WithFields(logrus.Fields{
			"response": response,
			"json_str": jsonStr,
			"error":    err,
		}).Error("Failed to parse JSON response")
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Создаем результаты
	results := make([]UnifiedProcessingResult, len(items))

	// Инициализируем все результаты как ошибки
	for i := range results {
		results[i] = UnifiedProcessingResult{
			Index:      i,
			Title:      "Не удалось обработать",
			Content:    "",
			CategoryID: models.CategoryPolitics, // Fallback категория
			Confidence: 0.1,
			Error:      fmt.Errorf("no processing found"),
		}
	}

	// Обрабатываем результаты
	for _, processing := range batchResponse.Processings {
		index := processing.Index - 1 // Индексы в JSON начинаются с 1
		if index < 0 || index >= len(items) {
			p.logger.WithFields(logrus.Fields{
				"index":       processing.Index,
				"items_count": len(items),
			}).Warn("Invalid index in DeepSeek response")
			continue
		}

		// Проверяем валидность category_id
		validCategories := []int{1, 2, 3, 4, 5, 6, 7, 8}
		isValid := false
		for _, validID := range validCategories {
			if processing.CategoryID == validID {
				isValid = true
				break
			}
		}
		if !isValid {
			p.logger.WithFields(logrus.Fields{
				"index":       processing.Index,
				"category_id": processing.CategoryID,
			}).Warn("Invalid category_id in DeepSeek response")
			processing.CategoryID = models.CategoryPolitics // Fallback
		}

		// Проверяем валидность confidence
		if processing.Confidence < 0.0 || processing.Confidence > 1.0 {
			processing.Confidence = 0.5 // Fallback
		}

		// Дополнительная очистка контента от непечатаемых символов
		cleanedTitle := utils.CleanText(processing.Title)
		cleanedContent := utils.CleanText(processing.Content)

		results[index] = UnifiedProcessingResult{
			Index:      index,
			Title:      cleanedTitle,
			Content:    cleanedContent,
			CategoryID: processing.CategoryID,
			Confidence: processing.Confidence,
			Error:      nil,
		}
	}

	return results, nil
}
