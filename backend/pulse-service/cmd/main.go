package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"pulse-service/internal/cache"
	"pulse-service/internal/config"
	"pulse-service/internal/database"
	"pulse-service/internal/handlers"
	"pulse-service/internal/repository"
	"pulse-service/internal/services"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	// Настройка логирования
	logger := setupLogger(cfg.Logging)
	logger.Info("Starting Pulse Service")

	// Подключение к базе данных
	db, err := database.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Проверка базы данных
	if err := db.Migrate(); err != nil {
		logger.WithError(err).Fatal("Failed to migrate database")
	}

	// Создание индексов для оптимизации
	if err := db.CreatePulseIndexes(); err != nil {
		logger.WithError(err).Warn("Failed to create pulse indexes")
	}

	// Создание кеша
	cacheInstance := cache.NewCache(cfg.Caching, logger)

	// Создание репозиториев
	pulseRepo := repository.NewPulseRepository(db, logger)
	feedRepo := repository.NewFeedRepository(db, logger)

	// Создание сервисов
	pulseService := services.NewPulseService(
		pulseRepo,
		feedRepo,
		cacheInstance,
		cfg,
		logger,
		db,
	)

	// Создание HTTP обработчиков
	handler := handlers.NewHandler(pulseService, logger)

	// Настройка HTTP сервера
	mux := handler.SetupRoutes()
	server := &http.Server{
		Addr:         cfg.GetServerAddr(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Запуск HTTP сервера в горутине
	go func() {
		logger.WithField("addr", server.Addr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Запуск health check сервера (если настроен)
	var healthServer *http.Server
	if cfg.Health.Port != cfg.Server.Port {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc(cfg.Health.Path, handler.HealthCheck)

		healthServer = &http.Server{
			Addr:    cfg.GetHealthAddr(),
			Handler: healthMux,
		}

		go func() {
			logger.WithField("addr", healthServer.Addr).Info("Starting health check server")
			if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Health check server failed")
			}
		}()
	}

	// Запуск metrics сервера (если включен)
	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		metricsMux := http.NewServeMux()
		metricsMux.HandleFunc(cfg.Metrics.Path, func(w http.ResponseWriter, r *http.Request) {
			// Здесь можно добавить Prometheus метрики
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "# Pulse Service Metrics\n")
			fmt.Fprintf(w, "# TYPE pulse_service_info gauge\n")
			fmt.Fprintf(w, "pulse_service_info{version=\"1.0.0\"} 1\n")

			// Добавляем метрики кеша
			cacheStats := pulseService.GetCacheStats()
			if enabled, ok := cacheStats["enabled"].(bool); ok && enabled {
				if totalItems, ok := cacheStats["total_items"].(int); ok {
					fmt.Fprintf(w, "# TYPE pulse_cache_items gauge\n")
					fmt.Fprintf(w, "pulse_cache_items %d\n", totalItems)
				}
			}
		})

		metricsServer = &http.Server{
			Addr:    cfg.GetMetricsAddr(),
			Handler: metricsMux,
		}

		go func() {
			logger.WithField("addr", metricsServer.Addr).Info("Starting metrics server")
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Metrics server failed")
			}
		}()
	}

	logger.Info("Pulse Service started successfully")

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Pulse Service...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Остановка HTTP серверов
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Failed to shutdown HTTP server")
	}

	if healthServer != nil {
		if err := healthServer.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Failed to shutdown health server")
		}
	}

	if metricsServer != nil {
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Failed to shutdown metrics server")
		}
	}

	logger.Info("Pulse Service stopped")
}

// setupLogger настраивает логирование
func setupLogger(cfg config.LoggingConfig) *logrus.Logger {
	logger := logrus.New()

	// Установка уровня логирования
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Установка формата логирования
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Установка вывода
	if cfg.Output == "file" && cfg.FilePath != "" {
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.WithError(err).Warn("Failed to open log file, using stdout")
		} else {
			logger.SetOutput(file)
		}
	}

	return logger
}
