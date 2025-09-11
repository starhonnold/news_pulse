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
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Println("Ошибка: не установлен DEEPSEEK_API_KEY")
		fmt.Println("Установите переменную окружения: export DEEPSEEK_API_KEY=your_api_key")
		os.Exit(1)
	}

	testNews := []services.DeepSeekNewsItem{
		{
			Index:       0,
			Title:       "Путин подписал указ о новых санкциях",
			Description: "Президент России подписал указ о введении новых экономических санкций против западных стран",
			Content:     "Президент России Владимир Путин подписал указ о введении новых экономических санкций против западных стран. Документ предусматривает ограничения на импорт товаров и услуг из стран, которые ввели санкции против России.",
			Categories:  []string{"политика", "россия"},
		},
		{
			Index:       1,
			Title:       "Новый iPhone 15 получил улучшенную камеру",
			Description: "Apple представила новый iPhone 15 с улучшенной системой камер и новым процессором A17 Pro",
			Content:     "Apple представила новый iPhone 15 с улучшенной системой камер и новым процессором A17 Pro. Смартфон получил титановый корпус и поддержку USB-C.",
			Categories:  []string{"технологии", "apple"},
		},
		{
			Index:       2,
			Title:       "Российские спортсмены завоевали золото на чемпионате мира",
			Description: "Российские спортсмены завоевали золотые медали на чемпионате мира по легкой атлетике",
			Content:     "Российские спортсмены показали отличные результаты на чемпионате мира по легкой атлетике, завоевав несколько золотых медалей.",
			Categories:  []string{"спорт", "россия"},
		},
		{
			Index:       3,
			Title:       "Курс доллара вырос до 95 рублей",
			Description: "Курс доллара к рублю вырос до 95 рублей на Московской бирже",
			Content:     "Курс доллара к рублю вырос до 95 рублей на Московской бирже. Аналитики связывают рост с геополитической напряженностью.",
			Categories:  []string{"экономика", "валюта"},
		},
		{
			Index:       4,
			Title:       "В Москве открылась новая выставка современного искусства",
			Description: "В Третьяковской галерее открылась выставка современного российского искусства",
			Content:     "В Третьяковской галерее открылась масштабная выставка современного российского искусства, на которой представлены работы ведущих художников страны.",
			Categories:  []string{"культура", "искусство"},
		},
	}

	fmt.Println("🚀 Тестирование классификации новостей через DeepSeek API")
	fmt.Println("📊 Модель: deepseek-chat")
	fmt.Println("💰 Стоимость: $0.07/1M входных токенов, $1.10/1M выходных токенов")
	fmt.Println(strings.Repeat("=", 60))

	// Создаем logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Создаем DeepSeek классификатор новостей
	classifier := services.NewDeepSeekNewsClassifier(apiKey, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	startTime := time.Now()
	results, err := classifier.ClassifyNewsBatch(ctx, testNews)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Ошибка классификации: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Классификация завершена за %s\n", duration)
	fmt.Println(strings.Repeat("-", 60))

	// Выводим результаты
	categories := classifier.GetAvailableCategories()

	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ Новость %d: Ошибка - %v\n", i+1, result.Error)
		} else {
			categoryName := "Неизвестная"
			if name, exists := categories[result.CategoryID]; exists {
				categoryName = name
			}

			fmt.Printf("📰 Новость %d: %s\n", i+1, truncateForDisplay(testNews[i].Title, 50))
			fmt.Printf("   📂 Категория: %s (ID: %d)\n", categoryName, result.CategoryID)
			fmt.Printf("   🎯 Уверенность: %.2f\n", result.Confidence)
			fmt.Printf("   📝 Описание: %s\n", truncateForDisplay(testNews[i].Description, 80))
		}
		fmt.Println()
	}

	// Примерная стоимость
	inputTokens := len(buildPromptForEstimation(testNews)) / 4
	outputTokens := len(fmt.Sprintf("%v", results)) / 4
	estimatedCost := float64(inputTokens)/1000000*0.07 + float64(outputTokens)/1000000*1.10

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("💰 Использовано токенов: %d входных, %d выходных, %d всего\n", inputTokens, outputTokens, inputTokens+outputTokens)
	fmt.Printf("💵 Примерная стоимость: $%.4f\n", estimatedCost)
	fmt.Printf("📊 Средняя стоимость за новость: $%.4f\n", estimatedCost/float64(len(testNews)))

	fmt.Println("\n🏁 Тестирование завершено")
	fmt.Println("\n💡 Примечание: DeepSeek API с моделью deepseek-chat")
	fmt.Println("   - Высокое качество классификации новостей")
	fmt.Println("   - Поддержка 12 категорий")
	fmt.Println("   - Оценка уверенности для каждой классификации")
}

// truncateForDisplay обрезает строку для вывода
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// buildPromptForEstimation создает фиктивный промпт для оценки токенов
func buildPromptForEstimation(news []services.DeepSeekNewsItem) string {
	var newsList strings.Builder
	for i, item := range news {
		newsList.WriteString(fmt.Sprintf("%d. Заголовок: %s\n", i+1, item.Title))
		if item.Description != "" {
			newsList.WriteString(fmt.Sprintf("   Описание: %s\n", item.Description))
		}
		if item.Content != "" {
			newsList.WriteString(fmt.Sprintf("   Контент: %s\n", item.Content))
		}
		if len(item.Categories) > 0 {
			newsList.WriteString(fmt.Sprintf("   RSS категории: %s\n", strings.Join(item.Categories, ", ")))
		}
		newsList.WriteString("\n")
	}

	return fmt.Sprintf(`Классифицируй следующие новости по категориям:

%s

Доступные категории:
в1 - Политика (правительство, президент, выборы, парламент, санкции, дипломатия, политические партии)
2 - Экономика (финансы, бизнес, рынки, валюта, банки, инвестиции, ВВП, торговля)
3 - Спорт (футбол, хоккей, баскетбол, теннис, олимпиады, чемпионаты, соревнования)
4 - Технологии (IT, интернет, ИИ, роботы, смартфоны, блокчейн, криптовалюты, стартапы)
5 - Культура (искусство, кино, музыка, театр, литература, выставки, музеи)
6 - Наука (исследования, открытия, медицина, космос, экология, образование)
7 - Общество (социальные вопросы, семья, образование, транспорт, экология, быт)
8 - Происшествия (аварии, катастрофы, преступления, ДТП, пожары, чрезвычайные ситуации)

Требования:
1. Проанализируй заголовок, описание и контент каждой новости
2. Определи наиболее подходящую категорию
3. Оцени уверенность от 0.0 до 1.0
4. Если новость не подходит ни к одной категории, используй категорию 1 (Россия) с низкой уверенностью

Ответ в JSON формате:
{
  "classifications": [
    {"index": 1, "category_id": 2, "confidence": 0.85},
    {"index": 2, "category_id": 3, "confidence": 0.92}
  ]
}`, newsList.String())
}
