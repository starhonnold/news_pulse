<template>
  <div class="q-pa-md">
    <!-- Заголовок страницы -->
    <div class="row q-mb-md">
      <div class="col">
        <div class="text-subtitle1 text-grey-7">
          Просматривайте все новости с возможностью фильтрации и сортировки
        </div>
      </div>
    </div>

    <!-- Фильтры и поиск -->
    <q-card class="q-mb-md modern-card glass-effect filters-card">
      <q-card-section class="filters-section">
        <div class="filters-grid">
          <!-- Поиск -->
          <div class="filter-item">
            <q-input
              v-model="searchQuery"
              placeholder="Поиск новостей..."
              dense
              outlined
              class="modern-input filter-input"
              @update:model-value="onSearch"
            >
              <template v-slot:prepend>
                <q-icon name="search" />
              </template>
              <template v-slot:append>
                <q-btn
                  v-if="searchQuery"
                  flat
                  round
                  dense
                  icon="clear"
                  @click="clearSearch"
                />
              </template>
            </q-input>
          </div>

          <!-- Фильтр по категориям -->
          <div class="filter-item">
            <q-select
              v-model="selectedCategories"
              :options="categoryOptions"
              label="Категории"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="modern-input filter-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCategoriesClear"
            />
          </div>

          <!-- Фильтр по странам -->
          <div class="filter-item">
            <q-select
              v-model="selectedCountries"
              :options="countryOptions"
              label="Страны"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="modern-input filter-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCountriesClear"
            />
          </div>

          <!-- Фильтр по дате -->
          <div class="filter-item">
            <q-input
              v-model="dateRangeText"
              label="Период"
              dense
              outlined
              readonly
              class="modern-input filter-input date-input"
              @click="showDatePicker = true"
            >
              <template v-slot:prepend>
                <q-icon name="calendar_month" class="cursor-pointer date-calendar-icon" @click="showDatePicker = true" />
              </template>
              <template v-slot:append>
                <q-btn
                  v-if="dateRangeText"
                  flat
                  round
                  dense
                  icon="clear"
                  @click="clearDateFilter"
                />
              </template>
            </q-input>
          </div>
        </div>
      </q-card-section>
    </q-card>

    <!-- Статистика -->
    <q-card class="status-card glass-effect q-mb-md">
      <q-card-section class="row items-center">
        <div class="col">
          <div class="text-caption text-secondary">
            Найдено: <span class="text-primary text-weight-medium">{{ totalNews }} новостей</span>
            <span v-if="allNews.length < totalNews" class="text-grey-6 q-ml-sm">
              (показано: {{ allNews.length }})
            </span>
            <!-- Иконка сортировки и кнопка очистки фильтров -->
            <span class="q-ml-md">
              <q-btn
                flat
                round
                dense
                :icon="sortOrder === 'desc' ? 'keyboard_arrow_down' : 'keyboard_arrow_up'"
                :color="sortOrder === 'desc' ? 'primary' : 'grey-6'"
                @click="toggleSortOrder"
                class="sort-btn q-mr-sm"
                size="sm"
              >
                <q-tooltip>
                  {{ sortOrder === 'desc' ? 'Сначала новые' : 'Сначала старые' }}
                </q-tooltip>
              </q-btn>
              <q-btn
                v-if="hasActiveFilters"
                flat
                dense
                icon="clear_all"
                label="Очистить все фильтры"
                color="secondary"
                @click="clearAllFilters"
                class="sort-btn"
                size="sm"
              >
                <q-tooltip>
                  Очистить все фильтры
                </q-tooltip>
              </q-btn>
            </span>
          </div>
        </div>
        <div class="col-auto">
          <div class="text-caption text-secondary">
            Обновлено: <span class="text-primary">{{ lastUpdate }}</span>
          </div>
        </div>
      </q-card-section>
    </q-card>

    <!-- Индикатор загрузки -->
    <div v-if="loading" class="row justify-center q-my-md">
      <q-spinner-dots color="primary" size="40px" />
    </div>

    <!-- Список новостей -->
    <div v-else class="row">
      <div class="col-12">
        <q-infinite-scroll @load="loadMoreNews" :offset="250">
          <!-- Сообщение если новостей нет -->
          <div v-if="paginatedNews.length === 0" class="row justify-center q-my-xl">
            <div class="text-center">
              <q-icon name="article" size="64px" color="grey-5" class="q-mb-md" />
              <div class="text-h6 text-grey-6">Новости не найдены</div>
              <div class="text-body2 text-grey-5">Попробуйте изменить фильтры или поисковый запрос</div>
            </div>
          </div>

          <!-- Список новостей -->
          <div v-else class="row q-gutter-md stagger-animation">
            <div 
              v-for="news in paginatedNews" 
              :key="news.id"
              class="col-12"
            >
              <q-card
                class="news-card cursor-pointer fade-in-up"
                @click="openNews(news)"
              >
                <!-- Изображение новости -->
                <div v-if="isValidImageUrl(news.image_url || news.image)" class="news-image-wrapper">
                  <q-img
                    :src="news.image_url || news.image"
                    :ratio="16/9"
                    class="news-image"
                    fit="cover"
                  >
                    <template v-slot:error>
                      <div class="absolute-full flex flex-center bg-grey-3">
                        <q-icon name="image" size="lg" color="grey-6" />
                      </div>
                    </template>
                  </q-img>
                </div>

                <!-- Контент новости -->
                <q-card-section class="q-pa-md mobile-card-section">
                      <!-- Мета информация -->
                      <div class="row items-center q-mb-sm mobile-news-meta-wrapper">
                        <div class="col-12 col-sm-auto">
                          <div class="news-meta mobile-news-meta">
                            <span class="country-flag q-mr-xs">{{ news.country?.flag_emoji || news.country?.flag || '🌍' }}</span>
                            <span class="source-name text-weight-medium text-primary mobile-source-name">
                              {{ news.source?.name || news.source_name || 'Неизвестный источник' }}
                            </span>
                            <q-separator vertical class="q-mx-sm mobile-separator" />
                            <span class="text-grey-7 mobile-date">{{ formatDate(news.published_at) }}</span>
                          </div>
                        </div>
                        <div class="col-12 col-sm-auto q-mt-xs q-mt-sm-none">
                          <q-chip
                            v-if="news.category && news.category.name"
                            :style="{
                              background: `linear-gradient(135deg, ${getCategoryColor(news.category.color)} 0%, ${lightenColor(news.category.color, 20)} 100%) !important`,
                              color: 'white !important',
                              border: 'none !important'
                            }"
                            class="category-chip-modern"
                          >
                            <q-icon :name="getCategoryIcon(news.category.icon || news.category.slug)" class="category-icon-modern" />
                            {{ news.category.name }}
                          </q-chip>
                        </div>
                      </div>

                      <!-- Заголовок -->
                      <div class="news-title text-h6 text-weight-medium q-mb-sm">
                        {{ cleanText(news.title) }}
                      </div>

                      <!-- Описание -->
                      <div class="news-description text-grey-8 q-mb-sm">
                        {{ cleanText(news.description) }}
                      </div>

                  <!-- Действия справа -->
                  <div class="row items-center justify-between q-mt-sm">
                    <div class="col">
                      <div class="news-actions">
                        <q-icon name="visibility" class="q-mr-xs" size="sm" />
                        <span class="text-body2">{{ news.view_count || 0 }}</span>
                      </div>
                    </div>
                    <div class="col-auto">
                      <div class="news-action-buttons">
                        <q-btn
                          flat
                          round
                          dense
                          icon="share"
                          size="sm"
                          @click.stop="shareNews(news)"
                          class="q-mr-xs"
                        >
                          <q-tooltip>Поделиться</q-tooltip>
                        </q-btn>
                        <q-btn
                          flat
                          round
                          dense
                          icon="bookmark_border"
                          size="sm"
                          @click.stop="bookmarkNews(news)"
                        >
                          <q-tooltip>В закладки</q-tooltip>
                        </q-btn>
                      </div>
                    </div>
                  </div>
                </q-card-section>

                <!-- Индикатор новой новости -->
                <div
                  v-if="isNewNews(news)"
                  class="absolute-top-left q-ma-sm"
                >
                  <q-badge color="green" floating>
                    Новое
                  </q-badge>
                </div>
              </q-card>
            </div>
          </div>

          <template v-slot:loading>
            <div class="row justify-center q-my-md">
              <q-spinner-dots color="primary" size="40px" />
            </div>
          </template>
        </q-infinite-scroll>
      </div>
    </div>

    <!-- Диалог просмотра новости -->
    <q-dialog v-model="showNewsDialog" maximized>
      <q-card>
        <q-card-section class="row items-center q-pb-none">
          <div class="text-h6">Новость</div>
          <q-space />
          <q-btn flat round dense icon="close" v-close-popup />
        </q-card-section>

        <q-card-section v-if="selectedNews" class="dialog-news-section">
          <!-- Изображение новости -->
          <div v-if="isValidImageUrl(selectedNews.image_url || selectedNews.image)" class="dialog-image-wrapper">
            <q-img
              :src="selectedNews.image_url || selectedNews.image"
              :ratio="16/9"
              class="rounded-borders dialog-news-image"
              fit="cover"
            >
              <template v-slot:error>
                <div class="absolute-full flex flex-center bg-grey-3">
                  <q-icon name="image" size="lg" color="grey-6" />
                </div>
              </template>
            </q-img>
          </div>

          <!-- Контент новости -->
          <div class="dialog-news-content">
                <!-- Мета информация -->
                <div class="row items-center q-mb-sm">
                  <div class="col-auto">
                    <div class="news-meta">
                      <span class="country-flag q-mr-xs">{{ selectedNews.country?.flag_emoji || selectedNews.country?.flag || '🌍' }}</span>
                            <span class="source-name text-weight-medium text-primary">
                              {{ cleanText(selectedNews.source?.name || 'Неизвестный источник') }}
                            </span>
                      <q-separator vertical class="q-mx-sm" />
                      <span class="text-grey-7">{{ formatDate(selectedNews.published_at) }}</span>
                    </div>
                  </div>
                  <div class="col-auto">
                    <q-chip
                      v-if="selectedNews.category && selectedNews.category.name"
                      :style="{
                        background: `linear-gradient(135deg, ${getCategoryColor(selectedNews.category.color)} 0%, ${lightenColor(selectedNews.category.color, 20)} 100%) !important`,
                        color: 'white !important',
                        border: 'none !important'
                      }"
                      class="category-chip-modern"
                    >
                      <q-icon :name="getCategoryIcon(selectedNews.category.icon || selectedNews.category.slug)" class="category-icon-modern" />
                      {{ selectedNews.category.name }}
                    </q-chip>
                  </div>
                </div>

                <!-- Заголовок -->
                <div class="news-title text-h5 text-weight-medium q-mb-md">
                  {{ cleanText(selectedNews.title) }}
                </div>

                <!-- Описание -->
                <div class="news-description text-body1 text-grey-8 q-mb-md">
                  {{ cleanText(selectedNews.description) }}
                </div>

                <!-- Полный текст новости -->
                <div v-if="selectedNews.content" class="news-content q-mb-lg">
                  <div 
                    class="news-content-text text-body1 text-grey-8 q-mb-md"
                    :class="{ 'error-content': isContentCorrupted(selectedNews.content) }"
                  >
                    {{ cleanNewsContent(selectedNews.content) }}
                  </div>
                  <div class="row justify-center">
                    <q-btn
                      color="primary"
                      label="Читать полностью"
                      @click="openOriginalNews(selectedNews.url)"
                      target="_blank"
                      icon="open_in_new"
                      class="q-px-lg"
                    />
                  </div>
                </div>

            <!-- Действия -->
            <div class="row justify-center q-mt-lg">
              <q-btn
                color="secondary"
                label="Закрыть"
                @click="showNewsDialog = false"
                flat
                class="q-px-xl"
              />
            </div>
          </div>
        </q-card-section>
      </q-card>
    </q-dialog>

    <!-- Диалог выбора периода -->
    <q-dialog v-model="showDatePicker" class="date-picker-dialog">
      <q-card class="date-picker-card">
        <q-card-section class="date-picker-header">
          <div class="date-picker-title">
            <q-icon name="event" size="24px" class="q-mr-sm" />
            <span>Выберите период</span>
          </div>
          <q-btn icon="close" flat round dense v-close-popup class="close-btn" />
        </q-card-section>

        <q-card-section class="date-picker-content">
          <!-- Выбор типа периода -->
          <div class="date-type-selector">
            <q-btn
              :class="dateFilterType === 'single' ? 'date-type-btn active' : 'date-type-btn'"
              flat
              label="Одна дата"
              icon="event"
              @click="dateFilterType = 'single'"
            />
            <q-btn
              :class="dateFilterType === 'range' ? 'date-type-btn active' : 'date-type-btn'"
              flat
              label="Период"
              icon="date_range"
              @click="dateFilterType = 'range'"
            />
          </div>

          <!-- Календарь для одной даты -->
          <div v-if="dateFilterType === 'single'" class="calendar-wrapper">
            <q-date
              v-model="selectedDate"
              :options="dateOptions"
              class="stylish-calendar"
              minimal
              @update:model-value="onSingleDateSelect"
            />
          </div>

          <!-- Календарь для периода -->
          <div v-if="dateFilterType === 'range'" class="calendar-wrapper">
            <q-date
              v-model="dateRange"
              range
              :options="dateOptions"
              class="stylish-calendar"
              minimal
              @update:model-value="onDateRangeSelect"
            />
          </div>
        </q-card-section>

        <q-card-actions class="date-picker-actions">
          <q-btn 
            flat 
            label="Отмена" 
            v-close-popup 
            class="action-btn cancel-btn" 
          />
          <q-btn
            label="Применить"
            @click="applyDateFilter"
            class="action-btn apply-btn"
            :disable="!hasDateSelection"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../services/api'

