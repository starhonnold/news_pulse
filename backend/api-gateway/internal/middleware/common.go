package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"api-gateway/internal/models"
)

// RequestIDMiddleware добавляет уникальный ID к каждому запросу
func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Добавляем ID в контекст
		ctx := context.WithValue(r.Context(), "request_id", requestID)

		// Добавляем ID в заголовок ответа
		w.Header().Set("X-Request-ID", requestID)

		next(w, r.WithContext(ctx))
	}
}

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware(config *CORSConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				next(w, r)
				return
			}

			origin := r.Header.Get("Origin")

			// Отладочная информация
			logrus.WithFields(logrus.Fields{
				"origin":          origin,
				"allowed_origins": config.AllowedOrigins,
				"enabled":         config.Enabled,
			}).Debug("CORS middleware processing request")

			// Проверяем разрешенные origins
			if isAllowedOrigin(origin, config.AllowedOrigins) {
				// Проверяем, не установлен ли уже заголовок
				if w.Header().Get("Access-Control-Allow-Origin") == "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					logrus.WithField("origin", origin).Debug("CORS: Set Access-Control-Allow-Origin")
				}
			} else {
				logrus.WithFields(logrus.Fields{
					"origin":          origin,
					"allowed_origins": config.AllowedOrigins,
				}).Warn("CORS: Origin not allowed")
			}

			// Устанавливаем остальные CORS заголовки только если они не установлены
			if w.Header().Get("Access-Control-Allow-Methods") == "" {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}
			if w.Header().Get("Access-Control-Allow-Headers") == "" {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}
			if w.Header().Get("Access-Control-Expose-Headers") == "" {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}
			if w.Header().Get("Access-Control-Max-Age") == "" {
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Обрабатываем preflight запросы
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}
}

// LoggingMiddleware логирует HTTP запросы
func LoggingMiddleware(logger *logrus.Logger, slowThreshold int) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем wrapper для ResponseWriter чтобы захватить status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Создаем контекст запроса
			ctx := &models.RequestContext{
				RequestID: GetRequestID(r.Context()),
				UserID:    GetUserIDFromContext(r.Context()),
				Username:  GetUsernameFromContext(r.Context()),
				StartTime: start,
				Method:    r.Method,
				Path:      r.URL.Path,
				UserAgent: r.UserAgent(),
				IP:        GetClientIP(r),
			}

			// Добавляем контекст в request
			r = r.WithContext(context.WithValue(r.Context(), "request_context", ctx))

			next(wrapped, r)

			duration := time.Since(start)

			// Определяем уровень логирования
			logLevel := logrus.InfoLevel
			if wrapped.statusCode >= 400 {
				logLevel = logrus.WarnLevel
			}
			if wrapped.statusCode >= 500 {
				logLevel = logrus.ErrorLevel
			}
			if duration.Milliseconds() > int64(slowThreshold) {
				logLevel = logrus.WarnLevel
			}

			logger.WithFields(logrus.Fields{
				"request_id":  ctx.RequestID,
				"method":      ctx.Method,
				"path":        ctx.Path,
				"status_code": wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"user_id":     ctx.UserID,
				"username":    ctx.Username,
				"ip":          ctx.IP,
				"user_agent":  ctx.UserAgent,
			}).Log(logLevel, "HTTP request completed")
		}
	}
}

// RecoveryMiddleware восстанавливается после паники
func RecoveryMiddleware(logger *logrus.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())

					logger.WithFields(logrus.Fields{
						"request_id": requestID,
						"method":     r.Method,
						"path":       r.URL.Path,
						"panic":      err,
						"ip":         GetClientIP(r),
					}).Error("Panic recovered in HTTP handler")

					// Отправляем ошибку 500
					apiError := models.NewAPIError(models.ErrorCodeInternalError, "Internal server error")
					response := models.NewErrorResponse(apiError, requestID)

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)

					// Игнорируем ошибку encoding, так как мы уже в recovery
					_ = json.NewEncoder(w).Encode(response)
				}
			}()

			next(w, r)
		}
	}
}

// SecurityHeadersMiddleware добавляет заголовки безопасности
func SecurityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Добавляем заголовки безопасности
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next(w, r)
	}
}

// Вспомогательные функции

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback на timestamp если не удалось сгенерировать случайные байты
		return string(rune(time.Now().UnixNano()))
	}
	return hex.EncodeToString(bytes)
}

// isAllowedOrigin проверяет, разрешен ли origin
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}

		// Поддержка wildcard поддоменов (например, *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}

// GetClientIP извлекает реальный IP клиента
func GetClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For может содержать список IP через запятую
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := r.Header.Get("X-Client-IP"); ip != "" {
		return ip
	}

	// Извлекаем IP из RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// GetRequestID извлекает ID запроса из контекста
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// GetRequestContext извлекает контекст запроса
func GetRequestContext(ctx context.Context) *models.RequestContext {
	if reqCtx, ok := ctx.Value("request_context").(*models.RequestContext); ok {
		return reqCtx
	}
	return nil
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

// CORSConfig для использования в middleware
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}
