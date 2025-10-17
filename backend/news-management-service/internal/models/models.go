package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// News представляет новость
type News struct {
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
	ParsedAt       time.Time `json:"parsed_at" db:"parsed_at"`
	RelevanceScore float64   `json:"relevance_score" db:"relevance_score"`
	ViewCount      int       `json:"view_count" db:"view_count"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`

	// Связанные данные (не хранятся в БД)
	Source   *NewsSource `json:"source,omitempty" db:"-"`
	Category *Category   `json:"category,omitempty" db:"-"`
	Country  *Country    `json:"country,omitempty" db:"-"`
	Tags     []Tag       `json:"tags,omitempty" db:"-"`
}

// NewsSource представляет источник новостей
type NewsSource struct {
	ID                   int        `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	Domain               string     `json:"domain" db:"domain"`
	RSSURL               string     `json:"rss_url" db:"rss_url"`
	WebsiteURL           string     `json:"website_url" db:"website_url"`
	CountryID            int        `json:"country_id" db:"country_id"`
	Language             string     `json:"language" db:"language"`
	Description          string     `json:"description" db:"description"`
	LogoURL              string     `json:"logo_url" db:"logo_url"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	LastParsedAt         *time.Time `json:"last_parsed_at" db:"last_parsed_at"`
	ParseIntervalMinutes int        `json:"parse_interval_minutes" db:"parse_interval_minutes"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Category представляет категорию новости
type Category struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Color       string    `json:"color" db:"color"`
	Icon        string    `json:"icon" db:"icon"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// GetDisplayName возвращает русское название категории
func (c *Category) GetDisplayName() string {
	displayNames := map[string]string{
		"sport":     "Спорт",
		"tech":      "Технологии",
		"politics":  "Политика",
		"economy":   "Экономика и финансы",
		"society":   "Общество",
	}

	if displayName, exists := displayNames[c.Slug]; exists {
		return displayName
	}
	return c.Name // Fallback к оригинальному названию
}

// Tag представляет тег новости
type Tag struct {
	ID         int       `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Slug       string    `json:"slug" db:"slug"`
	UsageCount int       `json:"usage_count" db:"usage_count"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Country представляет страну
type Country struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code" db:"code"`
	FlagEmoji string    `json:"flag_emoji" db:"flag_emoji"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NewsFilter представляет фильтры для поиска новостей
type NewsFilter struct {
	SourceIDs    []int      `json:"source_ids,omitempty"`
	CategoryIDs  []int      `json:"category_ids,omitempty"`
	CountryIDs   []int      `json:"country_ids,omitempty"`
	Keywords     string     `json:"keywords,omitempty"`
	DateFrom     *time.Time `json:"date_from,omitempty"`
	DateTo       *time.Time `json:"date_to,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	MinRelevance *float64   `json:"min_relevance,omitempty"`
	Page         int        `json:"page,omitempty"`
	PageSize     int        `json:"page_size,omitempty"`
	SortBy       string     `json:"sort_by,omitempty"`    // published_at, relevance_score, created_at, view_count
	SortOrder    string     `json:"sort_order,omitempty"` // asc, desc
}

// NewsResponse представляет ответ с новостями
type NewsResponse struct {
	News       []News     `json:"news"`
	Pagination Pagination `json:"pagination"`
	Filters    NewsFilter `json:"filters"`
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

// NewsStats представляет статистику новостей
type NewsStats struct {
	TotalNews     int             `json:"total_news"`
	ActiveNews    int             `json:"active_news"`
	NewsToday     int             `json:"news_today"`
	NewsThisWeek  int             `json:"news_this_week"`
	TopCategories []CategoryStats `json:"top_categories"`
	TopSources    []SourceStats   `json:"top_sources"`
	TopCountries  []CountryStats  `json:"top_countries"`
	RecentNews    []News          `json:"recent_news"`
	TrendingNews  []News          `json:"trending_news"`
}

// CategoryStats представляет статистику по категории
type CategoryStats struct {
	Category  Category `json:"category"`
	NewsCount int      `json:"news_count"`
}

// SourceStats представляет статистику по источнику
type SourceStats struct {
	Source    NewsSource `json:"source"`
	NewsCount int        `json:"news_count"`
}

// CountryStats представляет статистику по стране
type CountryStats struct {
	Country   Country `json:"country"`
	NewsCount int     `json:"news_count"`
}

// SearchResult представляет результат поиска
type SearchResult struct {
	News        []News     `json:"news"`
	Pagination  Pagination `json:"pagination"`
	Query       string     `json:"query"`
	SearchTime  string     `json:"search_time"`
	Suggestions []string   `json:"suggestions,omitempty"`
}

// StringArray представляет массив строк для работы с PostgreSQL
type StringArray []string

// Scan реализует интерфейс sql.Scanner для StringArray
func (a *StringArray) Scan(value interface{}) error {
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
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value реализует интерфейс driver.Valuer для StringArray
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Validation методы

// IsValid проверяет валидность новости
func (n *News) IsValid() bool {
	return len(n.Title) >= 10 && len(n.Title) <= 500 &&
		n.URL != "" &&
		n.SourceID > 0 &&
		!n.PublishedAt.IsZero()
}

// IsRecent проверяет, является ли новость свежей (опубликована в последние 24 часа)
func (n *News) IsRecent() bool {
	return time.Since(n.PublishedAt) <= 24*time.Hour
}

// IsTrending проверяет, является ли новость трендовой
func (n *News) IsTrending() bool {
	return n.RelevanceScore >= 0.8 && n.ViewCount > 100
}

// GetAgeInHours возвращает возраст новости в часах
func (n *News) GetAgeInHours() int {
	return int(time.Since(n.PublishedAt).Hours())
}

// Методы для работы с фильтрами

// Validate проверяет корректность фильтра
func (f *NewsFilter) Validate(maxPageSize int) error {
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

	if f.Keywords != "" && len(f.Keywords) > 500 {
		return fmt.Errorf("keywords too long")
	}

	if f.DateFrom != nil && f.DateTo != nil && f.DateFrom.After(*f.DateTo) {
		return fmt.Errorf("date_from cannot be after date_to")
	}

	if f.MinRelevance != nil && (*f.MinRelevance < 0 || *f.MinRelevance > 1) {
		return fmt.Errorf("min_relevance must be between 0 and 1")
	}

	return nil
}

// GetOffset возвращает offset для SQL запроса
func (f *NewsFilter) GetOffset() int {
	return (f.Page - 1) * f.PageSize
}

// HasFilters проверяет, есть ли активные фильтры
func (f *NewsFilter) HasFilters() bool {
	return len(f.SourceIDs) > 0 ||
		len(f.CategoryIDs) > 0 ||
		len(f.CountryIDs) > 0 ||
		f.Keywords != "" ||
		f.DateFrom != nil ||
		f.DateTo != nil ||
		f.MinRelevance != nil
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

// Константы для сортировки
const (
	SortByPublishedAt    = "published_at"
	SortByRelevanceScore = "relevance_score"
	SortByCreatedAt      = "created_at"
	SortByViewCount      = "view_count"
	SortByUpdatedAt      = "updated_at"

	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// GetValidSortFields возвращает список допустимых полей для сортировки
func GetValidSortFields() []string {
	return []string{
		SortByPublishedAt,
		SortByRelevanceScore,
		SortByCreatedAt,
		SortByViewCount,
		SortByUpdatedAt,
	}
}

// IsValidSortField проверяет, является ли поле валидным для сортировки
func IsValidSortField(field string) bool {
	validFields := GetValidSortFields()
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

// NormalizeSortParams нормализует параметры сортировки
func NormalizeSortParams(sortBy, sortOrder string) (string, string) {
	if !IsValidSortField(sortBy) {
		sortBy = SortByPublishedAt
	}

	if !IsValidSortOrder(sortOrder) {
		sortOrder = SortOrderDesc
	}

	return sortBy, sortOrder
}

// Константы для категорий (соответствуют данным в БД)
const (
	CategorySports     = 1 // Спорт
	CategoryTechnology = 2 // Технологии
	CategoryPolitics   = 3 // Политика
	CategoryEconomics  = 4 // Экономика и финансы
	CategorySociety    = 5 // Общество
)

// GetCategoryName возвращает название категории по ID
func GetCategoryName(id int) string {
	categories := map[int]string{
		CategorySports:     "Спорт",
		CategoryTechnology: "Технологии",
		CategoryPolitics:   "Политика",
		CategoryEconomics:  "Экономика и финансы",
		CategorySociety:    "Общество",
	}

	if name, exists := categories[id]; exists {
		return name
	}
	return "Неизвестная категория"
}
