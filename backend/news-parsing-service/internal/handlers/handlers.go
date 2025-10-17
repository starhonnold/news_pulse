package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/models"
	"news-parsing-service/internal/repository"
	"news-parsing-service/internal/services"
)

// Handler представляет HTTP обработчики
type Handler struct {
	parsingService   *services.ParsingService
	newsSourceRepo   *repository.NewsSourceRepository
	logger           *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(parsingService *services.ParsingService, newsSourceRepo *repository.NewsSourceRepository, logger *logrus.Logger) *Handler {
	return &Handler{
		parsingService: parsingService,
		newsSourceRepo: newsSourceRepo,
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
	stats := h.parsingService.GetStats()

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
		h.parsingService.ParseAllSources()
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
		// Получаем источник по ID
		source, err := h.newsSourceRepo.GetByID(r.Context(), sourceID)
		if err != nil {
			h.logger.WithError(err).WithField("source_id", sourceID).Error("Failed to get source")
			return
		}
		h.parsingService.ParseSource(r.Context(), *source)
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

	source := models.NewsSource{RSSURL: requestBody.RSSURL}
	if err := h.parsingService.ValidateSource(source); err != nil {
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

// ExtractContentRequest представляет запрос на извлечение контента
type ExtractContentRequest struct {
	URL string `json:"url"`
}

// ExtractContentResponse представляет ответ с извлеченным контентом
type ExtractContentResponse struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Length  int    `json:"length"`
}

// ExtractContent извлекает контент с веб-страницы
func (h *Handler) ExtractContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExtractContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Извлекаем контент
	content, err := h.parsingService.ExtractContent(r.Context(), req.URL)
	if err != nil {
		h.logger.WithError(err).WithField("url", req.URL).Error("Failed to extract content")
		response := Response{
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Success: true,
		Data: ExtractContentResponse{
			URL:     req.URL,
			Content: content,
			Length:  len(content),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	mux.HandleFunc("/api/extract-content", h.loggingMiddleware(h.corsMiddleware(h.ExtractContent)))

	return mux
}

var startTime = time.Now() // Время запуска сервиса