// Реактивные данные
const searchQuery = ref('')
const selectedCategories = ref([])
const selectedCountries = ref([])
const sortOrder = ref('desc')
const showNewsDialog = ref(false)
const selectedNews = ref(null)
const lastUpdate = ref('')
const allNews = ref([])
const loading = ref(false)

// Фильтр по дате
const showDatePicker = ref(false)
const dateFilterType = ref('single') // 'single' или 'range'
const selectedDate = ref('')
const dateRange = ref({ from: '', to: '' })
const dateRangeText = ref('')

// Инициализация с пустыми массивами для избежания null значений
selectedCategories.value = []
selectedCountries.value = []

// Опции для селектов
const categoryOptions = ref([])
const countryOptions = ref([])

// Вычисляемые свойства
const filteredNews = computed(() => {
  // Убеждаемся, что allNews.value - это массив
  let news = Array.isArray(allNews.value) ? allNews.value : []
  
  // Сортировка (фильтрация уже делается на сервере)
  if (sortOrder.value === 'desc') {
    news.sort((a, b) => new Date(b.published_at) - new Date(a.published_at))
  } else {
    news.sort((a, b) => new Date(a.published_at) - new Date(b.published_at))
  }
  
  return news
})

const paginatedNews = computed(() => {
  return filteredNews.value // Показываем все загруженные новости
})

