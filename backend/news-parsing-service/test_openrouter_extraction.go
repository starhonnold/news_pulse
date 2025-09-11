package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"news-parsing-service/internal/services"

	"github.com/sirupsen/logrus"
)

func main() {
	// Получаем API ключ из переменной окружения
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("Ошибка: не установлен OPENROUTER_API_KEY")
		fmt.Println("Установите переменную окружения: export OPENROUTER_API_KEY=your_api_key")
		fmt.Println("Получить ключ можно на: https://openrouter.ai/keys")
		return
	}

	// Тестовые URL новостей
	testURLs := []string{
		"https://ria.ru/20250911/buryatiya-2041228792.html",
		"https://lenta.ru/news/2025/09/11/vyyavlen-neozhidannyy-priznak-priblizheniya-dementsii/",
	}

	fmt.Println("🚀 Тестирование извлечения контента через OpenRouter API")
	fmt.Println("📊 Модель: openai/gpt-oss-20b")
	fmt.Println("💰 Стоимость: $0.04/1M входных токенов, $0.15/1M выходных токенов")
	fmt.Println(strings.Repeat("=", 60))

	// Создаем logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Создаем OpenRouter извлекатель контента
	extractor := services.NewOpenRouterContentExtractor(apiKey, logger)

	// Выводим информацию о модели
	fmt.Printf("ℹ️  %s\n", extractor.GetModelInfo())

	for i, url := range testURLs {
		fmt.Printf("\n📰 Тест %d: %s\n", i+1, url)
		fmt.Println(strings.Repeat("-", 50))

		// Проверяем валидность URL
		if !extractor.IsValidURL(url) {
			fmt.Printf("❌ URL не валиден для извлечения контента\n")
			continue
		}

		// Извлекаем контент через OpenRouter API
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		title, content, err := extractor.ExtractSingleContent(ctx, url)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			continue
		}

		if content == "" {
			fmt.Printf("⚠️  Контент не извлечен (пустая строка)\n")
			continue
		}

		fmt.Printf("✅ Заголовок: %s\n", title)
		fmt.Printf("📄 Длина контента: %d символов\n", len(content))
		fmt.Printf("📝 Начало контента: %s...\n", truncateString(content, 200))
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏁 Тестирование завершено")
	fmt.Println("\n💡 Примечание: OpenRouter API с моделью gpt-oss-20b")
	fmt.Println("   - Очень экономичная модель (в 3-4 раза дешевле GPT-4o-mini)")
	fmt.Println("   - Хорошее качество извлечения контента")
	fmt.Println("   - Поддержка batch-обработки для экономии")
}

// truncateString обрезает строку до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
