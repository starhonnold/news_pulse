package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"api-gateway/internal/config"
	"api-gateway/internal/models"
)

// ProxyService представляет сервис для проксирования запросов к микросервисам
type ProxyService struct {
	config     *config.Config
	logger     *logrus.Logger
	httpClient *http.Client
	proxies    map[string]*httputil.ReverseProxy
}

// NewProxyService создает новый сервис прокси
func NewProxyService(config *config.Config, logger *logrus.Logger) *ProxyService {
	// Создаем HTTP клиент с настроенными таймаутами
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: config.Proxy.DialTimeout,
			}).DialContext,
			ResponseHeaderTimeout: config.Proxy.ResponseHeaderTimeout,
			DisableCompression:    config.Proxy.DisableCompression,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	service := &ProxyService{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
		proxies:    make(map[string]*httputil.ReverseProxy),
	}

	// Создаем reverse proxy для каждого микросервиса
	service.initializeProxies()

	return service
}

// initializeProxies создает reverse proxy для всех микросервисов
func (s *ProxyService) initializeProxies() {
	services := map[string]config.ServiceConfig{
		"news-parsing":    s.config.Services.NewsParsing,
		"news-management": s.config.Services.NewsManagement,
		"pulse":           s.config.Services.Pulse,
	}

	for name, serviceConfig := range services {
		targetURL, err := url.Parse(serviceConfig.URL)
		if err != nil {
			s.logger.WithError(err).WithField("service", name).Error("Failed to parse service URL")
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Настраиваем director для модификации запросов
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			s.modifyProxyRequest(req, serviceConfig)
		}

		// Настраиваем обработчик ошибок
		proxy.ErrorHandler = s.createErrorHandler(name)

		// Настраиваем модификацию ответов
		proxy.ModifyResponse = s.createResponseModifier(name)

		// Настраиваем flush interval
		proxy.FlushInterval = s.config.Proxy.FlushInterval

		s.proxies[name] = proxy

		s.logger.WithFields(logrus.Fields{
			"service": name,
			"url":     serviceConfig.URL,
		}).Info("Initialized reverse proxy for service")
	}
}

// ProxyRequest проксирует запрос к соответствующему микросервису
func (s *ProxyService) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	serviceName := s.determineTargetService(r.URL.Path)
	if serviceName == "" {
		s.sendNotFoundError(w, r, "Service not found for path: "+r.URL.Path)
		return
	}

	proxy, exists := s.proxies[serviceName]
	if !exists {
		s.sendServiceUnavailableError(w, r, fmt.Sprintf("Service %s is not available", serviceName))
		return
	}

	// Логируем начало проксирования
	s.logger.WithFields(logrus.Fields{
		"service":    serviceName,
		"method":     r.Method,
		"path":       r.URL.Path,
		"request_id": r.Header.Get("X-Request-ID"),
	}).Debug("Proxying request to service")

	// Проксируем запрос
	proxy.ServeHTTP(w, r)
}

// determineTargetService определяет целевой микросервис по пути
func (s *ProxyService) determineTargetService(path string) string {
	// Маршрутизация по префиксам путей
	switch {
	case strings.HasPrefix(path, "/api/news") && !strings.HasPrefix(path, "/api/news/parse"):
		return "news-management"
	case strings.HasPrefix(path, "/api/news/parse") || strings.HasPrefix(path, "/api/parsing"):
		return "news-parsing"
	case strings.HasPrefix(path, "/api/pulses") || strings.HasPrefix(path, "/api/feeds"):
		return "pulse"
	case strings.HasPrefix(path, "/api/categories") || strings.HasPrefix(path, "/api/countries"):
		return "news-management"
	default:
		return ""
	}
}

// modifyProxyRequest модифицирует запрос перед отправкой в микросервис
func (s *ProxyService) modifyProxyRequest(req *http.Request, serviceConfig config.ServiceConfig) {
	// Добавляем настроенные заголовки
	for key, value := range s.config.Proxy.AddHeaders {
		req.Header.Set(key, value)
	}

	// Удаляем настроенные заголовки
	for _, header := range s.config.Proxy.RemoveHeaders {
		req.Header.Del(header)
	}

	// Добавляем информацию о gateway
	req.Header.Set("X-Gateway-Request", "true")
	req.Header.Set("X-Gateway-Timestamp", time.Now().Format(time.RFC3339))

	// Сохраняем оригинальный Host для логирования
	req.Header.Set("X-Original-Host", req.Host)
}