const hasActiveFilters = computed(() => {
  return searchQuery.value || 
         (selectedCategories.value && selectedCategories.value.length > 0) || 
         (selectedCountries.value && selectedCountries.value.length > 0) ||
         dateRangeText.value
})

const hasDateSelection = computed(() => {
  if (dateFilterType.value === 'single') {
    return selectedDate.value
  } else {
    return dateRange.value.from && dateRange.value.to
  }
})

// Методы
// Пагинация
const currentPage = ref(1)
const pageSize = ref(10)
const totalNews = ref(0)
const hasMoreNews = ref(true)

const loadNews = async (page = 1, reset = false) => {
  try {
    if (reset) {
      loading.value = true
      currentPage.value = 1
      allNews.value = []
      hasMoreNews.value = true
    }
    
    console.log(`Загружаем страницу ${page}...`)
    
    const params = {
      page: page,
      page_size: pageSize.value,
      sort_by: 'published_at',
      sort_order: 'desc'
    }
    
    // Добавляем параметры поиска и фильтрации
    if (searchQuery.value) {
      params.keywords = searchQuery.value
    }
    if (selectedCategories.value && selectedCategories.value.length > 0) {
      params.categories = selectedCategories.value.join(',')
    }
    if (selectedCountries.value && selectedCountries.value.length > 0) {
      params.countries = selectedCountries.value.join(',')
    }
    
    const response = await api.get('/news', { params })
    
    console.log('Ответ API новостей:', response.data)
    
    let newNews = []
    if (response.data?.success && response.data?.data?.news) {
      newNews = response.data.data.news
      // Всегда обновляем totalNews из пагинации
      totalNews.value = response.data.data.pagination?.total || 0
    } else if (response.data?.data && Array.isArray(response.data.data)) {
      newNews = response.data.data
      totalNews.value = newNews.length
    } else if (Array.isArray(response.data)) {
      newNews = response.data
      totalNews.value = newNews.length
    }
    
    if (reset) {
      allNews.value = newNews
    } else {
      allNews.value = [...allNews.value, ...newNews]
    }
    
    // Проверяем, есть ли еще новости для загрузки
    if (response.data?.success && response.data?.data?.pagination) {
      hasMoreNews.value = response.data.data.pagination.has_next || false
    } else {
      hasMoreNews.value = newNews.length === pageSize.value
    }
    
    console.log(`Загружено ${newNews.length} новостей, всего: ${allNews.value.length}`)
    console.log('Total news from API:', totalNews.value)
    console.log('Первая новость:', newNews[0])
    console.log('Фильтры:', { search: searchQuery.value, categories: selectedCategories.value, countries: selectedCountries.value })
    console.log('API параметры:', params)
    
    if (reset) {
      await loadFilters()
      lastUpdate.value = new Date().toLocaleTimeString()
    }
  } catch (error) {
    console.error('Ошибка загрузки новостей:', error)
    if (reset) {
      allNews.value = []
    }
  } finally {
    loading.value = false
  }
}

const loadFilters = async () => {
  try {
    // Загружаем категории
    const categoriesResponse = await api.get('/categories')
    const categories = categoriesResponse.data?.data || categoriesResponse.data || []
    console.log('Загруженные категории:', categories)
    categoryOptions.value = categories.map(cat => ({
      label: cat.name,
      value: cat.id,
      color: cat.color,
      icon: cat.icon,
      slug: cat.slug
    }))
    
    // Загружаем страны
    const countriesResponse = await api.get('/countries')
    const countries = countriesResponse.data?.data || countriesResponse.data || []
    console.log('Загруженные страны:', countries)
    countryOptions.value = countries.map(country => ({
      label: country.name,
      value: country.id,
      flag: country.flag
    }))
  } catch (error) {
    console.error('Ошибка загрузки фильтров:', error)
    // Показываем сообщение об ошибке пользователю
    console.warn('Не удалось загрузить фильтры. Проверьте подключение к серверу.')
  }
}

let searchTimeout = null

const onSearch = async () => {
  // Очищаем предыдущий таймаут
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  
  // Устанавливаем новый таймаут для поиска (500ms задержка)
  searchTimeout = setTimeout(async () => {
    await loadNews(1, true)
  }, 500)
}

const onFilterChange = async () => {
  // При изменении фильтров перезагружаем новости
  await loadNews(1, true)
}



const clearSearch = () => {
  searchQuery.value = ''
}

const clearAllFilters = () => {
  searchQuery.value = ''
  selectedCategories.value = []
  selectedCountries.value = []
  sortOrder.value = 'desc'
}

