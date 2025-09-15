package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"pulse-service/internal/database"
	"pulse-service/internal/models"
)

// PulseRepository представляет репозиторий для работы с пульсами пользователей
type PulseRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewPulseRepository создает новый репозиторий пульсов
func NewPulseRepository(db *database.DB, logger *logrus.Logger) *PulseRepository {
	return &PulseRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новый пульс пользователя
func (r *PulseRepository) Create(ctx context.Context, userID string, req models.PulseRequest) (*models.UserPulse, error) {
	var pulse models.UserPulse

	err := r.db.Transaction(func(tx *sql.Tx) error {
		// Проверяем лимит пульсов на пользователя
		var pulseCount int
		countQuery := `SELECT COUNT(*) FROM user_pulses WHERE user_id = $1 AND is_active = true`
		if err := tx.QueryRowContext(ctx, countQuery, userID).Scan(&pulseCount); err != nil {
			return fmt.Errorf("failed to count user pulses: %w", err)
		}

		// Здесь должна быть проверка лимита из конфигурации
		// if pulseCount >= maxPulsesPerUser { return error }

		// Если это первый пульс и не указан isDefault, делаем его дефолтным
		isDefault := false
		if req.IsDefault != nil {
			isDefault = *req.IsDefault
		} else if pulseCount == 0 {
			isDefault = true
		}

		// Если делаем пульс дефолтным, убираем дефолт с других
		if isDefault {
			updateQuery := `UPDATE user_pulses SET is_default = false WHERE user_id = $1 AND is_default = true`
			if _, err := tx.ExecContext(ctx, updateQuery, userID); err != nil {
				return fmt.Errorf("failed to update existing default pulses: %w", err)
			}
		}

		// Создаем пульс
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		insertQuery := `
			INSERT INTO user_pulses (user_id, name, description, keywords, refresh_interval_min, is_active, is_default)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, user_id, name, description, keywords, refresh_interval_min, is_active, is_default, created_at, updated_at`

		err := tx.QueryRowContext(ctx, insertQuery,
			userID, req.Name, req.Description, req.Keywords, req.RefreshIntervalMin, isActive, isDefault,
		).Scan(
			&pulse.ID, &pulse.UserID, &pulse.Name, &pulse.Description, &pulse.Keywords,
			&pulse.RefreshIntervalMin, &pulse.IsActive, &pulse.IsDefault,
			&pulse.CreatedAt, &pulse.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create pulse: %w", err)
		}

		// Добавляем источники
		if len(req.SourceIDs) > 0 {
			if err := r.addPulseSources(ctx, tx, pulse.ID, req.SourceIDs); err != nil {
				return fmt.Errorf("failed to add pulse sources: %w", err)
			}
		}

		// Добавляем категории
		if len(req.CategoryIDs) > 0 {
			if err := r.addPulseCategories(ctx, tx, pulse.ID, req.CategoryIDs); err != nil {
				return fmt.Errorf("failed to add pulse categories: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Загружаем полную информацию о созданном пульсе
	return r.GetByID(ctx, pulse.ID, userID)
}

// GetByID возвращает пульс по ID
func (r *PulseRepository) GetByID(ctx context.Context, pulseID, userID string) (*models.UserPulse, error) {
	query := `
		SELECT id, user_id, name, description, refresh_interval_min, is_active, is_default,
			   created_at, updated_at, last_refreshed_at
		FROM user_pulses
		WHERE id = $1 AND user_id = $2`

	var pulse models.UserPulse
	err := r.db.QueryRowContext(ctx, query, pulseID, userID).Scan(
		&pulse.ID, &pulse.UserID, &pulse.Name, &pulse.Description,
		&pulse.RefreshIntervalMin, &pulse.IsActive, &pulse.IsDefault,
		&pulse.CreatedAt, &pulse.UpdatedAt, &pulse.LastRefreshedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pulse with id %d not found for user %d", pulseID, userID)
		}
		return nil, fmt.Errorf("failed to get pulse: %w", err)
	}

	// Загружаем источники
	sources, err := r.getPulseSources(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse sources")
	} else {
		pulse.Sources = sources
	}

	// Загружаем категории
	categories, err := r.getPulseCategories(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse categories")
	} else {
		pulse.Categories = categories
	}

	// Подсчитываем количество новостей
	newsCount, err := r.getPulseNewsCount(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse news count")
	} else {
		pulse.NewsCount = newsCount
	}

	return &pulse, nil
}

// GetByUserID возвращает все пульсы пользователя
func (r *PulseRepository) GetByUserID(ctx context.Context, userID string, filter models.PulseFilter) ([]models.UserPulse, error) {
	query, args := r.buildUserPulsesQuery(userID, filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user pulses: %w", err)
	}
	defer rows.Close()

	var pulses []models.UserPulse
	for rows.Next() {
		var pulse models.UserPulse
		err := rows.Scan(
			&pulse.ID, &pulse.UserID, &pulse.Name, &pulse.Description, &pulse.Keywords,
			&pulse.RefreshIntervalMin, &pulse.IsActive, &pulse.IsDefault,
			&pulse.CreatedAt, &pulse.UpdatedAt, &pulse.LastRefreshedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pulse: %w", err)
		}

		// Загружаем источники и категории для каждого пульса
		sources, err := r.getPulseSources(ctx, pulse.ID)
		if err != nil {
			r.logger.WithError(err).WithField("pulse_id", pulse.ID).Warn("Failed to get pulse sources")
		} else {
			pulse.Sources = sources
		}

		categories, err := r.getPulseCategories(ctx, pulse.ID)
		if err != nil {
			r.logger.WithError(err).WithField("pulse_id", pulse.ID).Warn("Failed to get pulse categories")
		} else {
			pulse.Categories = categories
		}

		newsCount, err := r.getPulseNewsCount(ctx, pulse.ID)
		if err != nil {
			r.logger.WithError(err).WithField("pulse_id", pulse.ID).Warn("Failed to get pulse news count")
		} else {
			pulse.NewsCount = newsCount
		}

		pulses = append(pulses, pulse)
	}

	return pulses, rows.Err()
}

// GetDefaultByUserID возвращает дефолтный пульс пользователя
func (r *PulseRepository) GetDefaultByUserID(ctx context.Context, userID string) (*models.UserPulse, error) {
	query := `
		SELECT id, user_id, name, description, refresh_interval_min, is_active, is_default,
			   created_at, updated_at, last_refreshed_at
		FROM user_pulses
		WHERE user_id = $1 AND is_default = true AND is_active = true
		LIMIT 1`

	var pulse models.UserPulse
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&pulse.ID, &pulse.UserID, &pulse.Name, &pulse.Description,
		&pulse.RefreshIntervalMin, &pulse.IsActive, &pulse.IsDefault,
		&pulse.CreatedAt, &pulse.UpdatedAt, &pulse.LastRefreshedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no default pulse found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get default pulse: %w", err)
	}

	// Загружаем связанные данные
	sources, err := r.getPulseSources(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse sources")
	} else {
		pulse.Sources = sources
	}

	categories, err := r.getPulseCategories(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse categories")
	} else {
		pulse.Categories = categories
	}

	newsCount, err := r.getPulseNewsCount(ctx, pulse.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get pulse news count")
	} else {
		pulse.NewsCount = newsCount
	}

	return &pulse, nil
}

// Update обновляет пульс пользователя
func (r *PulseRepository) Update(ctx context.Context, pulseID, userID string, req models.PulseRequest) (*models.UserPulse, error) {
	err := r.db.Transaction(func(tx *sql.Tx) error {
		// Проверяем, что пульс принадлежит пользователю
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_pulses WHERE id = $1 AND user_id = $2)`
		if err := tx.QueryRowContext(ctx, checkQuery, pulseID, userID).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check pulse ownership: %w", err)
		}
		if !exists {
			return fmt.Errorf("pulse not found or access denied")
		}

		// Если делаем пульс дефолтным, убираем дефолт с других
		if req.IsDefault != nil && *req.IsDefault {
			updateQuery := `UPDATE user_pulses SET is_default = false WHERE user_id = $1 AND id != $2 AND is_default = true`
			if _, err := tx.ExecContext(ctx, updateQuery, userID, pulseID); err != nil {
				return fmt.Errorf("failed to update existing default pulses: %w", err)
			}
		}

		// Обновляем основные данные пульса
		updateQuery := `
			UPDATE user_pulses 
			SET name = $1, description = $2, refresh_interval_min = $3, updated_at = CURRENT_TIMESTAMP`
		args := []interface{}{req.Name, req.Description, req.RefreshIntervalMin}
		argIndex := 4

		if req.IsActive != nil {
			updateQuery += fmt.Sprintf(", is_active = $%d", argIndex)
			args = append(args, *req.IsActive)
			argIndex++
		}

		if req.IsDefault != nil {
			updateQuery += fmt.Sprintf(", is_default = $%d", argIndex)
			args = append(args, *req.IsDefault)
			argIndex++
		}

		updateQuery += fmt.Sprintf(" WHERE id = $%d AND user_id = $%d", argIndex, argIndex+1)
		args = append(args, pulseID, userID)

		if _, err := tx.ExecContext(ctx, updateQuery, args...); err != nil {
			return fmt.Errorf("failed to update pulse: %w", err)
		}

		// Обновляем источники
		if err := r.updatePulseSources(ctx, tx, pulseID, req.SourceIDs); err != nil {
			return fmt.Errorf("failed to update pulse sources: %w", err)
		}

		// Обновляем категории
		if err := r.updatePulseCategories(ctx, tx, pulseID, req.CategoryIDs); err != nil {
			return fmt.Errorf("failed to update pulse categories: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Возвращаем обновленный пульс
	return r.GetByID(ctx, pulseID, userID)
}

// Delete удаляет пульс пользователя (мягкое удаление)
func (r *PulseRepository) Delete(ctx context.Context, pulseID, userID string) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		// Проверяем, что пульс принадлежит пользователю
		var exists bool
		var isDefault bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_pulses WHERE id = $1 AND user_id = $2), 
		               COALESCE((SELECT is_default FROM user_pulses WHERE id = $1 AND user_id = $2), false)`
		if err := tx.QueryRowContext(ctx, checkQuery, pulseID, userID).Scan(&exists, &isDefault); err != nil {
			return fmt.Errorf("failed to check pulse: %w", err)
		}
		if !exists {
			return fmt.Errorf("pulse not found or access denied")
		}

		// Мягкое удаление
		deleteQuery := `UPDATE user_pulses SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND user_id = $2`
		if _, err := tx.ExecContext(ctx, deleteQuery, pulseID, userID); err != nil {
			return fmt.Errorf("failed to delete pulse: %w", err)
		}

		// Если удаляем дефолтный пульс, нужно назначить новый дефолтный
		if isDefault {
			setNewDefaultQuery := `
				UPDATE user_pulses 
				SET is_default = true, updated_at = CURRENT_TIMESTAMP 
				WHERE id = (
					SELECT id FROM user_pulses 
					WHERE user_id = $1 AND is_active = true AND id != $2
					ORDER BY created_at ASC 
					LIMIT 1
				)`
			if _, err := tx.ExecContext(ctx, setNewDefaultQuery, userID, pulseID); err != nil {
				r.logger.WithError(err).Warn("Failed to set new default pulse")
			}
		}

		return nil
	})
}

// UpdateLastRefreshed обновляет время последнего обновления пульса
func (r *PulseRepository) UpdateLastRefreshed(ctx context.Context, pulseID string) error {
	query := `UPDATE user_pulses SET last_refreshed_at = CURRENT_TIMESTAMP WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, pulseID)
	if err != nil {
		return fmt.Errorf("failed to update last refreshed time: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pulse with id %d not found", pulseID)
	}

	return nil
}

// Вспомогательные методы

// buildUserPulsesQuery строит запрос для получения пульсов пользователя
func (r *PulseRepository) buildUserPulsesQuery(userID string, filter models.PulseFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT id, user_id, name, description, keywords, refresh_interval_min, is_active, is_default,
			   created_at, updated_at, last_refreshed_at
		FROM user_pulses`

	// Базовое условие - пульсы пользователя
	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Фильтр по активности - по умолчанию показываем только активные пульсы
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	} else {
		// По умолчанию показываем только активные пульсы
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, true)
		argIndex++
	}

	// Фильтр по дефолтности
	if filter.IsDefault != nil {
		conditions = append(conditions, fmt.Sprintf("is_default = $%d", argIndex))
		args = append(args, *filter.IsDefault)
		argIndex++
	}

	// Поиск по ключевым словам
	if filter.Keywords != "" {
		conditions = append(conditions,
			fmt.Sprintf("to_tsvector('russian', name || ' ' || COALESCE(description, '')) @@ plainto_tsquery('russian', $%d)", argIndex))
		args = append(args, filter.Keywords)
		argIndex++
	}

	// Фильтр по дате создания от
	if filter.CreatedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedFrom)
		argIndex++
	}

	// Фильтр по дате создания до
	if filter.CreatedTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedTo)
		argIndex++
	}

	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Сортировка
	sortBy, sortOrder := models.NormalizePulseSortParams(filter.SortBy, filter.SortOrder)
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, strings.ToUpper(sortOrder))

	// Лимит и оффсет
	if filter.PageSize > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, filter.PageSize, filter.GetOffset())
	}

	return baseQuery, args
}

