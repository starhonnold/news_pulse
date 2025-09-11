package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"api-gateway/internal/config"
	"api-gateway/internal/models"
)

// RateLimitMiddleware представляет middleware для rate limiting
type RateLimitMiddleware struct {
	config        *config.Config
	logger        *logrus.Logger
	limiters      map[string]*rate.Limiter
	mu            sync.RWMutex
	globalLimiter *rate.Limiter
}

// NewRateLimitMiddleware создает новый middleware rate limiting
func NewRateLimitMiddleware(config *config.Config, logger *logrus.Logger) *RateLimitMiddleware {
	// Создаем глобальный лимитер
	globalLimiter := rate.NewLimiter(
		rate.Limit(config.RateLimiting.Global.RequestsPerMinute)/60, // requests per second
		config.RateLimiting.Global.Burst,
	)

	return &RateLimitMiddleware{
		config:        config,
		logger:        logger,
		limiters:      make(map[string]*rate.Limiter),
		globalLimiter: globalLimiter,
	}
}

// Middleware возвращает HTTP middleware функцию
func (m *RateLimitMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем rate limiting если он отключен
		if !m.config.RateLimiting.Enabled {
			next(w, r)
			return
		}

		clientIP := GetClientIP(r)

		// Проверяем белый список IP
		if m.config.IsWhitelistedIP(clientIP) {
			next(w, r)
			return
		}

		// Проверяем глобальный лимит
		if !m.globalLimiter.Allow() {
			m.sendRateLimitError(w, r, "Global rate limit exceeded", time.Second)
			return
		}

		// Определяем ключ для лимитера
		var limiterKey string
		var rateRule config.RateLimitRule

		userID := GetUserIDFromContext(r.Context())
		if userID > 0 {
			// Аутентифицированный пользователь
			limiterKey = fmt.Sprintf("user:%d", userID)
			rateRule = m.config.RateLimiting.PerUser
		} else {
			// Анонимный пользователь
			limiterKey = fmt.Sprintf("ip:%s", clientIP)
			rateRule = m.config.RateLimiting.Anonymous
		}

		// Получаем или создаем лимитер для этого ключа
		limiter := m.getLimiter(limiterKey, rateRule)

		// Проверяем лимит
		if !limiter.Allow() {
			// Вычисляем время до сброса лимита
			retryAfter := time.Until(time.Now().Add(time.Minute))
			m.sendRateLimitError(w, r, "Rate limit exceeded", retryAfter)

			m.logger.WithFields(logrus.Fields{
				"limiter_key": limiterKey,
				"client_ip":   clientIP,
				"user_id":     userID,
				"path":        r.URL.Path,
				"method":      r.Method,
			}).Warn("Rate limit exceeded")

			return
		}

		// Добавляем заголовки с информацией о лимитах
		m.addRateLimitHeaders(w, limiter, rateRule)

		// Передаем управление следующему middleware
		next(w, r)
	}
}

// getLimiter получает или создает лимитер для ключа
func (m *RateLimitMiddleware) getLimiter(key string, rule config.RateLimitRule) *rate.Limiter {
	m.mu.RLock()
	limiter, exists := m.limiters[key]
	m.mu.RUnlock()

	if exists {
		return limiter
	}

	// Создаем новый лимитер
	m.mu.Lock()
	defer m.mu.Unlock()

	// Проверяем еще раз на случай гонки
	if limiter, exists := m.limiters[key]; exists {
		return limiter
	}

	// Создаем лимитер: requests per minute / 60 = requests per second
	limiter = rate.NewLimiter(
		rate.Limit(rule.RequestsPerMinute)/60,
		rule.Burst,
	)

	m.limiters[key] = limiter

	// Запускаем горутину для очистки неиспользуемых лимитеров
	go m.cleanupLimiter(key, time.Hour)

	return limiter
}

// cleanupLimiter удаляет лимитер через указанное время
func (m *RateLimitMiddleware) cleanupLimiter(key string, after time.Duration) {
	time.Sleep(after)

	m.mu.Lock()
	delete(m.limiters, key)
	m.mu.Unlock()

	m.logger.WithField("limiter_key", key).Debug("Cleaned up unused rate limiter")
}

// addRateLimitHeaders добавляет заголовки с информацией о лимитах
func (m *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, limiter *rate.Limiter, rule config.RateLimitRule) {
	// Приблизительное количество оставшихся запросов
	// Это упрощенная реализация, в production лучше использовать более точный подсчет
	tokens := limiter.Tokens()
	remaining := int(tokens)
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rule.RequestsPerMinute))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
}

// sendRateLimitError отправляет ошибку 429 Too Many Requests
func (m *RateLimitMiddleware) sendRateLimitError(w http.ResponseWriter, r *http.Request, message string, retryAfter time.Duration) {
	requestID := GetRequestID(r.Context())

	apiError := models.NewAPIError(models.ErrorCodeRateLimitExceeded, message)
	response := models.NewErrorResponse(apiError, requestID)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
	w.WriteHeader(http.StatusTooManyRequests)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.WithError(err).Error("Failed to encode rate limit error response")
	}
}

// GetStats возвращает статистику rate limiting
func (m *RateLimitMiddleware) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":         m.config.RateLimiting.Enabled,
		"active_limiters": len(m.limiters),
		"global_tokens":   m.globalLimiter.Tokens(),
	}

	// Добавляем информацию о лимитерах
	limiterStats := make(map[string]interface{})
	for key, limiter := range m.limiters {
		limiterStats[key] = map[string]interface{}{
			"tokens": limiter.Tokens(),
			"limit":  limiter.Limit(),
			"burst":  limiter.Burst(),
		}
	}
	stats["limiters"] = limiterStats

	return stats
}

// Reset сбрасывает все лимитеры (для тестирования)
func (m *RateLimitMiddleware) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.limiters = make(map[string]*rate.Limiter)

	// Пересоздаем глобальный лимитер
	m.globalLimiter = rate.NewLimiter(
		rate.Limit(m.config.RateLimiting.Global.RequestsPerMinute)/60,
		m.config.RateLimiting.Global.Burst,
	)

	m.logger.Info("Rate limiters reset")
}
