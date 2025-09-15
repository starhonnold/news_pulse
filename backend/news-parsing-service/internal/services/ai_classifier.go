package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OllamaClient представляет клиент для работы с Ollama API
type OllamaClient struct {
	baseURL     string
	model       string
	temperature float64
	httpClient  *http.Client
	logger      *logrus.Logger
}

// OllamaRequest представляет запрос к Ollama API
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse представляет ответ от Ollama API
type OllamaResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

// AIClassifier представляет AI классификатор новостей
type AIClassifier struct {
	client *OllamaClient
	logger *logrus.Logger
}

// NewOllamaClient создает новый клиент Ollama
func NewOllamaClient(baseURL string, timeout time.Duration, temperature float64, logger *logrus.Logger) *OllamaClient {
	return &OllamaClient{
		baseURL:     baseURL,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// NewAIClassifier создает новый AI классификатор
func NewAIClassifier(ollamaURL string, model string, timeout time.Duration, temperature float64, logger *logrus.Logger) *AIClassifier {
	client := NewOllamaClient(ollamaURL, timeout, temperature, logger)
	client.model = model
	return &AIClassifier{
		client: client,
		logger: logger,
	}
}

// ProcessNewsBatch обрабатывает пакет новостей для классификации параллельно
func (c *AIClassifier) ProcessNewsBatch(ctx context.Context, items []UnifiedNewsItem) ([]UnifiedProcessingResult, error) {
	if len(items) == 0 {
		return []UnifiedProcessingResult{}, nil
	}

	// Ограничиваем количество параллельных запросов для CPU модели
	const maxConcurrent = 3
	semaphore := make(chan struct{}, maxConcurrent)

	results := make([]UnifiedProcessingResult, len(items))
	errors := make([]error, len(items))

	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(index int, newsItem UnifiedNewsItem) {
			defer wg.Done()

			// Получаем семафор
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				errors[index] = ctx.Err()
				return
			}
			defer func() { <-semaphore }()

			// Классифицируем новость
			result := c.classifyNewsItem(ctx, newsItem, index)
			results[index] = result
		}(i, item)
	}

	wg.Wait()

	// Проверяем на ошибки контекста
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// processIndividualItems обрабатывает новости индивидуально (fallback)
func (c *AIClassifier) processIndividualItems(ctx context.Context, items []UnifiedNewsItem) ([]UnifiedProcessingResult, error) {
	results := make([]UnifiedProcessingResult, len(items))

	for i, item := range items {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := c.classifyNewsItem(ctx, item, i)
		results[i] = result
	}

	return results, nil
}

// classifyNewsItem классифицирует одну новость
func (c *AIClassifier) classifyNewsItem(ctx context.Context, item UnifiedNewsItem, index int) UnifiedProcessingResult {
	// Создаем промпт для классификации
	prompt := c.createClassificationPrompt(item)

	// Отправляем запрос к Ollama
	response, err := c.client.Generate(ctx, prompt)
	if err != nil {
		c.logger.WithError(err).WithField("index", index).Error("Failed to classify news item")
		return UnifiedProcessingResult{
			Index:      index,
			Title:      item.Title,
			Content:    item.Content,
			CategoryID: 1, // Fallback to Politics
			Confidence: 0.1,
			Error:      err,
		}
	}

	// Парсим ответ и определяем категорию
	categoryID, confidence := c.parseClassificationResponse(response)

	// Проверяем, удалось ли определить категорию
	if categoryID == 0 {
		err := fmt.Errorf("failed to determine category from AI response: %s", response)
		c.logger.WithError(err).WithField("index", index).Warn("AI classification failed")
		return UnifiedProcessingResult{
			Index:      index,
			Title:      item.Title,
			Content:    item.Content,
			CategoryID: 0,
			Confidence: 0.0,
			Error:      err,
		}
	}

	c.logger.WithFields(logrus.Fields{
		"index":       index,
		"title":       truncateForLog(item.Title, 50),
		"category_id": categoryID,
		"confidence":  confidence,
		"response":    truncateForLog(response, 100),
	}).Info("News classified with AI")

	return UnifiedProcessingResult{
		Index:      index,
		Title:      item.Title,
		Content:    item.Content,
		CategoryID: categoryID,
		Confidence: confidence,
		Error:      nil,
	}
}

// createClassificationPrompt создает промпт для классификации новости
func (c *AIClassifier) createClassificationPrompt(item UnifiedNewsItem) string {
	return fmt.Sprintf(`You are a strict news classifier.
Your task is to assign the news into ONE category from the list.

Categories:
1. Politics - president, government, elections, parliament, minister, deputy, law, sanctions, negotiations, diplomacy, war, security, reforms, kremlin, nato, eu
2. Economy - economy, inflation, currency, exchange rate, ruble, dollar, euro, budget, bank, credit, investments, market, tax, rate
3. Sports - sports, match, team, player, coach, championship, tournament, goal, score, league, club, футбол, матч, команда, игрок, тренер, чемпионат, турнир, гол, счет, лига, клуб
4. Technology - technology, computer, internet, smartphone, application, artificial intelligence, robot, software, data, network, технология, компьютер, интернет, смартфон, приложение, искусственный интеллект, робот, софт, данные, сеть
5. Culture - culture, art, museum, theater, cinema, film, actor, director, music, concert, exhibition, book, культура, искусство, музей, театр, кино, фильм, актер, режиссер, музыка, концерт, выставка, книга
6. Science - science, research, scientist, experiment, laboratory, journal, university, discovery, development, наука, исследование, ученый, эксперимент, лаборатория, журнал, университет, открытие, разработка
7. Society - society, social, citizen, court, police, road, family, children, общество, социальный, гражданин, суд, полиция, дорога, семья, дети
8. Incidents - incident, accident, fire, explosion, disaster, terrorist attack, crime, murder, theft, earthquake, происшествие, ДТП, авария, пожар, взрыв, катастрофа, теракт, криминал, убийство, кража, землетрясение
9. Health - health, medical, doctor, hospital, treatment, disease, symptom, diagnosis, vaccine, epidemic, здоровье, медицинский, врач, больница, лечение, заболевание, симптом, диагноз, вакцина, эпидемия
10. Education - school, university, institute, exam, student, teacher, course, lecture, bachelor, master, школа, университет, институт, экзамен, студент, учитель, курс, лекция, бакалавр, магистр
11. International - international, summit, diplomacy, ambassador, visa, border, nato, eu, un, brussels, международный, саммит, дипломатия, посол, виза, граница, нато, ес, оон, брюссель
12. Business - business, company, corporation, director, manager, employee, office, recruitment, dismissal, profit, бизнес, компания, корпорация, директор, менеджер, сотрудник, офис, рекрутинг, увольнение, прибыль

Examples:
News: "Президент подписал новый закон"
Answer: 1

News: "Клуб выиграл матч чемпионата России по футболу"
Answer: 3

News: "Компания представила новый смартфон с искусственным интеллектом"
Answer: 4

News: "Ученые провели исследование в университете"
Answer: 6

News: "В больнице прошла операция по пересадке сердца"
Answer: 9

Now classify this news:

Title: %s
Description: %s
Content: %s

Answer ONLY with one number (1–12). No explanations, no text, no punctuation.
`,
		item.Title,
		item.Description,
		truncateForLog(item.Content, 1000))
}

// parseClassificationResponse парсит ответ от AI и определяет категорию
func (c *AIClassifier) parseClassificationResponse(response string) (int, float64) {
	// Ищем число в ответе
	var categoryID int
	var confidence float64 = 0.5 // Базовая уверенность

	// Простой парсинг - ищем первое число от 1 до 12
	for _, char := range response {
		if char >= '1' && char <= '9' {
			categoryID = int(char - '0')
			confidence = 0.8
			break
		} else if char == '1' {
			// Проверяем следующую цифру для 10, 11, 12
			if len(response) > 1 {
				nextChar := response[1]
				if nextChar == '0' {
					categoryID = 10
					confidence = 0.8
					break
				} else if nextChar == '1' {
					categoryID = 11
					confidence = 0.8
					break
				} else if nextChar == '2' {
					categoryID = 12
					confidence = 0.8
					break
				}
			}
		}
	}

	// Если не нашли валидную категорию, возвращаем ошибку
	if categoryID < 1 || categoryID > 12 {
		return 0, 0.0 // Возвращаем 0 как индикатор неудачи
	}

	return categoryID, confidence
}

// Generate отправляет запрос к Ollama API
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	request := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": c.temperature,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API error: %d - %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Response, nil
}

// HealthCheck проверяет доступность Ollama сервиса
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check failed: %d", resp.StatusCode)
	}

	return nil
}

// Close закрывает клиент
func (c *AIClassifier) Close() error {
	return nil
}
