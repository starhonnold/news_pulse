package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/services"
)

// Handler представляет HTTP обработчики
type Handler struct {
	parsingService *services.ParsingService
	logger         *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(parsingService *services.ParsingService, logger *logrus.Logger) *Handler {
	return &Handler{
		parsingService: parsingService,
		logger:         logger,
	}
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// HealthCheck возвращает статус здоровья сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "news-parsing-service",
		"version":   "1.0.0",
		"uptime":    time.Since(startTime),
	}
	
	// Проверяем статус парсинга
	if h.parsingService.IsRunning() {
		health["parsing_status"] = "running"
	} else {
		health["parsing_status"] = "stopped"
	}
	
	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    health,
	})
}

// GetStats возвращает статистику парсинга
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.parsingService.GetStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get parsing stats")
		h.sendResponse(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to get statistics",
		})
		return
	}
	
	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// ParseAllSources запускает парсинг всех источников
func (h *Handler) ParseAllSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendResponse(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}
	
	go func() {
		if err := h.parsingService.ParseAllSources(r.Context()); err != nil {
			h.logger.WithError(err).Error("Failed to parse all sources")
		}
	}()
	
	h.sendResponse(w, http.StatusAccepted, Response{
		Success: true,
		Message: "Parsing started",
	})
}

// ParseSource запускает парсинг конкретного источника
func (h *Handler) ParseSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendResponse(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}
	
	// Получаем ID источника из URL параметра
	sourceIDStr := r.URL.Query().Get("source_id")
	if sourceIDStr == "" {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "source_id parameter is required",
		})
		return
	}
	
	sourceID, err := strconv.Atoi(sourceIDStr)
	if err != nil {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid source_id parameter",
		})
		return
	}
	
	go func() {
		if err := h.parsingService.ParseSource(r.Context(), sourceID); err != nil {
			h.logger.WithError(err).WithField("source_id", sourceID).Error("Failed to parse source")
		}
	}()
	
	h.sendResponse(w, http.StatusAccepted, Response{
		Success: true,
		Message: "Source parsing started",
		Data: map[string]interface{}{
			"source_id": sourceID,
		},
	})
}

// ValidateSource проверяет корректность RSS ленты
func (h *Handler) ValidateSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendResponse(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}
	
	var requestBody struct {
		RSSURL string `json:"rss_url"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	if requestBody.RSSURL == "" {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "rss_url is required",
		})
		return
	}
	
	if err := h.parsingService.ValidateSource(r.Context(), requestBody.RSSURL); err != nil {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	
	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "RSS feed is valid",
		Data: map[string]interface{}{
			"rss_url": requestBody.RSSURL,
		},
	})
}

// GetFeedInfo возвращает информацию о RSS ленте
func (h *Handler) GetFeedInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendResponse(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}
	
	var requestBody struct {
		RSSURL string `json:"rss_url"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	if requestBody.RSSURL == "" {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "rss_url is required",
		})
		return
	}
	
	feedInfo, err := h.parsingService.GetFeedInfo(r.Context(), requestBody.RSSURL)
	if err != nil {
		h.sendResponse(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	
	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    feedInfo,
	})
}

// GetStatus возвращает статус сервиса парсинга
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service_running": h.parsingService.IsRunning(),
		"timestamp":       time.Now(),
	}
	
	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// sendResponse отправляет JSON ответ
func (h *Handler) sendResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode response")
	}
}

// corsMiddleware добавляет CORS заголовки (отключен, так как CORS обрабатывается в API Gateway)
func (h *Handler) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS заголовки обрабатываются в API Gateway
		// Не добавляем CORS заголовки здесь, чтобы избежать дублирования
		next(w, r)
	}
}

// loggingMiddleware логирует HTTP запросы
func (h *Handler) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Создаем wrapper для ResponseWriter чтобы захватить status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next(wrapped, r)
		
		duration := time.Since(start)
		
		h.logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"url":         r.URL.String(),
			"status_code": wrapped.statusCode,
			"duration":    duration,
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP request")
	}
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

// SetupRoutes настраивает маршруты HTTP сервера
func (h *Handler) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", h.loggingMiddleware(h.corsMiddleware(h.HealthCheck)))
	
	// API endpoints
	mux.HandleFunc("/api/stats", h.loggingMiddleware(h.corsMiddleware(h.GetStats)))
	mux.HandleFunc("/api/status", h.loggingMiddleware(h.corsMiddleware(h.GetStatus)))
	mux.HandleFunc("/api/parse/all", h.loggingMiddleware(h.corsMiddleware(h.ParseAllSources)))
	mux.HandleFunc("/api/parse/source", h.loggingMiddleware(h.corsMiddleware(h.ParseSource)))
	mux.HandleFunc("/api/validate", h.loggingMiddleware(h.corsMiddleware(h.ValidateSource)))
	mux.HandleFunc("/api/feed-info", h.loggingMiddleware(h.corsMiddleware(h.GetFeedInfo)))
	
	return mux
}

var startTime = time.Now() // Время запуска сервиса