// Обработчики для очистки отдельных фильтров
const onCategoriesClear = () => {
  selectedCategories.value = []
  onFilterChange()
}

const onCountriesClear = () => {
  selectedCountries.value = []
  onFilterChange()
}

const toggleSortOrder = async () => {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  await loadNews(1, true)
}

const loadMoreNews = async (index, done) => {
  if (!hasMoreNews.value || loading.value) {
    done()
    return
  }
  
  try {
    currentPage.value++
    await loadNews(currentPage.value, false)
    done()
  } catch (error) {
    console.error('Ошибка загрузки дополнительных новостей:', error)
    done()
  }
}

// Функция для очистки текста от HTML-сущностей
const cleanText = (text) => {
  if (!text) return ''
  
  return text
    .replace(/&nbsp;/g, ' ') // Заменяем неразрывные пробелы
    .replace(/&amp;/g, '&') // Заменяем HTML-сущности
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&apos;/g, "'")
    .replace(/&mdash;/g, '—')
    .replace(/&ndash;/g, '–')
    .replace(/&hellip;/g, '…')
    .replace(/&laquo;/g, '«')
    .replace(/&raquo;/g, '»')
    // Декодируем числовые HTML-сущности
    .replace(/&#(\d+);/g, (match, dec) => String.fromCharCode(dec))
    .replace(/&#x([0-9a-fA-F]+);/g, (match, hex) => String.fromCharCode(parseInt(hex, 16)))
    .replace(/<[^>]*>/g, '') // Убираем HTML теги
    .replace(/\s+/g, ' ') // Заменяем множественные пробелы на один
    .trim()
}

// Функция для проверки и очистки контента новости
const cleanNewsContent = (content) => {
  if (!content) return ''
  
  // Проверяем, является ли контент искаженным (содержит много непечатаемых символов)
  let nonPrintableCount = 0
  const totalLength = content.length
  
  // Подсчитываем непечатаемые символы вручную
  for (let i = 0; i < content.length; i++) {
    const charCode = content.charCodeAt(i)
    // Проверяем диапазоны непечатаемых символов
    if ((charCode >= 0 && charCode <= 8) || // \u0000-\u0008
        charCode === 11 || // \u000B
        charCode === 12 || // \u000C
        (charCode >= 14 && charCode <= 31) || // \u000E-\u001F
        (charCode >= 127 && charCode <= 159)) { // \u007F-\u009F
      nonPrintableCount++
    }
  }
  
  // Если более 20% символов непечатаемые, считаем контент искаженным
  if (nonPrintableCount / totalLength > 0.2) {
    console.warn('Обнаружен искаженный контент новости:', {
      totalLength,
      nonPrintableCount,
      ratio: nonPrintableCount / totalLength,
      preview: content.substring(0, 100)
    })
    return 'Контент новости недоступен или поврежден. Рекомендуется прочитать оригинальную статью.'
  }
  
  // Очищаем контент от непечатаемых символов
  let cleanedContent = ''
  for (let i = 0; i < content.length; i++) {
    const charCode = content.charCodeAt(i)
    // Пропускаем непечатаемые символы
    if (!((charCode >= 0 && charCode <= 8) ||
          charCode === 11 ||
          charCode === 12 ||
          (charCode >= 14 && charCode <= 31) ||
          (charCode >= 127 && charCode <= 159))) {
      cleanedContent += content[i]
    }
  }
  
  // Заменяем множественные пробелы на один
  cleanedContent = cleanedContent.replace(/\s+/g, ' ').trim()
  
  // Если после очистки контент стал слишком коротким, считаем его недоступным
  if (cleanedContent.length < 50) {
    return 'Контент новости недоступен или поврежден. Рекомендуется прочитать оригинальную статью.'
  }
  
  return cleanText(cleanedContent)
}

// Функция для валидации URL изображения
const isValidImageUrl = (url) => {
  if (!url) return false
  
  try {
    const urlObj = new URL(url)
    
    // Проверяем протокол
    if (urlObj.protocol !== 'http:' && urlObj.protocol !== 'https:') {
      return false
    }
    
    // Проверяем проблемные домены
    // Все поддомены cdnn*.img.ria.ru не работают
    if (urlObj.hostname.includes('cdnn') && urlObj.hostname.includes('img.ria.ru')) {
      return false
    }
    
    // Дополнительная проверка на другие проблемные домены
    const problematicDomains = [
      'example.com', // Добавьте другие проблемные домены при необходимости
    ]
    
    if (problematicDomains.includes(urlObj.hostname)) {
      return false
    }
    
    return true
  } catch {
    return false
  }
}


const openNews = (news) => {
  console.log('Открываем новость:', {
    id: news.id,
    title: news.title,
    description: news.description,
    content: news.content ? news.content.substring(0, 200) + '...' : 'Нет контента',
    url: news.url
  })
  
  selectedNews.value = news
  showNewsDialog.value = true
}

const openOriginalNews = (url) => {
  if (url) {
    window.open(url, '_blank')
  }
}

const shareNews = (news) => {
  if (navigator.share) {
    navigator.share({
      title: news.title,
      text: news.description,
      url: news.url
    }).catch(console.error)
  } else {
    // Fallback для браузеров без поддержки Web Share API
    const shareText = `${news.title}\n\n${news.description}\n\n${news.url}`
    navigator.clipboard.writeText(shareText).then(() => {
      // Можно показать уведомление о копировании
      console.log('Текст скопирован в буфер обмена')
    }).catch(console.error)
  }
}

const bookmarkNews = (news) => {
  // Здесь можно добавить логику сохранения в закладки
  console.log('Добавить в закладки:', news.title)
  // Можно показать уведомление
}

const formatDate = (date) => {
  if (!date) return ''
  const newsDate = new Date(date)
  const now = new Date()
  const diffInHours = Math.floor((now - newsDate) / (1000 * 60 * 60))
  
  // Если новость в будущем или очень старая, всегда показываем полную дату
  if (diffInHours < 0 || diffInHours > 24) {
    return newsDate.toLocaleString('ru-RU', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone
    })
  }
  
  // Если новость свежая (менее 24 часов), показываем относительное время
  if (diffInHours < 1) {
    const diffInMinutes = Math.floor((now - newsDate) / (1000 * 60))
    return diffInMinutes < 1 ? 'Только что' : `${diffInMinutes} мин. назад`
  }
  return `${diffInHours} ч. назад`
}

const isNewNews = (news) => {
  if (!news.published_at) return false
  const now = new Date()
  const newsDate = new Date(news.published_at)
  const diffInHours = (now - newsDate) / (1000 * 60 * 60)
  return diffInHours < 24
}

