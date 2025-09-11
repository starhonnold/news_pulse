package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DeepSeekRequest структура запроса к DeepSeek API
type DeepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
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

func main() {
	// Получаем API ключ из переменной окружения
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Println("Ошибка: не установлен DEEPSEEK_API_KEY")
		fmt.Println("Установите переменную окружения: export DEEPSEEK_API_KEY=your_api_key")
		fmt.Println("Получить ключ можно на: https://platform.deepseek.com/api_keys")
		return
	}

	// Тестовые URL новостей
	testURLs := []string{
		"https://ria.ru/20250911/buryatiya-2041228792.html",
		"https://lenta.ru/news/2025/09/11/vyyavlen-neozhidannyy-priznak-priblizheniya-dementsii/",
	}

	fmt.Println("🚀 Тестирование извлечения контента через DeepSeek API")
	fmt.Println("📊 Модель: deepseek-chat")
	fmt.Println("💰 Стоимость: $0.07/1M входных токенов, $1.10/1M выходных токенов")
	fmt.Println(strings.Repeat("=", 60))

	for i, url := range testURLs {
		fmt.Printf("\n📰 Тест %d: %s\n", i+1, url)
		fmt.Println(strings.Repeat("-", 50))

		// Извлекаем контент через DeepSeek API
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		content, usage, err := extractContentWithDeepSeek(ctx, apiKey, url)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			continue
		}

		if content == "" {
			fmt.Printf("⚠️  Контент не извлечен (пустая строка)\n")
			continue
		}

		fmt.Printf("✅ Контент извлечен успешно\n")
		fmt.Printf("📄 Длина контента: %d символов\n", len(content))
		fmt.Printf("📝 Начало контента: %s...\n", truncateString(content, 200))

		if usage != nil {
			cost := float64(usage.PromptTokens)*0.07/1000000 + float64(usage.CompletionTokens)*1.10/1000000
			fmt.Printf("💰 Использовано токенов: %d входных, %d выходных, %d всего\n",
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
			fmt.Printf("💵 Примерная стоимость: $%.4f\n", cost)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏁 Тестирование завершено")
	fmt.Println("\n💡 Примечание: DeepSeek API с моделью deepseek-chat")
	fmt.Println("   - Высокое качество извлечения контента")
	fmt.Println("   - Детальная информация об использовании токенов")
	fmt.Println("   - Скидки в непиковые часы (16:30-00:30 GMT)")
}

// extractContentWithDeepSeek извлекает контент с помощью DeepSeek API
func extractContentWithDeepSeek(ctx context.Context, apiKey, url string) (string, *DeepSeekUsage, error) {
	prompt := fmt.Sprintf(`Извлеки полный текст новости по ссылке: %s

Требования:
1. Перейди по ссылке и найди основную новость
2. Извлеки ТОЛЬКО основной текст новости (без рекламы, навигации, комментариев)
3. Удали все HTML-теги и оставь только чистый текст
4. Сохрани структуру текста (абзацы, заголовки)
5. Ограничь текст до 2000 символов
6. Если новость недоступна, верни "Новость недоступна"

Ответь только текстом новости без дополнительных комментариев.`, url)

	request := DeepSeekRequest{
		Model: "deepseek-chat",
		Messages: []DeepSeekMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1,
		Stream:      false,
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
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

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

// truncateString обрезает строку до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