// getPulseSources получает источники пульса
func (r *PulseRepository) getPulseSources(ctx context.Context, pulseID string) ([]models.PulseSource, error) {
	query := `
		SELECT ps.pulse_id, ps.source_id, ns.name, ns.domain, ns.logo_url, c.id as country_id, c.name as country_name
		FROM pulse_sources ps
		JOIN news_sources ns ON ps.source_id = ns.id
		LEFT JOIN countries c ON ns.country_id = c.id
		WHERE ps.pulse_id = $1 AND ns.is_active = true
		ORDER BY ns.name`

	rows, err := r.db.QueryContext(ctx, query, pulseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pulse sources: %w", err)
	}
	defer rows.Close()

	var sources []models.PulseSource
	for rows.Next() {
		var source models.PulseSource
		err := rows.Scan(
			&source.PulseID, &source.SourceID,
			&source.SourceName, &source.SourceDomain, &source.SourceLogoURL,
			&source.CountryID, &source.CountryName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pulse source: %w", err)
		}
		sources = append(sources, source)
	}

	return sources, rows.Err()
}

// getPulseCategories получает категории пульса
func (r *PulseRepository) getPulseCategories(ctx context.Context, pulseID string) ([]models.PulseCategory, error) {
	query := `
		SELECT pc.pulse_id, pc.category_id, c.name, c.slug, c.color, c.icon
		FROM pulse_categories pc
		JOIN categories c ON pc.category_id = c.id
		WHERE pc.pulse_id = $1 AND c.is_active = true
		ORDER BY c.name`

	rows, err := r.db.QueryContext(ctx, query, pulseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pulse categories: %w", err)
	}
	defer rows.Close()

	var categories []models.PulseCategory
	for rows.Next() {
		var category models.PulseCategory
		err := rows.Scan(
			&category.PulseID, &category.CategoryID,
			&category.CategoryName, &category.CategorySlug,
			&category.CategoryColor, &category.CategoryIcon,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pulse category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

// getPulseNewsCount подсчитывает количество новостей в пульсе
func (r *PulseRepository) getPulseNewsCount(ctx context.Context, pulseID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM pulse_news pn
		JOIN news n ON n.id = pn.news_id
		WHERE pn.pulse_id = $1::uuid
		AND n.is_active = true`

	var count int
	err := r.db.QueryRowContext(ctx, query, pulseID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pulse news: %w", err)
	}

	return count, nil
}

// addPulseSources добавляет источники к пульсу
func (r *PulseRepository) addPulseSources(ctx context.Context, tx *sql.Tx, pulseID string, sourceIDs []int) error {
	if len(sourceIDs) == 0 {
		return nil
	}

	// Создаем bulk insert
	values := make([]string, len(sourceIDs))
	args := make([]interface{}, 0, len(sourceIDs)*2)

	for i, sourceID := range sourceIDs {
		values[i] = fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, pulseID, sourceID)
	}

	query := fmt.Sprintf("INSERT INTO pulse_sources (pulse_id, source_id) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(values, ", "))

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert pulse sources: %w", err)
	}

	return nil
}

// addPulseCategories добавляет категории к пульсу
func (r *PulseRepository) addPulseCategories(ctx context.Context, tx *sql.Tx, pulseID string, categoryIDs []int) error {
	if len(categoryIDs) == 0 {
		return nil
	}

	// Создаем bulk insert
	values := make([]string, len(categoryIDs))
	args := make([]interface{}, 0, len(categoryIDs)*2)

	for i, categoryID := range categoryIDs {
		values[i] = fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, pulseID, categoryID)
	}

	query := fmt.Sprintf("INSERT INTO pulse_categories (pulse_id, category_id) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(values, ", "))

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert pulse categories: %w", err)
	}

	return nil
}

// updatePulseSources обновляет источники пульса
func (r *PulseRepository) updatePulseSources(ctx context.Context, tx *sql.Tx, pulseID string, sourceIDs []int) error {
	// Удаляем старые источники
	deleteQuery := `DELETE FROM pulse_sources WHERE pulse_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQuery, pulseID); err != nil {
		return fmt.Errorf("failed to delete old pulse sources: %w", err)
	}

	// Добавляем новые источники
	return r.addPulseSources(ctx, tx, pulseID, sourceIDs)
}

// updatePulseCategories обновляет категории пульса
func (r *PulseRepository) updatePulseCategories(ctx context.Context, tx *sql.Tx, pulseID string, categoryIDs []int) error {
	// Удаляем старые категории
	deleteQuery := `DELETE FROM pulse_categories WHERE pulse_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQuery, pulseID); err != nil {
		return fmt.Errorf("failed to delete old pulse categories: %w", err)
	}

	// Добавляем новые категории
	return r.addPulseCategories(ctx, tx, pulseID, categoryIDs)
}

// GetPulseSources возвращает источники пульса (публичный метод)
func (r *PulseRepository) GetPulseSources(ctx context.Context, pulseID string) ([]models.PulseSource, error) {
	return r.getPulseSources(ctx, pulseID)
}

// GetPulseCategories возвращает категории пульса (публичный метод)
func (r *PulseRepository) GetPulseCategories(ctx context.Context, pulseID string) ([]models.PulseCategory, error) {
	return r.getPulseCategories(ctx, pulseID)
}

// PulseExists проверяет существование пульса по ID
func (r *PulseRepository) PulseExists(ctx context.Context, pulseID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_pulses WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, pulseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pulse existence: %w", err)
	}
	return exists, nil
}

// GetByIDWithoutUser получает пульс по ID без проверки пользователя
func (r *PulseRepository) GetByIDWithoutUser(ctx context.Context, pulseID string) (*models.UserPulse, error) {
	query := `
		SELECT id, user_id, name, description, keywords, refresh_interval_min, is_active, is_default,
			   created_at, updated_at, last_refreshed_at
		FROM user_pulses
		WHERE id = $1`

	var pulse models.UserPulse
	err := r.db.QueryRowContext(ctx, query, pulseID).Scan(
		&pulse.ID, &pulse.UserID, &pulse.Name, &pulse.Description, &pulse.Keywords,
		&pulse.RefreshIntervalMin, &pulse.IsActive, &pulse.IsDefault,
		&pulse.CreatedAt, &pulse.UpdatedAt, &pulse.LastRefreshedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pulse with id %s not found", pulseID)
		}
		return nil, fmt.Errorf("failed to get pulse: %w", err)
	}

	return &pulse, nil
}