// Функция для преобразования цветов Quasar в CSS цвета
const getCategoryColor = (quasarColor) => {
  if (!quasarColor) return '#1976d2' // primary по умолчанию
  
  // Если уже hex-цвет, возвращаем как есть
  if (quasarColor.startsWith('#')) return quasarColor
  
  const colorMap = {
    'red-6': '#f44336',
    'green-6': '#4caf50',
    'blue-6': '#2196f3',
    'purple-6': '#9c27b0',
    'orange-6': '#ff9800',
    'indigo-6': '#3f51b5',
    'teal-6': '#009688',
    'amber-7': '#ff8f00',
    'pink-6': '#e91e63',
    'cyan-6': '#00bcd4',
    'deep-purple-6': '#673ab7',
    'brown-6': '#795548',
    'lime-6': '#cddc39',
    'yellow-6': '#ffeb3b'
  }
  
  return colorMap[quasarColor] || quasarColor || '#1976d2'
}

// Функция для осветления цвета (для градиента)
const lightenColor = (color, percent) => {
  if (!color) return '#8B8FF1'
  
  // Сначала преобразуем Quasar цвет в hex
  const hexColor = getCategoryColor(color)
  
  // Если цвет в формате hex
  if (hexColor.startsWith('#')) {
    const num = parseInt(hexColor.replace('#', ''), 16)
    const r = Math.min(255, ((num >> 16) & 0xFF) + Math.round(255 * percent / 100))
    const g = Math.min(255, ((num >> 8) & 0xFF) + Math.round(255 * percent / 100))
    const b = Math.min(255, (num & 0xFF) + Math.round(255 * percent / 100))
    return `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, '0')}`
  }
  
  return hexColor
}

// Функция для получения красивой иконки категории
const getCategoryIcon = (iconName) => {
  if (!iconName) return 'label'
  
  const iconMap = {
    // Спорт - яркие спортивные иконки
    'sport': 'sports_soccer',
    'sports': 'sports_soccer',
    'спорт': 'sports_soccer',
    'футбол': 'sports_soccer',
    'хоккей': 'sports_hockey',
    'баскетбол': 'sports_basketball',
    'теннис': 'sports_tennis',
    
    // Технологии - современные tech иконки
    'tech': 'devices',
    'technology': 'devices',
    'технологии': 'devices',
    'гаджеты': 'smartphone',
    'интернет': 'language',
    'ai': 'psychology',
    'ии': 'psychology',
    
    // Политика - официальные иконки
    'politics': 'account_balance',
    'политика': 'account_balance',
    'выборы': 'how_to_vote',
    'правительство': 'gavel',
    
    // Экономика и финансы - денежные иконки
    'economy': 'trending_up',
    'экономика': 'trending_up',
    'finance': 'paid',
    'финансы': 'paid',
    'банки': 'account_balance_wallet',
    'инвестиции': 'show_chart',
    'криптовалюта': 'currency_bitcoin',
    
    // Общество - социальные иконки
    'society': 'groups',
    'общество': 'groups',
    'социум': 'diversity_3',
    'люди': 'group',
    
    // Наука - научные иконки
    'science': 'biotech',
    'наука': 'biotech',
    'исследования': 'science',
    'космос': 'rocket_launch',
    
    // Культура - культурные иконки
    'culture': 'theater_comedy',
    'культура': 'theater_comedy',
    'искусство': 'palette',
    'музыка': 'music_note',
    'кино': 'movie',
    
    // Здоровье - медицинские иконки
    'health': 'medical_services',
    'здоровье': 'medical_services',
    'медицина': 'local_hospital',
    'вакцина': 'vaccines',
    
    // Развлечения - развлекательные иконки
    'entertainment': 'celebration',
    'развлечения': 'celebration',
    'шоу': 'stars',
    'игры': 'sports_esports',
    
    // Мир - международные иконки
    'world': 'public',
    'мир': 'public',
    'международные': 'flag',
    
    // Бизнес - деловые иконки
    'business': 'business_center',
    'бизнес': 'business_center',
    'компании': 'corporate_fare',
    'стартапы': 'rocket_launch',
    
    // Криминал - полицейские иконки
    'crime': 'local_police',
    'криминал': 'local_police',
    'преступность': 'gavel',
    
    // Происшествия - срочные иконки
    'incidents': 'emergency',
    'происшествия': 'emergency',
    'чп': 'warning',
    'авария': 'car_crash',
    
    // Образование - образовательные иконки
    'education': 'school',
    'образование': 'school',
    'университет': 'school',
    'студенты': 'menu_book',
    
    // Природа и экология - природные иконки
    'nature': 'eco',
    'природа': 'eco',
    'экология': 'energy_savings_leaf',
    'погода': 'wb_sunny',
    'климат': 'thermostat',
    
    // Дополнительные категории
    'авто': 'directions_car',
    'транспорт': 'commute',
    'недвижимость': 'home',
    'еда': 'restaurant',
    'туризм': 'flight',
    'мода': 'checkroom'
  }
  
  // Проверяем точное совпадение
  if (iconMap[iconName]) {
    return iconMap[iconName]
  }
  
  // Проверяем совпадение в нижнем регистре
  const lowerIconName = iconName.toLowerCase()
  if (iconMap[lowerIconName]) {
    return iconMap[lowerIconName]
  }
  
  // По умолчанию
  return 'label'
}


// Функция для проверки, является ли контент поврежденным
const isContentCorrupted = (content) => {
  if (!content) return false
  
  let nonPrintableCount = 0
  const totalLength = content.length
  
  // Подсчитываем непечатаемые символы вручную
  for (let i = 0; i < content.length; i++) {
    const charCode = content.charCodeAt(i)
    // Проверяем диапазоны непечатаемых символов
    if ((charCode >= 0 && charCode <= 8) || // \u0000-\u0008
        charCode === 11 || // \u000B
        charCode === 12 || // \u000C
        (charCode >= 14 && charCode <= 31) || // \u000E-\u001F
        (charCode >= 127 && charCode <= 159)) { // \u007F-\u009F
      nonPrintableCount++
    }
  }
  
  // Если более 20% символов непечатаемые, считаем контент поврежденным
  return nonPrintableCount / totalLength > 0.2
}

// Методы для работы с фильтром по дате
const dateOptions = (date) => {
  // Разрешаем выбирать только даты не позже сегодня
  return date <= new Date().toISOString().split('T')[0].replace(/-/g, '/')
}

const onSingleDateSelect = (val) => {
  selectedDate.value = val
}

