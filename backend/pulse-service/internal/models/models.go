package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// UserPulse представляет пульс пользователя
type UserPulse struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	Keywords           string    `json:"keywords" db:"keywords"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	IsDefault          bool      `json:"is_default" db:"is_default"`
	NewsCount          int       `json:"news_count" db:"news_count"`
	RefreshIntervalMin int       `json:"refresh_interval_min" db:"refresh_interval_min"`
	LastRefreshedAt    time.Time `json:"last_refreshed_at" db:"last_refreshed_at"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	LastUpdatedAt      time.Time `json:"last_updated_at" db:"last_updated_at"`

	// Связанные данные (не хранятся в БД)
	Sources    []PulseSource   `json:"sources,omitempty" db:"-"`
	Categories []PulseCategory `json:"categories,omitempty" db:"-"`
}

// PulseSource представляет источник новостей в пульсе
type PulseSource struct {
	PulseID  string `json:"pulse_id" db:"pulse_id"`
	SourceID int    `json:"source_id" db:"source_id"`

	// Связанные данные
	SourceName    string `json:"source_name,omitempty" db:"source_name"`
	SourceDomain  string `json:"source_domain,omitempty" db:"source_domain"`
	SourceLogoURL string `json:"source_logo_url,omitempty" db:"source_logo_url"`
	CountryID     int    `json:"country_id,omitempty" db:"country_id"`
	CountryName   string `json:"country_name,omitempty" db:"country_name"`
}

// PulseCategory представляет категорию новостей в пульсе
type PulseCategory struct {
	PulseID    string `json:"pulse_id" db:"pulse_id"`
	CategoryID int    `json:"category_id" db:"category_id"`

	// Связанные данные
	CategoryName  string `json:"category_name,omitempty" db:"category_name"`
	CategorySlug  string `json:"category_slug,omitempty" db:"category_slug"`
	CategoryColor string `json:"category_color,omitempty" db:"category_color"`
	CategoryIcon  string `json:"category_icon,omitempty" db:"category_icon"`
}

// PersonalizedFeed представляет персонализированную ленту новостей
type PersonalizedFeed struct {
	PulseID     string                `json:"pulse_id"`
	PulseName   string                `json:"pulse_name"`
	News        []PersonalizedNews    `json:"news"`
	Pagination  Pagination            `json:"pagination"`
	GeneratedAt time.Time             `json:"generated_at"`
	Stats       PersonalizedFeedStats `json:"stats"`
}

// PersonalizedNews представляет новость в персонализированной ленте
type PersonalizedNews struct {
	ID             int       `json:"id" db:"id"`
	Title          string    `json:"title" db:"title"`
	Description    string    `json:"description" db:"description"`
	Content        string    `json:"content" db:"content"`
	URL            string    `json:"url" db:"url"`
	ImageURL       string    `json:"image_url" db:"image_url"`
	Author         string    `json:"author" db:"author"`
	SourceID       int       `json:"source_id" db:"source_id"`
	CategoryID     *int      `json:"category_id" db:"category_id"`
	PublishedAt    time.Time `json:"published_at" db:"published_at"`
	RelevanceScore float64   `json:"relevance_score" db:"relevance_score"`
	ViewCount      int       `json:"view_count" db:"view_count"`

	// Связанные данные
	SourceName    string `json:"source_name" db:"source_name"`
	SourceDomain  string `json:"source_domain" db:"source_domain"`
	SourceLogoURL string `json:"source_logo_url" db:"source_logo_url"`
	CategoryName  string `json:"category_name,omitempty" db:"category_name"`
	CategorySlug  string `json:"category_slug,omitempty" db:"category_slug"`
	CategoryColor string `json:"category_color,omitempty" db:"category_color"`
	CategoryIcon  string `json:"category_icon,omitempty" db:"category_icon"`

	// Теги новости
	Tags []string `json:"tags,omitempty" db:"-"`

	// Мета-информация для персонализации
	MatchReason   string  `json:"match_reason,omitempty" db:"-"`
	PersonalScore float64 `json:"personal_score,omitempty" db:"-"`
}

