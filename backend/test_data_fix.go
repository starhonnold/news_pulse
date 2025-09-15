package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"news-parsing-service/internal/config"
	"news-parsing-service/internal/database"
	"news-parsing-service/internal/repository"
	"news-parsing-service/internal/services"

	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("Тестирование исправления проблемы с данными новостей...")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("news-parsing-service/config/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключаемся к базе данных
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	// Создаем репозитории
	newsSourceRepo := repository.NewNewsSourceRepository(db)
	newsRepo := repository.NewNewsRepository(db)
	parsingLogRepo := repository.NewParsingLogRepository(db)

	// Создаем логгер
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Создаем сервисы
	rssParser := services.NewRSSParser(cfg.Parsing, cfg.Proxy, logger)
	parsingService := services.NewParsingService(
		rssParser,
		newsSourceRepo,
		newsRepo,
		parsingLogRepo,
		cfg.Parsing,
		cfg,
		logger,
	)

	if parsingService == nil {
		log.Fatal("Не удалось создать сервис парсинга")
	}

	// Получаем тестовый источник
	ctx := context.Background()
	sources, err := newsSourceRepo.GetSourcesToParse(ctx)
	if err != nil {
		log.Fatalf("Ошибка получения источников: %v", err)
	}

	if len(sources) == 0 {
		log.Fatal("Нет активных источников для тестирования")
	}

	// Тестируем парсинг первого источника
	source := sources[0]
	fmt.Printf("Тестируем парсинг источника: %s (ID: %d)\n", source.Name, source.ID)

	// Парсим источник
	startTime := time.Now()
	result := rssParser.ParseFeed(ctx, source)
	parseTime := time.Since(startTime)

	fmt.Printf("Парсинг завершен за: %v\n", parseTime)
	fmt.Printf("Успешно: %v\n", result.Success)
	fmt.Printf("Количество элементов: %d\n", len(result.Items))

	if !result.Success {
		fmt.Printf("Ошибка парсинга: %s\n", result.Error)
		return
	}

	if len(result.Items) == 0 {
		fmt.Println("Нет элементов для обработки")
		return
	}

	// Обрабатываем элементы
	fmt.Println("\nОбрабатываем элементы...")
	newsList, err := parsingService.ProcessItems(ctx, result.Items, source)
	if err != nil {
		log.Fatalf("Ошибка обработки элементов: %v", err)
	}

	fmt.Printf("Обработано новостей: %d\n", len(newsList))

	// Проверяем соответствие данных
	fmt.Println("\nПроверяем соответствие данных:")
	for i, news := range newsList {
		if i >= 3 { // Показываем только первые 3
			break
		}

		fmt.Printf("\n--- Новость %d ---\n", i+1)
		fmt.Printf("Заголовок: %s\n", truncateString(news.Title, 100))
		fmt.Printf("Описание: %s\n", truncateString(news.Description, 150))
		fmt.Printf("Контент: %s\n", truncateString(news.Content, 200))
		fmt.Printf("URL: %s\n", news.URL)
		fmt.Printf("Категория ID: %v\n", news.CategoryID)
		fmt.Printf("Источник ID: %d\n", news.SourceID)
		fmt.Printf("Опубликовано: %s\n", news.PublishedAt.Format("2006-01-02 15:04:05"))

		// Проверяем, что данные соответствуют
		if news.Title == "" {
			fmt.Println("⚠️  ПРОБЛЕМА: Пустой заголовок!")
		}
		if news.Content == "" {
			fmt.Println("⚠️  ПРОБЛЕМА: Пустой контент!")
		}
		if news.CategoryID == nil {
			fmt.Println("⚠️  ПРОБЛЕМА: Не определена категория!")
		}
		if news.SourceID != source.ID {
			fmt.Println("⚠️  ПРОБЛЕМА: Неправильный ID источника!")
		}
	}

	fmt.Println("\nТестирование завершено!")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