const onDateRangeSelect = (val) => {
  dateRange.value = val
}

const applyDateFilter = () => {
  if (dateFilterType.value === 'single' && selectedDate.value) {
    const formattedDate = formatDateForDisplay(selectedDate.value)
    dateRangeText.value = formattedDate
  } else if (dateFilterType.value === 'range' && dateRange.value.from && dateRange.value.to) {
    const from = formatDateForDisplay(dateRange.value.from)
    const to = formatDateForDisplay(dateRange.value.to)
    dateRangeText.value = `${from} - ${to}`
  }
  showDatePicker.value = false
  onFilterChange()
}

const clearDateFilter = () => {
  selectedDate.value = ''
  dateRange.value = { from: '', to: '' }
  dateRangeText.value = ''
  onFilterChange()
}

const formatDateForDisplay = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const year = date.getFullYear()
  return `${day}.${month}.${year}`
}

// Жизненный цикл
onMounted(() => {
  loadNews(1, true)
})
</script>

<style lang="scss" scoped>
// === ФИЛЬТРЫ ===
.filters-card {
  border-radius: 16px !important;
  box-shadow: var(--shadow-md) !important;
}

.filters-section {
  padding: 20px !important;
}

.filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  align-items: start;
}

.filter-item {
  display: flex;
  flex-direction: column;
}

.filter-input {
  .q-field__control {
    min-height: 48px !important;
    background: var(--bg-tertiary) !important;
    border: 1px solid var(--border-primary) !important;
    border-radius: 12px !important;
    transition: all 0.3s ease !important;
    
    &:hover {
      border-color: var(--primary-color) !important;
      box-shadow: var(--shadow-sm) !important;
    }
    
    &:focus-within {
      border-color: var(--primary-color) !important;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1) !important;
    }
  }
  
  .q-field__native {
    color: var(--text-primary) !important;
    padding: 8px 12px !important;
  }
  
  .q-field__label {
    color: var(--text-tertiary) !important;
  }
  
  .q-field__prepend,
  .q-field__append {
    color: var(--text-secondary) !important;
  }
}

.date-input {
  cursor: pointer !important;
  
  .q-field__control {
    cursor: pointer !important;
  }
  
  .q-field__native {
    cursor: pointer !important;
  }
}

.date-calendar-icon {
  color: var(--primary-color) !important;
  font-size: 1.5rem !important;
  transition: all 0.3s ease !important;
  filter: drop-shadow(0 0 4px rgba(99, 102, 241, 0.3)) !important;
  
  &:hover {
    color: var(--primary-light) !important;
    transform: scale(1.1) !important;
    filter: drop-shadow(0 0 8px rgba(99, 102, 241, 0.5)) !important;
  }
}

// Адаптация категорий для мобильных
@media (max-width: 768px) {
  .category-chip-modern {
    font-size: 0.8rem !important;
    padding: 6px 14px !important;
    
    .category-icon-modern {
      font-size: 1.1rem !important;
      margin-right: 5px !important;
    }
  }
}

@media (max-width: 480px) {
  .category-chip-modern {
    font-size: 0.75rem !important;
    padding: 5px 12px !important;
    
    .category-icon-modern {
      font-size: 1rem !important;
      margin-right: 4px !important;
    }
  }
}

// Адаптация фильтров для мобильных
@media (max-width: 1200px) {
  .filters-grid {
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;
  }
}

@media (max-width: 768px) {
  .filters-section {
    padding: 16px !important;
  }
  
  .filters-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  
  .filter-input {
    .q-field__control {
      min-height: 44px !important;
    }
  }
}

@media (max-width: 480px) {
  .filters-section {
    padding: 12px !important;
  }
  
  .filters-grid {
    gap: 10px;
  }
}

.news-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.news-card {
  transition: all 0.3s ease;
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-lg);
    border-color: var(--border-accent);
  }
}

// Стили для диалога новостей
.news-meta {
  display: flex;
  align-items: center;
  font-size: 0.9em;
}

.country-flag {
  font-size: 1.2em;
}

.source-name {
  font-weight: 500;
}

.news-title {
  line-height: 1.3;
  font-weight: 600;
}

.news-description {
  line-height: 1.5;
  color: var(--text-secondary);
}

.news-image {
  border-radius: 8px;
  overflow: hidden;
}

.news-action-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
}

.news-actions {
  display: flex;
  align-items: center;
  font-size: 0.9em;
  color: var(--text-secondary);
}

.status-card {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
}

.modern-card {
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  box-shadow: var(--shadow-sm);
}

.modern-input {
  .q-field__control {
    border-radius: 8px;
  }
}

.stagger-animation > * {
  animation: fadeInUp 0.6s ease-out;
  animation-fill-mode: both;
}

.stagger-animation > *:nth-child(1) { animation-delay: 0.1s; }
.stagger-animation > *:nth-child(2) { animation-delay: 0.2s; }
.stagger-animation > *:nth-child(3) { animation-delay: 0.3s; }
.stagger-animation > *:nth-child(4) { animation-delay: 0.4s; }
.stagger-animation > *:nth-child(5) { animation-delay: 0.5s; }

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.sort-btn {
  transition: all 0.3s ease;
  border: 1px solid transparent;
  
  &:hover {
    transform: scale(1.1);
    border-color: var(--q-primary);
  }
  
  &.q-btn--dense {
    min-height: 24px;
    padding: 4px;
  }
}

// Стили для множественного выбора
.q-select--multiple {
  .q-field__native {
    min-height: 40px;
  }
  
  .q-chip {
    margin: 2px;
  }
}

// Стили для опций в выпадающем списке
.q-item {
  &.q-item--clickable {
    &:hover {
      background-color: var(--q-primary-light);
    }
  }
}

// Стили для кнопки очистки фильтров
.q-btn {
  &.q-btn--disabled {
    opacity: 0.5;
  }
}

// Стили для полного текста новости
.news-content {
  border-top: 1px solid var(--border-primary);
  padding-top: 16px;
  
  .news-content-text {
    line-height: 1.6;
    text-align: justify;
    white-space: pre-wrap;
    word-wrap: break-word;
    
    // Стили для сообщения об ошибке контента
    &.error-content {
      background-color: var(--q-orange-1);
      border: 1px solid var(--q-orange-3);
      border-radius: 8px;
      padding: 16px;
      text-align: center;
      color: var(--q-orange-8);
      font-style: italic;
    }
  }
}

// Стили для изображения новости
.news-image-wrapper {
  width: 100%;
  overflow: hidden;
  
  .news-image {
    width: 100%;
    transition: transform 0.3s ease;
  }
}

.news-card:hover .news-image {
  transform: scale(1.05);
}

