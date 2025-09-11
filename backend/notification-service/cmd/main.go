package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"notification-service/internal/config"
	"notification-service/internal/database"
	"notification-service/internal/handlers"
	"notification-service/internal/repository"
	"notification-service/internal/services"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	// Настройка логирования
	logger := setupLogger(cfg.Logging)
	logger.Info("Starting Notification Service")

	// Подключение к базе данных
	db, err := database.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Запуск автоочистки уведомлений
	db.StartCleanupRoutine()

	// Создание репозиториев
	notificationRepo := repository.NewNotificationRepository(db.GetDB(), logger)

	// Создание сервисов
	websocketService := services.NewWebSocketService(cfg, logger)
	notificationService := services.NewNotificationService(cfg, logger, notificationRepo, websocketService)

	// Создание HTTP обработчиков
	handler := handlers.NewHandler(notificationService, websocketService, logger)

	// Настройка маршрутов
	mux := setupRoutes(handler, logger)

	// Настройка HTTP сервера
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
			// Простые метрики для демонстрации
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "# Notification Service Metrics\n")
			fmt.Fprintf(w, "# TYPE notification_service_info gauge\n")
			fmt.Fprintf(w, "notification_service_info{version=\"1.0.0\"} 1\n")
			
			// WebSocket статистика
			wsStats := websocketService.GetStats()
			if connected, ok := wsStats["connected"].(bool); ok {
				connectedValue := 0
				if connected {
					connectedValue = 1
				}
				fmt.Fprintf(w, "# TYPE notification_websocket_connected gauge\n")
				fmt.Fprintf(w, "notification_websocket_connected %d\n", connectedValue)
			}
			
			if queueSize, ok := wsStats["queue_size"].(int); ok {
				fmt.Fprintf(w, "# TYPE notification_websocket_queue_size gauge\n")
				fmt.Fprintf(w, "notification_websocket_queue_size %d\n", queueSize)
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

	// Запуск сервисов
	websocketService.Start()
	notificationService.Start()

	logger.Info("Notification Service started successfully")

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Notification Service...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Остановка сервисов
	notificationService.Stop()
	websocketService.Stop()

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

	logger.Info("Notification Service stopped")
}

// setupRoutes настраивает маршруты HTTP сервера
func setupRoutes(handler *handlers.Handler, logger *logrus.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", handler.HealthCheck)

	// Статистика сервиса
	mux.HandleFunc("/api/stats", handler.GetServiceStats)

	// Тестирование WebSocket
	mux.HandleFunc("/api/test/websocket", handler.TestWebSocket)

	// CRUD операции с уведомлениями
	mux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateNotification(w, r)
		case http.MethodGet:
			handler.GetNotification(w, r)
		case http.MethodPatch:
			handler.MarkNotificationAsRead(w, r)
		case http.MethodDelete:
			handler.DeleteNotification(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Уведомления пользователя
	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		if strings.Contains(path, "/notifications") {
			if strings.HasSuffix(path, "/mark-all-read") {
				handler.MarkAllNotificationsAsRead(w, r)
			} else if strings.HasSuffix(path, "/stats") {
				handler.GetNotificationStats(w, r)
			} else if strings.HasSuffix(path, "/unread-count") {
				handler.GetUnreadCount(w, r)
			} else {
				handler.GetUserNotifications(w, r)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	// Специальные типы уведомлений
	mux.HandleFunc("/api/notifications/news-alert", handler.CreateNewsAlert)
	mux.HandleFunc("/api/notifications/pulse-update", handler.CreatePulseUpdate)
	mux.HandleFunc("/api/notifications/system-message", handler.CreateSystemMessage)

	// Middleware для логирования
	return loggingMiddleware(mux, logger)
}

// loggingMiddleware добавляет логирование HTTP запросов
func loggingMiddleware(next http.Handler, logger *logrus.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Создаем wrapper для ResponseWriter чтобы захватить status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP request completed")
	})
	
	return mux
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

// responseWriter обертка для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

