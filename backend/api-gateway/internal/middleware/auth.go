package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"api-gateway/internal/config"
	"api-gateway/internal/models"
)

// AuthMiddleware представляет middleware для аутентификации
type AuthMiddleware struct {
	config *config.Config
	logger *logrus.Logger
}

// NewAuthMiddleware создает новый middleware аутентификации
func NewAuthMiddleware(config *config.Config, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
		logger: logger,
	}
}

// Middleware возвращает HTTP middleware функцию
func (m *AuthMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем аутентификацию если она отключена
		if !m.config.Auth.Enabled {
			next(w, r)
			return
		}
		
		// Проверяем, является ли маршрут публичным
		if m.config.IsPublicRoute(r.URL.Path) {
			next(w, r)
			return
		}
		
		// Извлекаем токен из заголовка
		token := m.extractToken(r)
		if token == "" {
			m.sendUnauthorizedError(w, r, "Missing authorization token")
			return
		}
		
		// Валидируем токен
		claims, err := m.validateToken(token)
		if err != nil {
			m.sendUnauthorizedError(w, r, err.Error())
			return
		}
		
		// Проверяем, не истек ли токен
		if claims.IsExpired() {
			m.sendUnauthorizedError(w, r, "Token has expired")
			return
		}
		
		// Добавляем информацию о пользователе в контекст
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "email", claims.Email)
		
		// Добавляем заголовок с ID пользователя для микросервисов
		r.Header.Set("X-User-ID", string(rune(claims.UserID)))
		
		// Логируем успешную аутентификацию
		m.logger.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"path":     r.URL.Path,
			"method":   r.Method,
		}).Debug("User authenticated successfully")
		
		// Передаем управление следующему middleware
		next(w, r.WithContext(ctx))
	}
}

// extractToken извлекает JWT токен из заголовка Authorization
func (m *AuthMiddleware) extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	
	// Ожидаем формат "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	
	return parts[1]
}

// validateToken валидирует JWT токен и возвращает claims
func (m *AuthMiddleware) validateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(m.config.Auth.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if !token.Valid {
		return nil, jwt.ErrTokenNotValidYet
	}
	
	// Извлекаем claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return m.parseClaimsFromMap(claims)
	}
	
	return nil, jwt.ErrInvalidKey
}

// parseClaimsFromMap парсит claims из map
func (m *AuthMiddleware) parseClaimsFromMap(claims jwt.MapClaims) (*models.JWTClaims, error) {
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}
	
	username, ok := claims["username"].(string)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}
	
	email, ok := claims["email"].(string)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}
	
	issuedAt, ok := claims["iat"].(float64)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}
	
	expiresAt, ok := claims["exp"].(float64)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}
	
	return &models.JWTClaims{
		UserID:    int(userID),
		Username:  username,
		Email:     email,
		IssuedAt:  int64(issuedAt),
		ExpiresAt: int64(expiresAt),
	}, nil
}

// sendUnauthorizedError отправляет ошибку 401
func (m *AuthMiddleware) sendUnauthorizedError(w http.ResponseWriter, r *http.Request, message string) {
	requestID := GetRequestID(r.Context())
	
	apiError := models.NewAPIError(models.ErrorCodeUnauthorized, message)
	response := models.NewErrorResponse(apiError, requestID)
	
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		m.logger.WithError(err).Error("Failed to encode unauthorized error response")
	}
	
	m.logger.WithFields(logrus.Fields{
		"path":       r.URL.Path,
		"method":     r.Method,
		"error":      message,
		"request_id": requestID,
		"ip":         GetClientIP(r),
	}).Warn("Authentication failed")
}

// GenerateToken создает новый JWT токен для пользователя
func (m *AuthMiddleware) GenerateToken(user models.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.GetJWTExpiration())
	
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"iat":      now.Unix(),
		"exp":      expiresAt.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Auth.JWTSecret))
}

// GenerateRefreshToken создает новый refresh токен
func (m *AuthMiddleware) GenerateRefreshToken(user models.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.GetJWTRefreshExpiration())
	
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Auth.JWTSecret))
}

// ValidateRefreshToken валидирует refresh токен
func (m *AuthMiddleware) ValidateRefreshToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(m.config.Auth.JWTSecret), nil
	})
	
	if err != nil {
		return 0, err
	}
	
	if !token.Valid {
		return 0, jwt.ErrTokenNotValidYet
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Проверяем тип токена
		tokenType, ok := claims["type"].(string)
		if !ok || tokenType != "refresh" {
			return 0, jwt.ErrInvalidKey
		}
		
		userID, ok := claims["user_id"].(float64)
		if !ok {
			return 0, jwt.ErrInvalidKey
		}
		
		return int(userID), nil
	}
	
	return 0, jwt.ErrInvalidKey
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) int {
	if userID, ok := ctx.Value("user_id").(int); ok {
		return userID
	}
	return 0
}

// GetUsernameFromContext извлекает имя пользователя из контекста
func GetUsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value("username").(string); ok {
		return username
	}
	return ""
}

// GetEmailFromContext извлекает email пользователя из контекста
func GetEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value("email").(string); ok {
		return email
	}
	return ""
}