// PersonalizedFeedStats представляет статистику персонализированной ленты
type PersonalizedFeedStats struct {
	TotalNews         int                 `json:"total_news"`
	NewsSources       int                 `json:"news_sources"`
	NewsCategories    int                 `json:"news_categories"`
	AverageScore      float64             `json:"average_score"`
	SourceBreakdown   []SourceBreakdown   `json:"source_breakdown"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
}

// SourceBreakdown представляет разбивку по источникам
type SourceBreakdown struct {
	SourceID   int     `json:"source_id"`
	SourceName string  `json:"source_name"`
	NewsCount  int     `json:"news_count"`
	Percentage float64 `json:"percentage"`
}

// CategoryBreakdown представляет разбивку по категориям
type CategoryBreakdown struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	NewsCount    int     `json:"news_count"`
	Percentage   float64 `json:"percentage"`
}

// PulseFilter представляет фильтры для поиска пульсов
type PulseFilter struct {
	UserID      *string    `json:"user_id,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	IsDefault   *bool      `json:"is_default,omitempty"`
	Keywords    string     `json:"keywords,omitempty"`
	CreatedFrom *time.Time `json:"created_from,omitempty"`
	CreatedTo   *time.Time `json:"created_to,omitempty"`
	Page        int        `json:"page,omitempty"`
	PageSize    int        `json:"page_size,omitempty"`
	SortBy      string     `json:"sort_by,omitempty"`    // created_at, updated_at, name
	SortOrder   string     `json:"sort_order,omitempty"` // asc, desc
}

// FeedRequest представляет запрос на получение персонализированной ленты
type FeedRequest struct {
	PulseID   string     `json:"pulse_id"`
	Page      int        `json:"page,omitempty"`
	PageSize  int        `json:"page_size,omitempty"`
	DateFrom  *time.Time `json:"date_from,omitempty"`
	DateTo    *time.Time `json:"date_to,omitempty"`
	MinScore  *float64   `json:"min_score,omitempty"`
	SortBy    string     `json:"sort_by,omitempty"`    // published_at, relevance_score, personal_score
	SortOrder string     `json:"sort_order,omitempty"` // asc, desc
}

// PulseRequest представляет запрос на создание/обновление пульса
type PulseRequest struct {
	Name               string `json:"name" validate:"required,min=1,max=100"`
	Description        string `json:"description" validate:"max=500"`
	Keywords           string `json:"keywords" validate:"max=500"`
	RefreshIntervalMin int    `json:"refresh_interval_min" validate:"min=5,max=1440"`
	SourceIDs          []int  `json:"source_ids" validate:"required,min=1,max=50"`
	CategoryIDs        []int  `json:"category_ids" validate:"max=10"`
	IsActive           *bool  `json:"is_active,omitempty"`
	IsDefault          *bool  `json:"is_default,omitempty"`
}

// Pagination представляет информацию о пагинации
type Pagination struct {
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// PulseStats представляет статистику пульсов
type PulseStats struct {
	TotalPulses           int                  `json:"total_pulses"`
	ActivePulses          int                  `json:"active_pulses"`
	DefaultPulses         int                  `json:"default_pulses"`
	AvgSourcesPerPulse    float64              `json:"avg_sources_per_pulse"`
	AvgCategoriesPerPulse float64              `json:"avg_categories_per_pulse"`
	TopSources            []PulseSourceStats   `json:"top_sources"`
	TopCategories         []PulseCategoryStats `json:"top_categories"`
	RecentPulses          []UserPulse          `json:"recent_pulses"`
}

// PulseSourceStats представляет статистику источника в пульсах
type PulseSourceStats struct {
	SourceID   int     `json:"source_id"`
	SourceName string  `json:"source_name"`
	PulseCount int     `json:"pulse_count"`
	Percentage float64 `json:"percentage"`
}

// PulseCategoryStats представляет статистику категории в пульсах
type PulseCategoryStats struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	PulseCount   int     `json:"pulse_count"`
	Percentage   float64 `json:"percentage"`
}

