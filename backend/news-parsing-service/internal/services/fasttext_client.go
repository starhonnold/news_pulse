package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// FastTextClassifierClient - клиент для FastText классификатора
type FastTextClassifierClient struct {
	baseURL string
	client  *http.Client
	logger  *logrus.Logger
	enabled bool
}

// FastTextClassifyRequest - запрос на классификацию
type FastTextClassifyRequest struct {
	Text string `json:"text"`
}

// FastTextClassifyResponse - ответ классификации
type FastTextClassifyResponse struct {
	OriginalCategory string  `json:"original_category"`
	OriginalScore    float64 `json:"original_score"`
	CategoryID       int     `json:"category_id"`
	CategoryName     string  `json:"category_name"`
	Confidence       float64 `json:"confidence"`
}

// FastTextBatchItem - элемент для пакетной классификации
type FastTextBatchItem struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// FastTextBatchRequest - запрос пакетной классификации
type FastTextBatchRequest struct {
	Items []FastTextBatchItem `json:"items"`
}

// FastTextBatchResultItem - результат классификации одного элемента
type FastTextBatchResultItem struct {
	Index            int     `json:"index"`
	OriginalCategory string  `json:"original_category"`
	OriginalScore    float64 `json:"original_score"`
	CategoryID       int     `json:"category_id"`
	CategoryName     string  `json:"category_name"`
	Confidence       float64 `json:"confidence"`
}

// FastTextBatchResponse - ответ пакетной классификации
type FastTextBatchResponse struct {
	Results []FastTextBatchResultItem `json:"results"`
}

// FastTextHealthResponse - ответ health check
type FastTextHealthResponse struct {
	Status      string                 `json:"status"`
	ModelLoaded bool                   `json:"model_loaded"`
	ModelInfo   map[string]interface{} `json:"model_info"`
	Uptime      float64                `json:"uptime"`
}

// NewFastTextClassifierClient создает новый клиент
func NewFastTextClassifierClient(baseURL string, timeout time.Duration, logger *logrus.Logger, enabled bool) *FastTextClassifierClient {
	if logger == nil {
		logger = logrus.New()
	}

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &FastTextClassifierClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		logger:  logger,
		enabled: enabled,
	}
}

// IsEnabled возвращает статус включенности классификатора
func (c *FastTextClassifierClient) IsEnabled() bool {
	return c.enabled
}

// SetEnabled устанавливает статус включенности
func (c *FastTextClassifierClient) SetEnabled(enabled bool) {
	c.enabled = enabled
	c.logger.WithField("enabled", enabled).Info("FastText classifier status changed")
}

// HealthCheck проверяет доступность сервиса
func (c *FastTextClassifierClient) HealthCheck(ctx context.Context) error {
	if !c.enabled {
		return fmt.Errorf("fasttext classifier is disabled")
	}

	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var healthResp FastTextHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	if !healthResp.ModelLoaded {
		return fmt.Errorf("model not loaded")
	}

	return nil
}

// Classify классифицирует одну новость
func (c *FastTextClassifierClient) Classify(ctx context.Context, title, description string) (*FastTextClassifyResponse, error) {
	if !c.enabled {
		return nil, fmt.Errorf("fasttext classifier is disabled")
	}

	// Объединяем заголовок и описание
	text := fmt.Sprintf("%s. %s", title, description)

	request := FastTextClassifyRequest{
		Text: text,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/classify", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("classification failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result FastTextClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ClassifyBatch классифицирует пакет новостей
func (c *FastTextClassifierClient) ClassifyBatch(ctx context.Context, items []UnifiedNewsItem) ([]UnifiedProcessingResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("fasttext classifier is disabled")
	}

	if len(items) == 0 {
		return []UnifiedProcessingResult{}, nil
	}

	// Конвертируем в формат FastText API
	batchItems := make([]FastTextBatchItem, len(items))
	for i, item := range items {
		batchItems[i] = FastTextBatchItem{
			Index:       item.Index,
			Title:       item.Title,
			Description: item.Description,
		}
	}

	request := FastTextBatchRequest{
		Items: batchItems,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	url := fmt.Sprintf("%s/classify/batch", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("batch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch classification failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var batchResp FastTextBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	// Конвертируем результаты в UnifiedProcessingResult
	results := make([]UnifiedProcessingResult, len(batchResp.Results))
	for i, item := range batchResp.Results {
		results[i] = UnifiedProcessingResult{
			Index:      item.Index,
			Title:      items[i].Title,
			Content:    items[i].Content,
			CategoryID: item.CategoryID,
			Confidence: item.Confidence,
			Error:      nil,
		}
	}

	return results, nil
}

// GetStats возвращает статистику клиента
func (c *FastTextClassifierClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"base_url": c.baseURL,
		"enabled":  c.enabled,
		"timeout":  c.client.Timeout.String(),
	}
}
