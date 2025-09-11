package services

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"notification-service/internal/config"
	"notification-service/internal/models"
)

// EventProcessor представляет процессор событий уведомлений
type EventProcessor struct {
	config              *config.Config
	logger              *logrus.Logger
	notificationService NotificationServiceInterface
	eventQueue          chan *models.NotificationEvent
	workers             []*EventWorker
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	stats               *EventProcessorStats
}

// NotificationServiceInterface интерфейс для избежания циклических зависимостей
type NotificationServiceInterface interface {
	ProcessEvent(event *models.NotificationEvent) error
}

// EventWorker представляет воркер для обработки событий
type EventWorker struct {
	id                  int
	config              *config.Config
	logger              *logrus.Entry
	notificationService NotificationServiceInterface
	eventQueue          chan *models.NotificationEvent
	stopCh              chan struct{}
	stats               *WorkerStats
}

// EventProcessorStats статистика процессора событий
type EventProcessorStats struct {
	mu                  sync.RWMutex
	TotalEvents         int64     `json:"total_events"`
	ProcessedEvents     int64     `json:"processed_events"`
	FailedEvents        int64     `json:"failed_events"`
	QueueSize           int       `json:"queue_size"`
	ActiveWorkers       int       `json:"active_workers"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	LastProcessedAt     time.Time `json:"last_processed_at"`
}

// WorkerStats статистика воркера
type WorkerStats struct {
	mu              sync.RWMutex
	WorkerID        int           `json:"worker_id"`
	ProcessedEvents int64         `json:"processed_events"`
	FailedEvents    int64         `json:"failed_events"`
	LastProcessedAt time.Time     `json:"last_processed_at"`
	IsActive        bool          `json:"is_active"`
}

// NewEventProcessor создает новый процессор событий
func NewEventProcessor(
	config *config.Config,
	logger *logrus.Logger,
	notificationService NotificationServiceInterface,
) *EventProcessor {
	return &EventProcessor{
		config:              config,
		logger:              logger,
		notificationService: notificationService,
		eventQueue:          make(chan *models.NotificationEvent, config.Events.BufferSize),
		stopCh:              make(chan struct{}),
		stats: &EventProcessorStats{
			ActiveWorkers: config.Events.WorkerCount,
		},
	}
}

// Start запускает процессор событий
func (p *EventProcessor) Start() {
	p.logger.WithField("worker_count", p.config.Events.WorkerCount).Info("Starting event processor")
	
	// Создаем и запускаем воркеров
	p.workers = make([]*EventWorker, p.config.Events.WorkerCount)
	
	for i := 0; i < p.config.Events.WorkerCount; i++ {
		worker := &EventWorker{
			id:                  i + 1,
			config:              p.config,
			logger:              p.logger.WithField("worker_id", i+1),
			notificationService: p.notificationService,
			eventQueue:          p.eventQueue,
			stopCh:              p.stopCh,
			stats: &WorkerStats{
				WorkerID: i + 1,
				IsActive: true,
			},
		}
		
		p.workers[i] = worker
		
		p.wg.Add(1)
		go worker.run(&p.wg)
	}
	
	// Запускаем горутину для сбора статистики
	p.wg.Add(1)
	go p.statsCollector(&p.wg)
	
	p.logger.Info("Event processor started successfully")
}

// Stop останавливает процессор событий
func (p *EventProcessor) Stop() {
	p.logger.Info("Stopping event processor")
	
	close(p.stopCh)
	p.wg.Wait()
	
	p.logger.Info("Event processor stopped")
}

// ProcessEvent добавляет событие в очередь обработки
func (p *EventProcessor) ProcessEvent(event *models.NotificationEvent) error {
	select {
	case p.eventQueue <- event:
		p.incrementTotalEvents()
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
		}).Debug("Event queued for processing")
		return nil
		
	default:
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
		}).Error("Event queue is full, dropping event")
		return models.NewAPIError(models.ErrorCodeInternalError, "Event queue is full")
	}
}

// GetStats возвращает статистику процессора
func (p *EventProcessor) GetStats() *EventProcessorStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	
	// Копируем статистику для безопасного возврата
	stats := &EventProcessorStats{
		TotalEvents:        p.stats.TotalEvents,
		ProcessedEvents:    p.stats.ProcessedEvents,
		FailedEvents:       p.stats.FailedEvents,
		QueueSize:          len(p.eventQueue),
		ActiveWorkers:      p.stats.ActiveWorkers,
		AverageProcessTime: p.stats.AverageProcessTime,
		LastProcessedAt:    p.stats.LastProcessedAt,
	}
	
	return stats
}

// GetWorkerStats возвращает статистику воркеров
func (p *EventProcessor) GetWorkerStats() []*WorkerStats {
	var workerStats []*WorkerStats
	
	for _, worker := range p.workers {
		worker.stats.mu.RLock()
		stats := &WorkerStats{
			WorkerID:        worker.stats.WorkerID,
			ProcessedEvents: worker.stats.ProcessedEvents,
			FailedEvents:    worker.stats.FailedEvents,
			LastProcessedAt: worker.stats.LastProcessedAt,
			IsActive:        worker.stats.IsActive,
		}
		worker.stats.mu.RUnlock()
		
		workerStats = append(workerStats, stats)
	}
	
	return workerStats
}

// incrementTotalEvents увеличивает счетчик общих событий
func (p *EventProcessor) incrementTotalEvents() {
	p.stats.mu.Lock()
	p.stats.TotalEvents++
	p.stats.mu.Unlock()
}

// incrementProcessedEvents увеличивает счетчик обработанных событий
func (p *EventProcessor) incrementProcessedEvents() {
	p.stats.mu.Lock()
	p.stats.ProcessedEvents++
	p.stats.LastProcessedAt = time.Now()
	p.stats.mu.Unlock()
}

// incrementFailedEvents увеличивает счетчик неудачных событий
func (p *EventProcessor) incrementFailedEvents() {
	p.stats.mu.Lock()
	p.stats.FailedEvents++
	p.stats.mu.Unlock()
}

// statsCollector собирает статистику
func (p *EventProcessor) statsCollector(wg *sync.WaitGroup) {
	defer wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.logStats()
		}
	}
}

// logStats логирует статистику
func (p *EventProcessor) logStats() {
	stats := p.GetStats()
	
	p.logger.WithFields(logrus.Fields{
		"total_events":     stats.TotalEvents,
		"processed_events": stats.ProcessedEvents,
		"failed_events":    stats.FailedEvents,
		"queue_size":       stats.QueueSize,
		"active_workers":   stats.ActiveWorkers,
	}).Info("Event processor statistics")
}

// Методы EventWorker

// run запускает воркер
func (w *EventWorker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	
	w.logger.Info("Event worker started")
	defer w.logger.Info("Event worker stopped")
	
	for {
		select {
		case <-w.stopCh:
			w.setInactive()
			return
			
		case event := <-w.eventQueue:
			w.processEvent(event)
		}
	}
}

// processEvent обрабатывает событие
func (w *EventWorker) processEvent(event *models.NotificationEvent) {
	start := time.Now()
	
	w.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Debug("Processing event")
	
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), w.config.Events.ProcessingTimeout)
	defer cancel()
	
	// Обрабатываем событие с retry логикой
	err := w.processEventWithRetry(ctx, event)
	
	duration := time.Since(start)
	
	if err != nil {
		w.incrementFailedEvents()
		w.logger.WithError(err).WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"duration":   duration,
		}).Error("Failed to process event")
	} else {
		w.incrementProcessedEvents()
		w.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"duration":   duration,
		}).Debug("Event processed successfully")
	}
}

// processEventWithRetry обрабатывает событие с повторными попытками
func (w *EventWorker) processEventWithRetry(ctx context.Context, event *models.NotificationEvent) error {
	var lastErr error
	
	maxAttempts := 1
	if w.config.Events.Retry.Enabled {
		maxAttempts = w.config.Events.Retry.MaxAttempts
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		err := w.notificationService.ProcessEvent(event)
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		if attempt < maxAttempts {
			// Вычисляем задержку для следующей попытки
			delay := w.calculateRetryDelay(attempt)
			
			w.logger.WithFields(logrus.Fields{
				"event_id": event.ID,
				"attempt":  attempt,
				"delay":    delay,
				"error":    err.Error(),
			}).Warn("Event processing failed, retrying")
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
	}
	
	return lastErr
}

// calculateRetryDelay вычисляет задержку для повторной попытки
func (w *EventWorker) calculateRetryDelay(attempt int) time.Duration {
	if !w.config.Events.Retry.Enabled {
		return 0
	}
	
	delay := w.config.Events.Retry.InitialDelay
	
	// Экспоненциальная задержка
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * w.config.Events.Retry.Multiplier)
	}
	
	// Ограничиваем максимальной задержкой
	if delay > w.config.Events.Retry.MaxDelay {
		delay = w.config.Events.Retry.MaxDelay
	}
	
	return delay
}

// incrementProcessedEvents увеличивает счетчик обработанных событий воркера
func (w *EventWorker) incrementProcessedEvents() {
	w.stats.mu.Lock()
	w.stats.ProcessedEvents++
	w.stats.LastProcessedAt = time.Now()
	w.stats.mu.Unlock()
}

// incrementFailedEvents увеличивает счетчик неудачных событий воркера
func (w *EventWorker) incrementFailedEvents() {
	w.stats.mu.Lock()
	w.stats.FailedEvents++
	w.stats.mu.Unlock()
}

// setInactive помечает воркер как неактивный
func (w *EventWorker) setInactive() {
	w.stats.mu.Lock()
	w.stats.IsActive = false
	w.stats.mu.Unlock()
}