// IntArray представляет массив целых чисел для работы с PostgreSQL
type IntArray []int

// Scan реализует интерфейс sql.Scanner для IntArray
func (a *IntArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into IntArray", value)
	}
}

// Value реализует интерфейс driver.Valuer для IntArray
func (a IntArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Validation методы

// IsValid проверяет валидность пульса
func (p *UserPulse) IsValid() bool {
	return len(p.Name) >= 1 && len(p.Name) <= 100 &&
		len(p.Description) <= 500 &&
		p.RefreshIntervalMin >= 5 && p.RefreshIntervalMin <= 1440 &&
		p.UserID != ""
}

// NeedsRefresh проверяет, нужно ли обновить пульс
func (p *UserPulse) NeedsRefresh() bool {
	if p.LastRefreshedAt.IsZero() {
		return true
	}

	refreshInterval := time.Duration(p.RefreshIntervalMin) * time.Minute
	return time.Since(p.LastRefreshedAt) >= refreshInterval
}

// GetRefreshInterval возвращает интервал обновления как Duration
func (p *UserPulse) GetRefreshInterval() time.Duration {
	return time.Duration(p.RefreshIntervalMin) * time.Minute
}

// Validate проверяет корректность запроса на создание пульса
func (r *PulseRequest) Validate() error {
	if len(r.Name) == 0 || len(r.Name) > 100 {
		return fmt.Errorf("pulse name must be 1-100 characters")
	}

	if len(r.Description) > 500 {
		return fmt.Errorf("pulse description must be max 500 characters")
	}

	if r.RefreshIntervalMin < 5 || r.RefreshIntervalMin > 1440 {
		return fmt.Errorf("refresh interval must be 5-1440 minutes")
	}

	if len(r.SourceIDs) == 0 {
		return fmt.Errorf("at least one source is required")
	}

	if len(r.SourceIDs) > 50 {
		return fmt.Errorf("maximum 50 sources allowed per pulse")
	}

	if len(r.CategoryIDs) > 10 {
		return fmt.Errorf("maximum 10 categories allowed per pulse")
	}

	return nil
}

// Validate проверяет корректность фильтра пульсов
func (f *PulseFilter) Validate(maxPageSize int) error {
	if f.Page < 0 {
		f.Page = 1
	}
	if f.Page == 0 {
		f.Page = 1
	}

	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > maxPageSize {
		f.PageSize = maxPageSize
	}

	if f.Keywords != "" && len(f.Keywords) > 100 {
		return fmt.Errorf("keywords too long")
	}

	if f.CreatedFrom != nil && f.CreatedTo != nil && f.CreatedFrom.After(*f.CreatedTo) {
		return fmt.Errorf("created_from cannot be after created_to")
	}

	return nil
}

// GetOffset возвращает offset для SQL запроса
func (f *PulseFilter) GetOffset() int {
	return (f.Page - 1) * f.PageSize
}

// Validate проверяет корректность запроса на получение ленты
func (r *FeedRequest) Validate(maxPageSize, maxNewsPerFeed int) error {
	if r.PulseID == "" {
		return fmt.Errorf("pulse_id is required")
	}

	if r.Page <= 0 {
		r.Page = 1
	}

	if r.PageSize <= 0 {
		r.PageSize = 20
	}
	if r.PageSize > maxPageSize {
		r.PageSize = maxPageSize
	}
	if r.PageSize > maxNewsPerFeed {
		r.PageSize = maxNewsPerFeed
	}

	if r.DateFrom != nil && r.DateTo != nil && r.DateFrom.After(*r.DateTo) {
		return fmt.Errorf("date_from cannot be after date_to")
	}

	if r.MinScore != nil && (*r.MinScore < 0 || *r.MinScore > 1) {
		return fmt.Errorf("min_score must be between 0 and 1")
	}

	return nil
}

// GetOffset возвращает offset для SQL запроса
func (r *FeedRequest) GetOffset() int {
	return (r.Page - 1) * r.PageSize
}

// NewPagination создает новую пагинацию
func NewPagination(page, pageSize, total int) Pagination {
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// Константы для сортировки пульсов
const (
	PulseSortByCreatedAt = "created_at"
	PulseSortByUpdatedAt = "updated_at"
	PulseSortByName      = "name"

	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// GetValidPulseSortFields возвращает список допустимых полей для сортировки пульсов
func GetValidPulseSortFields() []string {
	return []string{
		PulseSortByCreatedAt,
		PulseSortByUpdatedAt,
		PulseSortByName,
	}
}

// IsValidPulseSortField проверяет, является ли поле валидным для сортировки пульсов
func IsValidPulseSortField(field string) bool {
	validFields := GetValidPulseSortFields()
	for _, validField := range validFields {
		if field == validField {
			return true
		}
	}
	return false
}

// IsValidSortOrder проверяет, является ли порядок сортировки валидным
func IsValidSortOrder(order string) bool {
	return order == SortOrderAsc || order == SortOrderDesc
}

// NormalizePulseSortParams нормализует параметры сортировки для пульсов
func NormalizePulseSortParams(sortBy, sortOrder string) (string, string) {
	if !IsValidPulseSortField(sortBy) {
		sortBy = PulseSortByCreatedAt
	}

	if !IsValidSortOrder(sortOrder) {
		sortOrder = SortOrderDesc
	}

	return sortBy, sortOrder
}

// Константы для сортировки персонализированной ленты
const (
	FeedSortByPublishedAt    = "published_at"
	FeedSortByRelevanceScore = "relevance_score"
	FeedSortByPersonalScore  = "personal_score"
)

// GetValidFeedSortFields возвращает список допустимых полей для сортировки ленты
func GetValidFeedSortFields() []string {
	return []string{
		FeedSortByPublishedAt,
		FeedSortByRelevanceScore,
		FeedSortByPersonalScore,
	}
}

// IsValidFeedSortField проверяет, является ли поле валидным для сортировки ленты
func IsValidFeedSortField(field string) bool {
	validFields := GetValidFeedSortFields()
	for _, validField := range validFields {
		if field == validField {
			return true
		}
	}
	return false
}

// NormalizeFeedSortParams нормализует параметры сортировки для ленты
func NormalizeFeedSortParams(sortBy, sortOrder string) (string, string) {
	if !IsValidFeedSortField(sortBy) {
		sortBy = FeedSortByPublishedAt
	}

	if !IsValidSortOrder(sortOrder) {
		sortOrder = SortOrderDesc
	}

	return sortBy, sortOrder
}

// CalculatePersonalScore вычисляет персональный скор новости для пульса
func (n *PersonalizedNews) CalculatePersonalScore() {
	// Базовая формула: релевантность * коэффициент свежести * коэффициент популярности
	baseScore := n.RelevanceScore

	// Коэффициент свежести (новости свежее = выше скор)
	hoursOld := time.Since(n.PublishedAt).Hours()
	freshnessCoeff := 1.0
	if hoursOld <= 1 {
		freshnessCoeff = 1.2
	} else if hoursOld <= 6 {
		freshnessCoeff = 1.1
	} else if hoursOld <= 24 {
		freshnessCoeff = 1.0
	} else if hoursOld <= 72 {
		freshnessCoeff = 0.9
	} else {
		freshnessCoeff = 0.8
	}

	// Коэффициент популярности (больше просмотров = выше скор)
	popularityCoeff := 1.0
	if n.ViewCount > 1000 {
		popularityCoeff = 1.2
	} else if n.ViewCount > 500 {
		popularityCoeff = 1.1
	} else if n.ViewCount > 100 {
		popularityCoeff = 1.05
	}

	n.PersonalScore = baseScore * freshnessCoeff * popularityCoeff

	// Ограничиваем скор в пределах [0, 1]
	if n.PersonalScore > 1.0 {
		n.PersonalScore = 1.0
	}
	if n.PersonalScore < 0.0 {
		n.PersonalScore = 0.0
	}
}