// Ограничение размера изображений для десктопа
@media (min-width: 600px) {
  .news-card {
    display: flex;
    flex-direction: row;
    
    .news-image-wrapper {
      width: 240px;
      min-width: 240px;
      max-width: 240px;
      height: 160px;
      flex-shrink: 0;
      
      .news-image {
        height: 100%;
        width: 100%;
        object-fit: cover;
      }
    }
  }
}

@media (min-width: 1024px) {
  .news-card {
    .news-image-wrapper {
      width: 280px;
      min-width: 280px;
      max-width: 280px;
      height: 180px;
    }
  }
}

// Мобильная адаптация
@media (max-width: 599px) {
  .mobile-card-section {
    padding: 16px !important;
  }
  
  .mobile-news-meta-wrapper {
    flex-direction: column !important;
    align-items: flex-start !important;
    gap: 8px;
  }
  
  .mobile-news-meta {
    font-size: 0.875rem !important;
    flex-wrap: wrap;
    display: flex;
    align-items: center;
    gap: 8px;
    
    .mobile-source-name {
      font-size: 0.9rem !important;
      font-weight: 600 !important;
    }
    
    .mobile-date {
      font-size: 0.85rem !important;
    }
    
    .mobile-separator {
      height: 14px !important;
    }
  }
  
  // Современные чипы категорий с градиентом
  .category-chip-modern {
    font-size: 0.875rem !important;
    font-weight: 600 !important;
    padding: 8px 16px !important;
    border-radius: 24px !important;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15) !important;
    transition: all 0.3s ease !important;
    border: none !important;
    cursor: default !important;
    
    // Убираем стандартный before эффект Quasar
    &:before {
      display: none !important;
    }
    
    // Важно: не перезаписываем фон, чтобы работали inline-стили
    &:not([style*="background"]) {
      background: #6366F1 !important;
    }
    
    &:hover {
      transform: translateY(-2px) scale(1.05) !important;
      box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25) !important;
    }
    
    .q-chip__content {
      color: white !important;
    }
    
    .category-icon-modern {
      color: white !important;
      font-size: 1.2rem !important;
      margin-right: 6px !important;
      filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2)) !important;
      transition: all 0.3s ease !important;
    }
    
    &:hover .category-icon-modern {
      transform: scale(1.1) rotate(5deg) !important;
      filter: drop-shadow(0 3px 6px rgba(0, 0, 0, 0.3)) !important;
    }
  }
  
  .news-card {
    border-radius: 16px !important;
    margin-bottom: 16px !important;
    overflow: hidden;
    
    .news-image-wrapper {
      width: 100%;
      
      .news-image {
        width: 100%;
      }
    }
    
    .news-title {
      font-size: 1.1rem !important;
      line-height: 1.5 !important;
      font-weight: 600 !important;
      margin-bottom: 12px !important;
      word-break: break-word;
      overflow-wrap: break-word;
    }
    
    .news-description {
      font-size: 0.95rem !important;
      line-height: 1.6 !important;
      color: var(--text-secondary) !important;
      word-break: break-word;
      overflow-wrap: break-word;
    }
    
    .news-actions {
      display: flex;
      align-items: center;
      font-size: 0.9rem;
    }
    
    .news-action-buttons {
      display: flex;
      gap: 4px;
      
      .q-btn {
        padding: 8px !important;
        min-width: 44px;
        min-height: 44px;
      }
    }
  }
  
  // Фильтры
  .modern-card {
    .row.q-gutter-sm {
      .col-12 {
        margin-bottom: 8px;
      }
    }
  }
  
  // Статистика
  .status-card {
    .q-card-section {
      padding: 12px !important;
      
      .text-caption {
        font-size: 0.75rem !important;
      }
      
      .row {
        flex-direction: column !important;
        gap: 8px;
        
        .col, .col-auto {
          width: 100%;
        }
      }
    }
  }
}

@media (max-width: 400px) {
  .mobile-card-section {
    padding: 10px !important;
  }
  
  .news-card {
    .q-img {
      height: 150px !important;
    }
    
    .news-title {
      font-size: 0.95rem !important;
    }
    
    .news-description {
      font-size: 0.8rem !important;
    }
  }
  
  .mobile-news-meta {
    font-size: 0.75rem !important;
    
    .country-flag {
      font-size: 1rem !important;
    }
  }
}

// Стили для диалога просмотра новости
.dialog-news-section {
  padding: 0 !important;
}

.dialog-image-wrapper {
  width: 100%;
  margin-bottom: 20px;
  
  .dialog-news-image {
    width: 100%;
    border-radius: 0;
  }
}

// Ограничение размера изображения в диалоге для десктопа
@media (min-width: 600px) {
  .dialog-image-wrapper {
    max-width: 800px;
    max-height: 450px;
    margin: 0 auto 20px auto;
    overflow: hidden;
    
    .dialog-news-image {
      max-height: 450px;
      object-fit: contain;
      width: 100%;
    }
  }
}

.dialog-news-content {
  padding: 20px;
  
  .news-meta {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 16px;
  }
  
  .news-title {
    font-size: 1.4rem;
    line-height: 1.5;
    font-weight: 600;
    margin-bottom: 16px;
    word-break: break-word;
    overflow-wrap: break-word;
  }
  
  .news-description {
    font-size: 1rem;
    line-height: 1.6;
    margin-bottom: 16px;
    word-break: break-word;
    overflow-wrap: break-word;
  }
  
  .news-content-text {
    font-size: 0.95rem;
    line-height: 1.7;
    word-break: break-word;
    overflow-wrap: break-word;
  }
}

// Адаптация диалога просмотра новости для мобильных
@media (max-width: 599px) {
  .q-dialog .q-card {
    margin: 0 !important;
    max-width: 100% !important;
    border-radius: 0 !important;
  }
  
  .dialog-news-content {
    padding: 16px !important;
    
    .news-meta {
      font-size: 0.875rem;
      gap: 6px;
      
      .country-flag {
        font-size: 1rem;
      }
      
      .source-name {
        font-size: 0.9rem;
        font-weight: 600;
      }
    }
    
    .news-title {
      font-size: 1.2rem !important;
      line-height: 1.4 !important;
      margin-bottom: 12px !important;
    }
    
    .news-description {
      font-size: 0.95rem !important;
      line-height: 1.6 !important;
      margin-bottom: 12px !important;
    }
    
    .news-content-text {
      font-size: 0.9rem !important;
      line-height: 1.6 !important;
    }
    
    .q-chip {
      font-size: 0.8rem !important;
      height: 28px !important;
    }
  }
}

