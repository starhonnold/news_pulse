package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"pulse-service/internal/config"
)

// Cache представляет простой in-memory кеш для пульсов
type Cache struct {
	data   map[string]*cacheItem
	mu     sync.RWMutex
	config config.CachingConfig
	logger *logrus.Logger
}

// cacheItem представляет элемент кеша
type cacheItem struct {
	value     interface{}
	expiredAt time.Time
}

// NewCache создает новый кеш
func NewCache(cfg config.CachingConfig, logger *logrus.Logger) *Cache {
	cache := &Cache{
		data:   make(map[string]*cacheItem),
		config: cfg,
		logger: logger,
	}

	if cfg.Enabled {
		// Запускаем горутину для очистки устаревших элементов
		go cache.cleanup()
	}

	return cache
}

// Set сохраняет значение в кеше
func (c *Cache) Set(key string, value interface{}, ttl int) error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем максимальный размер кеша
	if len(c.data) >= c.config.MaxSize {
		c.evictLRU()
	}

	expiredAt := time.Now().Add(time.Duration(ttl) * time.Second)
	c.data[key] = &cacheItem{
		value:     value,
		expiredAt: expiredAt,
	}

	c.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl,
	}).Debug("Cache item set")

	return nil
}

// Get получает значение из кеша
func (c *Cache) Get(key string, result interface{}) (bool, error) {
	if !c.config.Enabled {
		return false, nil
	}

	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Проверяем, не истек ли срок действия
	if time.Now().After(item.expiredAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return false, nil
	}

	// Десериализуем значение
	if err := c.deserialize(item.value, result); err != nil {
		return false, fmt.Errorf("failed to deserialize cache value: %w", err)
	}

	c.logger.WithField("key", key).Debug("Cache hit")
	return true, nil
}

// Delete удаляет значение из кеша
func (c *Cache) Delete(key string) {
	if !c.config.Enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	c.logger.WithField("key", key).Debug("Cache item deleted")
}

// DeleteByPattern удаляет все ключи, соответствующие паттерну
func (c *Cache) DeleteByPattern(pattern string) {
	if !c.config.Enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	deleted := 0
	for key := range c.data {
		if matchPattern(key, pattern) {
			delete(c.data, key)
			deleted++
		}
	}

	if deleted > 0 {
		c.logger.WithFields(logrus.Fields{
			"pattern": pattern,
			"deleted": deleted,
		}).Debug("Cache items deleted by pattern")
	}
}

// Clear очищает весь кеш
func (c *Cache) Clear() {
	if !c.config.Enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*cacheItem)
	c.logger.Info("Cache cleared")
}

// GetStats возвращает статистику кеша
func (c *Cache) GetStats() map[string]interface{} {
	if !c.config.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	expired := 0
	now := time.Now()

	for _, item := range c.data {
		if now.After(item.expiredAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"enabled":     true,
		"total_items": len(c.data),
		"expired":     expired,
		"max_size":    c.config.MaxSize,
	}
}

// cleanup периодически очищает устаревшие элементы
func (c *Cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		deleted := 0

		for key, item := range c.data {
			if now.After(item.expiredAt) {
				delete(c.data, key)
				deleted++
			}
		}

		c.mu.Unlock()

		if deleted > 0 {
			c.logger.WithField("deleted_items", deleted).Debug("Cache cleanup completed")
		}
	}
}

// evictLRU удаляет элементы по принципу LRU (простая реализация - удаляем случайный элемент)
func (c *Cache) evictLRU() {
	// Простая реализация - удаляем первый попавшийся элемент
	// В production лучше использовать настоящий LRU алгоритм
	for key := range c.data {
		delete(c.data, key)
		c.logger.WithField("evicted_key", key).Debug("Cache item evicted")
		break
	}
}

// deserialize десериализует значение из кеша
func (c *Cache) deserialize(cached interface{}, result interface{}) error {
	// Если это уже нужный тип, просто копируем
	if cached == result {
		return nil
	}

	// Сериализуем в JSON и обратно для универсальности
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

// matchPattern проверяет, соответствует ли строка простому паттерну с *
func matchPattern(str, pattern string) bool {
	// Простая реализация для паттернов вида "prefix:*"
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}

	return str == pattern
}

// Вспомогательные методы для создания ключей кеша

// UserPulsesCacheKey создает ключ для кеширования пульсов пользователя
func UserPulsesCacheKey(userID string, filterHash string) string {
	return fmt.Sprintf("user_pulses:%s:%s", userID, filterHash)
}

// PersonalizedFeedCacheKey создает ключ для кеширования персонализированной ленты
func PersonalizedFeedCacheKey(pulseID string, requestHash string) string {
	return fmt.Sprintf("personalized_feed:%s:%s", pulseID, requestHash)
}

// PulseStatsCacheKey создает ключ для кеширования статистики пульсов
func PulseStatsCacheKey() string {
	return "pulse_stats:all"
}

// DefaultPulseCacheKey создает ключ для кеширования дефолтного пульса пользователя
func DefaultPulseCacheKey(userID string) string {
	return fmt.Sprintf("default_pulse:%s", userID)
}

// PulseCacheKey создает ключ для кеширования конкретного пульса
func PulseCacheKey(pulseID, userID string) string {
	return fmt.Sprintf("pulse:%s:%s", pulseID, userID)
}

// LatestFeedCacheKey создает ключ для кеширования последних новостей ленты
func LatestFeedCacheKey(pulseID string, limit int) string {
	return fmt.Sprintf("latest_feed:%s:%d", pulseID, limit)
}

// TrendingFeedCacheKey создает ключ для кеширования трендовых новостей ленты
func TrendingFeedCacheKey(pulseID string, limit int) string {
	return fmt.Sprintf("trending_feed:%s:%d", pulseID, limit)
}

// InvalidateUserCache удаляет все кешированные данные пользователя
func (c *Cache) InvalidateUserCache(userID string) {
	patterns := []string{
		fmt.Sprintf("user_pulses:%s:*", userID),
		fmt.Sprintf("default_pulse:%s", userID),
		fmt.Sprintf("pulse:*:%s", userID),
	}

	for _, pattern := range patterns {
		c.DeleteByPattern(pattern)
	}
}

// InvalidatePulseCache удаляет все кешированные данные пульса
func (c *Cache) InvalidatePulseCache(pulseID string) {
	patterns := []string{
		fmt.Sprintf("personalized_feed:%s:*", pulseID),
		fmt.Sprintf("latest_feed:%s:*", pulseID),
		fmt.Sprintf("trending_feed:%s:*", pulseID),
		fmt.Sprintf("pulse:%s:*", pulseID),
	}

	for _, pattern := range patterns {
		c.DeleteByPattern(pattern)
	}
}