// createErrorHandler создает обработчик ошибок для proxy
func (s *ProxyService) createErrorHandler(serviceName string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		requestID := r.Header.Get("X-Request-ID")

		s.logger.WithFields(logrus.Fields{
			"service":    serviceName,
			"error":      err.Error(),
			"request_id": requestID,
			"method":     r.Method,
			"path":       r.URL.Path,
		}).Error("Proxy error occurred")

		// Определяем тип ошибки и соответствующий HTTP статус
		var statusCode int
		var message string

		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host") {
			statusCode = http.StatusServiceUnavailable
			message = fmt.Sprintf("Service %s is temporarily unavailable", serviceName)
		} else if strings.Contains(err.Error(), "timeout") {
			statusCode = http.StatusGatewayTimeout
			message = fmt.Sprintf("Service %s request timeout", serviceName)
		} else {
			statusCode = http.StatusBadGateway
			message = fmt.Sprintf("Bad gateway for service %s", serviceName)
		}

		apiError := models.NewAPIError(models.ErrorCodeServiceUnavailable, message)
		response := models.NewErrorResponse(apiError, requestID)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)

		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			s.logger.WithError(encodeErr).Error("Failed to encode proxy error response")
		}
	}
}

// createResponseModifier создает модификатор ответов для proxy
func (s *ProxyService) createResponseModifier(serviceName string) func(*http.Response) error {
	return func(resp *http.Response) error {
		// Добавляем заголовки с информацией о сервисе
		resp.Header.Set("X-Service-Name", serviceName)
		resp.Header.Set("X-Gateway-Response", "true")

		// Удаляем внутренние заголовки микросервиса
		resp.Header.Del("X-Internal-Service")

		return nil
	}
}

// CheckServiceHealth проверяет здоровье микросервиса
func (s *ProxyService) CheckServiceHealth(ctx context.Context, serviceName string) models.ServiceHealth {
	serviceConfig, exists := s.getServiceConfig(serviceName)
	if !exists {
		return models.ServiceHealth{
			Name:   serviceName,
			Status: models.ServiceStatusUnknown,
			Error:  "Service configuration not found",
		}
	}

	health := models.ServiceHealth{
		Name:      serviceConfig.Name,
		URL:       serviceConfig.URL,
		LastCheck: time.Now(),
	}

	// Создаем контекст с таймаутом для health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Формируем URL для health check
	healthURL := serviceConfig.URL + serviceConfig.HealthEndpoint

	start := time.Now()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		health.Status = models.ServiceStatusUnhealthy
		health.Error = fmt.Sprintf("Failed to create health check request: %v", err)
		return health
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		health.Status = models.ServiceStatusUnhealthy
		health.Error = fmt.Sprintf("Health check request failed: %v", err)
		return health
	}
	defer resp.Body.Close()

	health.ResponseTime = time.Since(start)

	if resp.StatusCode == http.StatusOK {
		health.Status = models.ServiceStatusHealthy
	} else {
		health.Status = models.ServiceStatusUnhealthy
		health.Error = fmt.Sprintf("Health check returned status %d", resp.StatusCode)
	}

	return health
}

// CheckAllServicesHealth проверяет здоровье всех микросервисов
func (s *ProxyService) CheckAllServicesHealth(ctx context.Context) map[string]models.ServiceHealth {
	services := []string{"news-parsing", "news-management", "pulse"}
	results := make(map[string]models.ServiceHealth)

	// Проверяем все сервисы параллельно
	type healthResult struct {
		name   string
		health models.ServiceHealth
	}

	resultChan := make(chan healthResult, len(services))

	for _, serviceName := range services {
		go func(name string) {
			health := s.CheckServiceHealth(ctx, name)
			resultChan <- healthResult{name: name, health: health}
		}(serviceName)
	}

	// Собираем результаты
	for i := 0; i < len(services); i++ {
		result := <-resultChan
		results[result.name] = result.health
	}

	return results
}

// getServiceConfig возвращает конфигурацию сервиса по имени
func (s *ProxyService) getServiceConfig(serviceName string) (config.ServiceConfig, bool) {
	switch serviceName {
	case "news-parsing":
		return s.config.Services.NewsParsing, true
	case "news-management":
		return s.config.Services.NewsManagement, true
	case "pulse":
		return s.config.Services.Pulse, true
	default:
		return config.ServiceConfig{}, false
	}
}

// sendNotFoundError отправляет ошибку 404
func (s *ProxyService) sendNotFoundError(w http.ResponseWriter, r *http.Request, message string) {
	requestID := r.Header.Get("X-Request-ID")

	apiError := models.NewAPIError(models.ErrorCodeNotFound, message)
	response := models.NewErrorResponse(apiError, requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.WithError(err).Error("Failed to encode not found error response")
	}
}

// sendServiceUnavailableError отправляет ошибку 503
func (s *ProxyService) sendServiceUnavailableError(w http.ResponseWriter, r *http.Request, message string) {
	requestID := r.Header.Get("X-Request-ID")

	apiError := models.NewAPIError(models.ErrorCodeServiceUnavailable, message)
	response := models.NewErrorResponse(apiError, requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.WithError(err).Error("Failed to encode service unavailable error response")
	}
}

// GetStats возвращает статистику прокси сервиса
func (s *ProxyService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_proxies": len(s.proxies),
		"services":       s.getServicesList(),
	}
}

// getServicesList возвращает список настроенных сервисов
func (s *ProxyService) getServicesList() []string {
	services := make([]string, 0, len(s.proxies))
	for name := range s.proxies {
		services = append(services, name)
	}
	return services
}
