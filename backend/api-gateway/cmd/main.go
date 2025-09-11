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

	"api-gateway/internal/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
	"api-gateway/internal/services"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	// Настройка логирования
	logger := setupLogger(cfg.Logging)
	logger.Info("Starting API Gateway")

	// Создание сервисов
	proxyService := services.NewProxyService(cfg, logger)
	websocketService := services.NewWebSocketService(cfg, logger)

	// Создание middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(cfg, logger)

	// Создание HTTP обработчиков
	handler := handlers.NewHandler(proxyService, websocketService, authMiddleware, logger)

	// Настройка маршрутов
	mux := setupRoutes(handler, authMiddleware, rateLimitMiddleware, cfg, logger)

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
		healthMux.HandleFunc(cfg.Health.Path, applyMiddleware(
			handler.HealthCheck,
			middleware.RequestIDMiddleware,
			middleware.LoggingMiddleware(logger, cfg.Logging.SlowRequestThreshold),
		))
		
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
			fmt.Fprintf(w, "# API Gateway Metrics\n")
			fmt.Fprintf(w, "# TYPE api_gateway_info gauge\n")
			fmt.Fprintf(w, "api_gateway_info{version=\"1.0.0\"} 1\n")
			
			// Добавляем метрики WebSocket соединений
			wsStats := websocketService.GetStats()
			if enabled, ok := wsStats["enabled"].(bool); ok && enabled {
				if totalConns, ok := wsStats["total_connections"].(int); ok {
					fmt.Fprintf(w, "# TYPE api_gateway_websocket_connections gauge\n")
					fmt.Fprintf(w, "api_gateway_websocket_connections %d\n", totalConns)
				}
			}
			
			// Добавляем метрики rate limiting
			rateLimitStats := rateLimitMiddleware.GetStats()
			if enabled, ok := rateLimitStats["enabled"].(bool); ok && enabled {
				if activeLimiters, ok := rateLimitStats["active_limiters"].(int); ok {
					fmt.Fprintf(w, "# TYPE api_gateway_rate_limiters gauge\n")
					fmt.Fprintf(w, "api_gateway_rate_limiters %d\n", activeLimiters)
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

	logger.Info("API Gateway started successfully")

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API Gateway...")

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

	logger.Info("API Gateway stopped")
}

// setupRoutes настраивает маршруты HTTP сервера
func setupRoutes(
	handler *handlers.Handler,
	authMiddleware *middleware.AuthMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	cfg *config.Config,
	logger *logrus.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	
	// Базовые middleware для всех запросов
	baseMiddleware := []func(http.HandlerFunc) http.HandlerFunc{
		middleware.RequestIDMiddleware,
		middleware.RecoveryMiddleware(logger),
		middleware.SecurityHeadersMiddleware,
		middleware.CORSMiddleware(&middleware.CORSConfig{
			Enabled:          cfg.CORS.Enabled,
			AllowedOrigins:   cfg.CORS.AllowedOrigins,
			AllowedMethods:   cfg.CORS.AllowedMethods,
			AllowedHeaders:   cfg.CORS.AllowedHeaders,
			ExposedHeaders:   cfg.CORS.ExposedHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAge,
		}),
		middleware.LoggingMiddleware(logger, cfg.Logging.SlowRequestThreshold),
		rateLimitMiddleware.Middleware,
	}
	
	// Middleware с аутентификацией (только если аутентификация включена)
	var authMiddlewareChain []func(http.HandlerFunc) http.HandlerFunc
	if cfg.Auth.Enabled {
		authMiddlewareChain = append(baseMiddleware, authMiddleware.Middleware)
	} else {
		authMiddlewareChain = baseMiddleware
	}
	
	// Health check (публичный)
	mux.HandleFunc("/health", applyMiddleware(handler.HealthCheck, baseMiddleware...))
	
	// Аутентификация (публичные маршруты)
	mux.HandleFunc("/api/auth/login", applyMiddleware(handler.Login, baseMiddleware...))
	mux.HandleFunc("/api/auth/register", applyMiddleware(handler.Register, baseMiddleware...))
	mux.HandleFunc("/api/auth/refresh", applyMiddleware(handler.RefreshToken, baseMiddleware...))
	
	// WebSocket (отдельная аутентификация)
	if cfg.WebSocket.Enabled {
		mux.HandleFunc(cfg.WebSocket.Path, applyMiddleware(handler.HandleWebSocket, baseMiddleware...))
	}
	
	// Статистика (требует аутентификации только если она включена)
	if cfg.Auth.Enabled {
		mux.HandleFunc("/api/stats", applyMiddleware(handler.GetStats, authMiddlewareChain...))
	} else {
		mux.HandleFunc("/api/stats", applyMiddleware(handler.GetStats, baseMiddleware...))
	}
	
	// Прокси ко всем остальным API
	mux.HandleFunc("/api/", applyMiddleware(handler.ProxyRequest, authMiddlewareChain...))
	
	// 404 для всех остальных маршрутов
	mux.HandleFunc("/", applyMiddleware(handler.NotFound, baseMiddleware...))
	
	return mux
}

// applyMiddleware применяет цепочку middleware к handler
func applyMiddleware(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
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