@media (max-width: 400px) {
  .dialog-news-content {
    padding: 12px !important;
    
    .news-title {
      font-size: 1.1rem !important;
    }
    
    .news-description {
      font-size: 0.9rem !important;
    }
    
    .news-content-text {
      font-size: 0.85rem !important;
    }
  }
}

// === СТИЛЬНЫЙ КАЛЕНДАРЬ ===
.date-picker-dialog {
  .q-dialog__inner {
    padding: 16px;
  }
}

.date-picker-card {
  max-width: 420px !important;
  width: 100% !important;
  background: var(--bg-card) !important;
  border-radius: 24px !important;
  box-shadow: var(--shadow-2xl) !important;
  border: 1px solid var(--border-primary) !important;
  overflow: hidden !important;
}

.date-picker-header {
  background: var(--gradient-primary) !important;
  color: white !important;
  padding: 20px 24px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: space-between !important;
}

.date-picker-title {
  display: flex;
  align-items: center;
  font-size: 1.25rem;
  font-weight: 600;
  color: white !important;
  
  .q-icon {
    color: white !important;
  }
}

.close-btn {
  color: white !important;
  
  &:hover {
    background: rgba(255, 255, 255, 0.1) !important;
  }
}

.date-picker-content {
  padding: 24px !important;
}

.date-type-selector {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  background: var(--bg-secondary);
  padding: 6px;
  border-radius: 12px;
}

.date-type-btn {
  flex: 1;
  color: var(--text-secondary) !important;
  font-weight: 500 !important;
  text-transform: none !important;
  border-radius: 8px !important;
  transition: all 0.3s ease !important;
  padding: 10px 16px !important;
  
  .q-icon {
    color: var(--text-secondary) !important;
    margin-right: 8px;
  }
  
  &:hover {
    background: var(--bg-tertiary) !important;
  }
  
  &.active {
    background: var(--gradient-primary) !important;
    color: white !important;
    box-shadow: var(--shadow-md) !important;
    
    .q-icon {
      color: white !important;
    }
  }
}

.calendar-wrapper {
  display: flex;
  justify-content: center;
}

.stylish-calendar {
  width: 100% !important;
  box-shadow: none !important;
  border: none !important;
  
  :deep(.q-date__header) {
    background: transparent !important;
    color: var(--text-primary) !important;
    padding: 12px !important;
    border-bottom: 1px solid var(--border-primary) !important;
    margin-bottom: 12px !important;
  }
  
  :deep(.q-date__view) {
    padding: 8px !important;
  }
  
  :deep(.q-date__calendar) {
    padding: 0 !important;
  }
  
  :deep(.q-date__calendar-item) {
    button {
      border-radius: 12px !important;
      font-weight: 500 !important;
      transition: all 0.2s ease !important;
      
      &:hover {
        background: var(--bg-tertiary) !important;
        transform: scale(1.05) !important;
      }
    }
  }
  
  :deep(.q-date__calendar-item--in) {
    button {
      color: var(--text-primary) !important;
    }
  }
  
  :deep(.q-date__calendar-item--out) {
    button {
      color: var(--text-tertiary) !important;
      opacity: 0.5 !important;
    }
  }
  
  :deep(.q-date__today) {
    box-shadow: 0 0 0 2px var(--primary-color) inset !important;
    font-weight: 700 !important;
  }
  
  :deep(.q-date__range) {
    background: rgba(99, 102, 241, 0.1) !important;
  }
  
  :deep(.q-date__range-from),
  :deep(.q-date__range-to) {
    background: var(--gradient-primary) !important;
    color: white !important;
    font-weight: 700 !important;
    box-shadow: var(--shadow-md) !important;
  }
  
  :deep(.q-date__calendar-weekdays) {
    color: var(--text-tertiary) !important;
    font-weight: 600 !important;
    text-transform: uppercase !important;
    font-size: 0.75rem !important;
    letter-spacing: 0.5px !important;
    padding: 8px 0 !important;
  }
  
  :deep(.q-btn--flat) {
    &:before {
      display: none !important;
    }
  }
}

.date-picker-actions {
  padding: 16px 24px !important;
  background: var(--bg-secondary) !important;
  border-top: 1px solid var(--border-primary) !important;
  display: flex !important;
  justify-content: flex-end !important;
  gap: 12px !important;
}

.action-btn {
  text-transform: none !important;
  font-weight: 600 !important;
  padding: 10px 24px !important;
  border-radius: 12px !important;
  transition: all 0.3s ease !important;
  
  &:before {
    display: none !important;
  }
}

.cancel-btn {
  color: var(--text-secondary) !important;
  
  &:hover {
    background: var(--bg-tertiary) !important;
    color: var(--text-primary) !important;
  }
}

.apply-btn {
  background: var(--gradient-primary) !important;
  color: white !important;
  box-shadow: var(--shadow-md) !important;
  
  &:hover {
    box-shadow: var(--shadow-lg) !important;
    transform: translateY(-1px) !important;
  }
  
  &:disabled {
    opacity: 0.5 !important;
    cursor: not-allowed !important;
    transform: none !important;
  }
}

// Мобильная адаптация календаря
@media (max-width: 599px) {
  .date-picker-card {
    max-width: 100% !important;
    margin: 0 !important;
    border-radius: 20px !important;
  }
  
  .date-picker-header {
    padding: 16px 20px !important;
  }
  
  .date-picker-title {
    font-size: 1.1rem !important;
  }
  
  .date-picker-content {
    padding: 20px !important;
  }
  
  .date-type-selector {
    margin-bottom: 20px;
  }
  
  .date-type-btn {
    font-size: 0.9rem !important;
    padding: 8px 12px !important;
    
    .q-icon {
      font-size: 1rem !important;
    }
  }
  
  .stylish-calendar {
    :deep(.q-date__calendar-item) {
      button {
        font-size: 0.9rem !important;
        min-height: 36px !important;
        min-width: 36px !important;
      }
    }
  }
  
  .date-picker-actions {
    padding: 12px 20px !important;
  }
  
  .action-btn {
    padding: 8px 20px !important;
    font-size: 0.9rem !important;
  }
}

@media (max-width: 400px) {
  .date-picker-header {
    padding: 12px 16px !important;
  }
  
  .date-picker-title {
    font-size: 1rem !important;
    
    .q-icon {
      font-size: 20px !important;
    }
  }
  
  .date-picker-content {
    padding: 16px !important;
  }
  
  .stylish-calendar {
    :deep(.q-date__calendar-item) {
      button {
        font-size: 0.85rem !important;
        min-height: 32px !important;
        min-width: 32px !important;
      }
    }
  }
}
</style>
