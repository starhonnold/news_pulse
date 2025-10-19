<template>
  <q-page class="news-page">
    <!-- Заголовок страницы -->
    <div class="page-header">
      <div class="header-content">
        <div class="title-section">
        </div>
      </div>
    </div>

    <!-- Поисковая панель -->
    <div class="search-section">
      <div class="search-panel">
        <q-input
          v-model="searchQuery"
          placeholder="Поиск новостей..."
          class="search-field"
          @keyup.enter="performSearch"
          @input="performSearch"
        />
        <div class="search-controls">
          <q-btn flat round icon="tune" @click="showFiltersDialog = true" />
          <q-btn 
            flat 
            round 
            :icon="sortOrder === 'desc' ? 'keyboard_arrow_down' : 'keyboard_arrow_up'" 
            @click="toggleSortOrder"
            class="sort-btn"
          />
          <span class="news-counter">{{ totalNews }} новостей</span>
        </div>
      </div>
    </div>

    <!-- Индикатор загрузки -->
    <div v-if="loading" class="loading-section">
      <q-spinner-dots color="primary" size="40px" />
      <p class="loading-text">Загружаем новости...</p>
    </div>

    <!-- Список новостей -->
    <div v-else class="news-feed">
          <!-- Сообщение если новостей нет -->
      <div v-if="paginatedNews.length === 0" class="empty-state">
        <q-icon name="article" size="64px" class="empty-icon" />
        <h3 class="empty-title">Новости не найдены</h3>
        <p class="empty-description">Попробуйте изменить фильтры или поисковый запрос</p>
          </div>

      <!-- Список новостей с infinite scroll -->
      <q-infinite-scroll @load="loadMoreNews" :offset="250">
        <transition-group name="news-list" tag="div" class="news-list" appear>
            <div 
            v-for="(news, index) in paginatedNews" 
              :key="news.id"
            :style="{ '--animation-delay': `${index * 0.1}s` }"
            class="news-item"
                @click="openNews(news)"
              >
            <!-- Полоска слева -->
            <div class="news-accent-bar"></div>
            
            <!-- Контент карточки -->
            <div class="news-content">
                <!-- Изображение новости -->
                <div v-if="isValidImageUrl(news.image_url || news.image)" class="news-image-wrapper">
                  <q-img
                    :src="news.image_url || news.image"
                    :ratio="16/9"
                    class="news-image"
                    fit="cover"
                  loading="lazy"
                  >
                    <template v-slot:error>
                      <div class="absolute-full flex flex-center bg-grey-3">
                        <q-icon name="image" size="lg" color="grey-6" />
                      </div>
                    </template>
                  </q-img>
                </div>

              <!-- Информация о новости -->
              <div class="news-info">
                      <!-- Мета информация -->
                <div class="news-meta">
                  <div class="news-source">
                    <span class="country-flag">{{ news.country?.flag_emoji || news.country?.flag || '🌍' }}</span>
                    <span class="source-name">{{ news.source?.name || news.source_name || 'Неизвестный источник' }}</span>
                          </div>
                  <div class="news-date">{{ formatDate(news.published_at) }}</div>
                        </div>

                <!-- Заголовок -->
                <h3 class="news-title">{{ cleanText(news.title) }}</h3>

                <!-- Описание -->
                <p class="news-description">{{ cleanText(news.description) }}</p>

                <!-- Категория и действия -->
                <div class="news-category">
                          <q-chip
                            v-if="news.category && news.category.name"
                            :style="{
                              background: `linear-gradient(135deg, ${getCategoryColor(news.category.color)} 0%, ${lightenColor(news.category.color, 20)} 100%) !important`,
                              color: 'white !important',
                              border: 'none !important'
                            }"
                    class="category-chip"
                          >
                    <q-icon :name="getCategoryIcon(news.category.icon || news.category.slug)" class="category-icon" />
                            {{ news.category.name }}
                          </q-chip>
                          
                          <!-- Действия для мобильной версии -->
                          <div class="news-actions-mobile">
                            <q-btn
                              flat
                              round
                              dense
                              icon="share"
                              size="sm"
                              @click.stop="shareNews(news)"
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

                <!-- Действия для десктопа -->
                      <div class="news-actions">
                      <div class="news-action-buttons">
                        <q-btn
                          flat
                          round
                          dense
                          icon="share"
                          size="sm"
                          @click.stop="shareNews(news)"
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
            </div>

            </div>
        </transition-group>

          <template v-slot:loading>
          <div class="loading-more">
            <q-spinner-dots color="primary" size="32px" />
            </div>
          </template>
        </q-infinite-scroll>
      </div>

    <!-- Диалог с фильтрами -->
    <q-dialog v-model="showFiltersDialog" position="right" :maximized="$q.platform.is.mobile">
      <q-card class="filters-dialog-card" :style="$q.platform.is.mobile ? '' : 'width: 400px'">
        <q-card-section class="row items-center q-pb-none filters-dialog-header">
          <div class="text-h6">
            <q-icon name="tune" class="q-mr-sm" />
            Фильтры
    </div>
          <q-space />
          <q-btn icon="close" flat round dense v-close-popup />
        </q-card-section>

        <q-card-section class="filters-dialog-content">
          <!-- Фильтр по категориям -->
          <div class="filter-item q-mb-md">
            <q-select
              v-model="selectedCategories"
              :options="categoryOptions"
              label="Категории"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="filter-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCategoriesClear"
            />
          </div>

          <!-- Фильтр по странам -->
          <div class="filter-item q-mb-md">
            <q-select
              v-model="selectedCountries"
              :options="countryOptions"
              label="Страны"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="filter-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCountriesClear"
            />
          </div>

          <!-- Фильтр по дате -->
          <div class="filter-item q-mb-md">
            <q-input
              v-model="dateRangeText"
              label="Период"
              dense
              outlined
              readonly
              class="filter-input date-input"
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
        </q-card-section>

        <q-card-actions align="between" class="filters-dialog-actions">
          <q-btn
            unelevated
            label="Сбросить все"
            color="negative"
            @click="clearAllFilters"
            icon="refresh"
            class="reset-btn"
          />
          <q-btn
            unelevated
            label="Применить"
            color="primary"
            @click="applyFiltersAndClose"
            class="gradient-btn"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>

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
                color="primary"
                label="Закрыть"
                @click="showNewsDialog = false"
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
            v-close-popup
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
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
const allNews = ref([])
const loading = ref(false)

// Фильтр по дате
const showDatePicker = ref(false)
const showFiltersDialog = ref(false)
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


const hasDateSelection = computed(() => {
  if (dateFilterType.value === 'single') {
    return selectedDate.value !== ''
  } else if (dateFilterType.value === 'range') {
    return dateRange.value.from && dateRange.value.to
  }
  return false
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
    console.log('Current filters:', {
      searchQuery: searchQuery.value,
      selectedCategories: selectedCategories.value,
      selectedCountries: selectedCountries.value,
      dateRangeText: dateRangeText.value
    })
    
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
    
    // Добавляем параметры даты
    if (dateFilterType.value === 'single' && selectedDate.value) {
      // Для одной даты устанавливаем и date_from и date_to на одну и ту же дату
      const dateStr = selectedDate.value.replace(/\//g, '-')
      params.date_from = dateStr
      params.date_to = dateStr
    } else if (dateFilterType.value === 'range' && dateRange.value.from && dateRange.value.to) {
      // Для периода устанавливаем date_from и date_to
      params.date_from = dateRange.value.from.replace(/\//g, '-')
      params.date_to = dateRange.value.to.replace(/\//g, '-')
    }
    
    console.log('Final API params:', params)
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
    console.log('Фильтры:', { 
      search: searchQuery.value, 
      categories: selectedCategories.value, 
      countries: selectedCountries.value,
      dateFilterType: dateFilterType.value,
      selectedDate: selectedDate.value,
      dateRange: dateRange.value
    })
    console.log('API параметры:', params)
    
    if (reset) {
      await loadFilters()
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

const performSearch = async () => {
  // Немедленный поиск для мобильной версии
  await loadNews(1, true)
}

const onFilterChange = async () => {
  console.log('onFilterChange called with filters:', {
    searchQuery: searchQuery.value,
    selectedCategories: selectedCategories.value,
    selectedCountries: selectedCountries.value,
    dateRangeText: dateRangeText.value
  })
  // При изменении фильтров перезагружаем новости
  await loadNews(1, true)
}

const toggleSortOrder = async () => {
  // Переключаем порядок сортировки
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  // Перезагружаем новости с новым порядком сортировки
  await loadNews(1, true)
}

const clearAllFilters = () => {
  alert('Кнопка "Сбросить все" работает!')
  console.log('Сбрасываем все фильтры...')
  
  // Очищаем все фильтры
  searchQuery.value = ''
  selectedCategories.value = []
  selectedCountries.value = []
  selectedDate.value = ''
  dateRange.value = { from: '', to: '' }
  dateRangeText.value = ''
  sortOrder.value = 'desc'
  
  // Закрываем диалог
  showFiltersDialog.value = false
  
  // Перезагружаем новости
  loadNews(1, true)
}

const applyFiltersAndClose = () => {
  showFiltersDialog.value = false
  onFilterChange()
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
  console.log('Single date selected:', val)
  selectedDate.value = val
}

const onDateRangeSelect = (val) => {
  console.log('Date range selected:', val)
  dateRange.value = val
}

const applyDateFilter = () => {
  console.log('applyDateFilter called:', {
    dateFilterType: dateFilterType.value,
    selectedDate: selectedDate.value,
    dateRange: dateRange.value
  })
  
  if (dateFilterType.value === 'single' && selectedDate.value) {
    const formattedDate = formatDateForDisplay(selectedDate.value)
    dateRangeText.value = formattedDate
    console.log('Single date applied:', formattedDate)
  } else if (dateFilterType.value === 'range' && dateRange.value.from && dateRange.value.to) {
    const from = formatDateForDisplay(dateRange.value.from)
    const to = formatDateForDisplay(dateRange.value.to)
    dateRangeText.value = `${from} - ${to}`
    console.log('Date range applied:', `${from} - ${to}`)
  }
  
  console.log('Final dateRangeText:', dateRangeText.value)
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
  
  // Quasar возвращает дату в формате YYYY/MM/DD
  // Преобразуем в формат DD.MM.YYYY для отображения
  const parts = dateStr.split('/')
  if (parts.length === 3) {
    const [year, month, day] = parts
    return `${day}.${month}.${year}`
  }
  
  // Fallback для других форматов
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
// === ОСНОВНЫЕ СТИЛИ СТРАНИЦЫ ===
.news-page {
  padding: var(--spacing-xl);
  background: var(--bg-main);
  min-height: 100vh;
  width: 100%;
  display: flex;
  flex-direction: column;
}

// === ЗАГОЛОВОК СТРАНИЦЫ ===
.page-header {
  margin-bottom: var(--spacing-lg);
  width: 100%;
  
  .header-content {
    text-align: center;
    margin-bottom: var(--spacing-md);
  }
}



// === ЗАГРУЗКА ===
.loading-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--text-secondary);
  
  .loading-text {
    margin-top: var(--spacing-md);
    font-size: var(--font-size-base);
  }
}

.loading-more {
  display: flex;
  justify-content: center;
  padding: var(--spacing-xl);
}

// === ПУСТОЕ СОСТОЯНИЕ ===
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  text-align: center;
  
  .empty-icon {
    color: var(--text-tertiary);
    margin-bottom: var(--spacing-lg);
  }
  
  .empty-title {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-sm);
  }
  
  .empty-description {
    font-size: var(--font-size-base);
    color: var(--text-secondary);
  }
}

// === ЛЕНТА НОВОСТЕЙ ===
.news-feed {
  width: 100%;
  margin: 0 auto;
}

.news-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

// === КАРТОЧКА НОВОСТИ ===
.news-item {
  position: relative;
  background: var(--bg-elevated);
  border-radius: 16px;
  border: 1px solid var(--border-primary);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  overflow: hidden;
  margin-bottom: var(--spacing-lg);
  backdrop-filter: blur(10px);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    border-color: var(--accent-color);
  }
}

.news-accent-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background: linear-gradient(180deg, var(--accent-color) 0%, rgba(245, 158, 11, 0.7) 100%);
  border-radius: 0 2px 2px 0;
  opacity: 0.9;
}

.news-content {
  display: flex;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  align-items: flex-start;
}

.news-page .news-image-wrapper {
  flex-shrink: 0 !important;
  width: 160px !important;
  height: 100px !important;
  border-radius: var(--radius-md) !important;
  overflow: hidden !important;
  position: relative !important;
  background: var(--bg-secondary) !important;
  max-width: 160px !important;
  max-height: 100px !important;
  
  .news-image {
    width: 100% !important;
    height: 100% !important;
    object-fit: cover !important;
    transition: transform var(--transition-normal) !important;
    max-width: 160px !important;
    max-height: 100px !important;
  }
  
  .no-image {
    display: flex !important;
    align-items: center !important;
    justify-content: center !important;
    width: 100% !important;
    height: 100% !important;
    background: var(--bg-secondary) !important;
    color: var(--text-tertiary) !important;
    font-size: var(--font-size-sm) !important;
    max-width: 160px !important;
    max-height: 100px !important;
  }
}

// Дополнительные принудительные стили
.news-item .news-image-wrapper {
  width: 160px !important;
  height: 100px !important;
  max-width: 160px !important;
  max-height: 100px !important;
}

.news-content .news-image-wrapper {
  width: 160px !important;
  height: 100px !important;
  max-width: 160px !important;
  max-height: 100px !important;
}

.news-item:hover .news-image {
  transform: scale(1.05);
}

.news-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  min-height: 100px;
}

.news-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin-bottom: var(--spacing-sm);
  
  .news-source {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);

    .country-flag {
      font-size: 1.2em;
    }

    .source-name {
      font-weight: 600;
      color: var(--accent-color);
      font-size: var(--font-size-sm);
    }
  }
  
  .news-date {
    color: var(--text-tertiary);
    font-size: var(--font-size-sm);
    font-weight: 500;
  }
}

.news-title {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--text-primary);
  line-height: var(--line-height-tight);
  margin: 0 0 var(--spacing-sm) 0;
  flex: 1;
  letter-spacing: -0.01em;
}

.news-description {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  line-height: var(--line-height-relaxed);
  margin: 0 0 var(--spacing-sm) 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
  font-weight: 500;
}

.news-category {
  margin-top: auto;
  margin-bottom: var(--spacing-sm);
  
  .news-actions-mobile {
    display: none; // Скрываем мобильные действия в десктопной версии
  }
}

.category-chip {
  font-size: var(--font-size-xs) !important;
  font-weight: 700 !important;
  padding: 6px 12px !important;
  border-radius: 16px !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15) !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
  border: none !important;
  
  &:hover {
    transform: translateY(-1px) scale(1.02) !important;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2) !important;
  }
  
  .category-icon {
    color: white !important;
    font-size: 1rem !important;
    margin-right: 4px !important;
  }
}

.news-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-top: auto;
  padding-top: var(--spacing-sm);
  border-top: 1px solid var(--border-primary);
}

.news-action-buttons {
  display: flex;
  gap: var(--spacing-xs);
  
  .q-btn {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    min-height: 36px;
    min-width: 36px;
    border-radius: 8px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
    
    &:hover {
      background: var(--bg-tertiary) !important;
      transform: translateY(-1px) scale(1.05);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    }
    
    &:active {
      transform: translateY(0) scale(0.95);
    }
  }
}


// === АНИМАЦИИ ===
.news-list-enter-active {
  animation: newsSlideIn 0.6s ease-out;
  animation-delay: var(--animation-delay, 0s);
  animation-fill-mode: both;
}

@keyframes newsSlideIn {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// === АДАПТИВНОСТЬ ===

// Скрытие элементов статистики в мобильной версии
@media (max-width: 768px) {
  .header-actions .unified-search-panel {
    .stats-count,
    .stats-update,
    .stats-actions {
      display: none !important;
    }
  }
}

@media (max-width: 1024px) {
  .news-page {
    padding: var(--spacing-lg);
    max-width: 100%;
  }
  
  .page-header {
    margin-bottom: var(--spacing-xl);
    max-width: 100%;
    
    .header-content {
      text-align: center;
      margin-bottom: var(--spacing-lg);
      
    }
    
    .header-actions {
      .search-filters-row {
        flex-direction: column;
        gap: var(--spacing-md);
        max-width: 100%;
        
        .search-input {
          width: 100%;
          min-width: unset;
          
          .q-field__control {
            min-height: 48px !important;
            height: 48px !important;
          }
        }
        
        .filters-icon-text {
          align-self: center;
          min-width: 140px;
        }
      }
    }
  }
  
  .stats-content {
    flex-direction: column;
    gap: var(--spacing-lg);
    text-align: center;
    max-width: 100%;
    padding: var(--spacing-lg);
    
    .stats-info {
      flex-direction: column;
      gap: var(--spacing-sm);
      
      .stats-text {
        font-size: var(--font-size-base);
      }
    }
    
    .stats-actions {
      justify-content: center;
    }
  }
  
  .news-feed {
    max-width: 100%;
  }
  
  .news-content {
    flex-direction: column;
    gap: var(--spacing-lg);
    padding: var(--spacing-lg);
  }
  
  .news-image-wrapper {
    width: 100%;
    height: 200px;
    border-radius: var(--radius-lg);
  }
  
  .news-info {
    min-height: auto;
  }
}

// === МОБИЛЬНЫЕ УЛУЧШЕНИЯ UX/UI ===
@media (max-width: 768px) {
  .news-page {
    padding: var(--spacing-sm);
    background: #0E1621;
  }
  
  .page-header {
    margin-bottom: var(--spacing-sm);
    position: sticky;
    top: 0;
    z-index: 100;
    background: rgba(14, 22, 33, 0.95);
    backdrop-filter: blur(10px);
    padding: var(--spacing-sm) 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    
    .header-content {
      display: none; // Скрываем заголовок на мобильных
    }
    
    .header-actions {
      .search-filters-row {
        display: none; // Скрываем поиск и фильтры на мобильных
      }
    }
  }
  
  // Универсальная панель поиска уже показана в основных стилях
  
// === ПОИСКОВАЯ ПАНЕЛЬ ===
.search-section {
  width: 100%;
  margin: 20px 0;
  padding: 0 20px;
}

.search-panel {
  display: flex;
  align-items: center;
  background: var(--bg-elevated);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  padding: 12px 16px;
  gap: 12px;
  transition: all 0.3s ease;
}

.search-panel:focus-within {
  border-color: var(--accent-color);
  box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.1);
}


.search-field {
  flex: 1;
}

.search-field :deep(.q-field__control) {
  background: transparent;
  border: none;
  box-shadow: none;
}

.search-field :deep(.q-field__native) {
  color: var(--text-primary);
  font-size: 16px;
}

.search-field :deep(.q-field__native::placeholder) {
  color: var(--text-tertiary);
}

.search-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-controls .q-btn {
  color: var(--accent-color);
}

.sort-btn {
  transition: all 0.3s ease;
}

.sort-btn:hover {
  color: var(--primary-light) !important;
  transform: scale(1.1);
}

.clear-filters-btn {
  transition: all 0.3s ease;
  color: #ff6b6b !important;
}

.clear-filters-btn:hover {
  color: #ff5252 !important;
  transform: scale(1.1);
  background: rgba(255, 107, 107, 0.1) !important;
}

.news-counter {
  color: var(--accent-color);
  font-size: 14px;
  font-weight: 500;
}

// Принудительные стили для десктопной версии
@media (min-width: 769px) {
  .news-counter {
    color: #00d1c1 !important;
    font-size: 14px !important;
    font-weight: 600 !important;
    background: none !important;
    padding: 0 !important;
    border: none !important;
    border-radius: 0 !important;
    text-shadow: none !important;
    transition: none !important;
  }
  
  .news-counter:hover {
    color: #00d1c1 !important;
    background: none !important;
    border: none !important;
    transform: none !important;
    box-shadow: none !important;
  }
}

// Принудительные стили с :deep() для Vue scoped
@media (min-width: 769px) {
  :deep(.news-counter) {
    color: #00d1c1 !important;
    font-size: 14px !important;
    font-weight: 600 !important;
    background: none !important;
    padding: 0 !important;
    border: none !important;
    border-radius: 0 !important;
    text-shadow: none !important;
    transition: none !important;
  }
  
  :deep(.news-counter:hover) {
    color: #00d1c1 !important;
    background: none !important;
    border: none !important;
    transform: none !important;
    box-shadow: none !important;
  }
}

// Десктопная версия - выделяем счетчик новостей
@media (min-width: 769px) {
  .search-controls .news-counter {
    color: #00d1c1 !important;
    font-size: 14px !important;
    font-weight: 600 !important;
    background: none !important;
    padding: 0 !important;
    border: none !important;
    border-radius: 0 !important;
    text-shadow: none !important;
    transition: none !important;
  }
  
  .search-controls .news-counter:hover {
    color: #00d1c1 !important;
    background: none !important;
    border: none !important;
    transform: none !important;
    box-shadow: none !important;
  }
}

// Альтернативный селектор для большей специфичности
@media (min-width: 769px) {
  .news-page .search-controls .news-counter {
    color: #00d1c1 !important;
    font-size: 14px !important;
    font-weight: 600 !important;
    background: none !important;
    padding: 0 !important;
    border: none !important;
    border-radius: 0 !important;
    text-shadow: none !important;
    transition: none !important;
  }
  
  .news-page .search-controls .news-counter:hover {
    color: #00d1c1 !important;
    background: none !important;
    border: none !important;
    transform: none !important;
    box-shadow: none !important;
  }
}

/* Мобильная адаптация */
@media (max-width: 768px) {
  .search-section {
    padding: 0 12px;
    margin: 12px 0;
  }
  
  .search-panel {
    flex-direction: column;
    align-items: stretch;
    padding: 8px;
    gap: 8px;
    border-radius: 8px;
  }
  
  .search-field :deep(.q-field__control) {
    min-height: 28px !important;
  }
  
  .search-field :deep(.q-field__native) {
    font-size: 14px !important;
    padding: 4px 10px !important;
    line-height: 1.2 !important;
  }
  
  .search-controls {
    justify-content: flex-start;
    gap: 4px;
  }
  
  .search-controls .q-btn {
    min-width: 28px !important;
    min-height: 28px !important;
    padding: 4px !important;
  }
  
  .search-controls .q-btn .q-icon {
    font-size: 16px !important;
  }
  
  .clear-filters-btn {
    color: #ff6b6b !important;
  }
  
  .clear-filters-btn:hover {
    color: #ff5252 !important;
    background: rgba(255, 107, 107, 0.1) !important;
  }
  
  .search-controls .q-btn:active {
    transform: scale(0.95);
    transition: transform 0.1s ease;
  }
  
  .news-counter {
    font-size: 11px;
    font-weight: 600;
    color: #00d1c1;
  }
  
  // Улучшения для touch-интерфейса
  .search-field :deep(.q-field__control) {
    touch-action: manipulation;
  }
  
  .search-panel {
    touch-action: manipulation;
  }
}

/* Дополнительная адаптация для средних мобильных экранов */
@media (max-width: 600px) {
  .search-section {
    padding: 0 10px;
    margin: 10px 0;
  }
  
  .search-panel {
    padding: 7px;
    gap: 7px;
    border-radius: 7px;
  }
  
  .search-field :deep(.q-field__control) {
    min-height: 26px !important;
  }
  
  .search-field :deep(.q-field__native) {
    font-size: 13px !important;
    padding: 3px 9px !important;
    line-height: 1.2 !important;
  }
  
  .search-controls {
    justify-content: flex-start;
    gap: 3px;
  }
  
  .search-controls .q-btn {
    min-width: 26px !important;
    min-height: 26px !important;
    padding: 3px !important;
  }
  
  .search-controls .q-btn .q-icon {
    font-size: 15px !important;
  }
  
  .clear-filters-btn {
    color: #ff6b6b !important;
  }
  
  .clear-filters-btn:hover {
    color: #ff5252 !important;
    background: rgba(255, 107, 107, 0.1) !important;
  }
  
  .news-counter {
    font-size: 10px;
  }
}
  
  .stats-content {
    display: none; // Скрываем статистику на мобильных
  }
  
  // Компактная статистика внизу при прокрутке
  .mobile-stats-floating {
    position: fixed;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(28, 37, 51, 0.95);
    backdrop-filter: blur(10px);
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: 20px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    color: #00E0FF;
    font-size: 12px;
    font-weight: 600;
    z-index: 50;
    opacity: 0.8;
    transition: opacity 0.3s ease;
    
    &:hover {
      opacity: 1;
    }
  }
  
  .news-item {
    min-height: 100px;
    border-radius: 14px;
    background: #1C2533;
    border: none;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
    margin-bottom: var(--spacing-md);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    animation: slideUpFadeIn 0.6s ease-out;
    
    &:active {
      transform: scale(0.98);
      transition: transform 0.1s ease;
    }
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 8px 30px rgba(0, 0, 0, 0.3);
    }
  }
  
  .news-content {
    flex-direction: row;
    gap: var(--spacing-md);
    padding: var(--spacing-md);
    align-items: flex-start;
  }
  
  .news-image-wrapper {
    width: 80px !important;
    height: 80px !important;
    border-radius: 12px;
    flex-shrink: 0;
    overflow: hidden;
    
    .news-image {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
  
  .news-info {
    flex: 1;
    min-height: 80px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }
  
  .news-meta {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-xs);
    
    .news-source {
      .source-name {
        font-size: 12px;
        color: #FFA726;
        font-weight: 600;
      }
      
      .country-flag {
        font-size: 14px;
      }
    }
    
    .news-date {
      font-size: 11px;
      color: rgba(255, 255, 255, 0.6);
    }
  }
  
  .news-title {
    font-size: 16px;
    font-weight: 600;
    line-height: 1.4;
    color: #ffffff;
    margin-bottom: var(--spacing-xs);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .news-description {
    display: none; // Скрываем описание на мобильных для компактности
  }
  
  .news-category {
    margin-top: auto;
    margin-bottom: var(--spacing-xs);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-xs);
    
    .category-chip {
      font-size: 8px !important;
      padding: 2px 6px !important;
      border-radius: 8px !important;
      font-weight: 600 !important;
      min-height: 20px !important;
      
      .category-icon {
        font-size: 10px !important;
        margin-right: 2px !important;
      }
    }
    
    .news-actions-mobile {
      display: flex;
      gap: var(--spacing-xs);
      
      .q-btn {
        min-height: 20px;
        min-width: 20px;
        border-radius: 3px;
        background: rgba(255, 255, 255, 0.1);
        color: rgba(255, 255, 255, 0.7);
        
        .q-icon {
          font-size: 10px !important;
        }
        
        &:active {
          transform: scale(0.9);
          background: rgba(255, 255, 255, 0.2);
        }
      }
    }
  }
  
  .news-actions {
    display: none; // Скрываем десктопные действия в мобильной версии
  }
  
  
  // Анимация появления карточек
  @keyframes slideUpFadeIn {
    from {
      opacity: 0;
      transform: translateY(30px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
}

// === ДОПОЛНИТЕЛЬНЫЕ УЛУЧШЕНИЯ ДЛЯ МАЛЕНЬКИХ ЭКРАНОВ ===
@media (max-width: 480px) {
  .news-page {
    padding: var(--spacing-xs);
    background: #0E1621;
  }
  
  .page-header {
    margin-bottom: var(--spacing-xs);
    padding: var(--spacing-xs) 0;
  }
  
  // Дополнительная адаптация поисковой панели для маленьких экранов
  .search-section {
    padding: 0 8px;
    margin: 8px 0;
  }
  
  .search-panel {
    padding: 6px;
    gap: 6px;
    border-radius: 6px;
  }
  
  .search-field :deep(.q-field__control) {
    min-height: 24px !important;
  }
  
  .search-field :deep(.q-field__native) {
    font-size: 12px !important;
    padding: 2px 8px !important;
    line-height: 1.2 !important;
  }
  
  .search-controls {
    justify-content: flex-start;
    gap: 3px;
  }
  
  .search-controls .q-btn {
    min-width: 24px !important;
    min-height: 24px !important;
    padding: 2px !important;
  }
  
  .search-controls .q-btn .q-icon {
    font-size: 14px !important;
  }
  
  .clear-filters-btn {
    color: #ff6b6b !important;
  }
  
  .clear-filters-btn:hover {
    color: #ff5252 !important;
    background: rgba(255, 107, 107, 0.1) !important;
  }
  
  .news-counter {
    font-size: 9px;
  }
  
  .mobile-search-bar {
    padding: var(--spacing-xs) var(--spacing-sm);
    margin-bottom: var(--spacing-sm);
    gap: var(--spacing-xs);
    
    .search-icon, .filters-icon {
      font-size: 18px;
    }
    
    .mobile-search-input {
      .q-field__control {
        min-height: 32px !important;
      }
      
      .q-field__native {
        font-size: 13px !important;
        padding: 6px 10px !important;
      }
      
      .q-field__label {
        font-size: 13px !important;
      }
    }
    
    .mobile-stats-text {
      font-size: 11px;
    }
    
    .filters-icon {
      &:hover {
        color: #FFB74D;
        transform: scale(1.05);
      }
    }
  }
  
  .news-item {
    min-height: 90px;
    margin-bottom: var(--spacing-sm);
    border-radius: 12px;
  }
  
  .news-content {
    padding: var(--spacing-sm);
    gap: var(--spacing-sm);
  }
  
  .news-image-wrapper {
    width: 70px !important;
    height: 70px !important;
    border-radius: 10px;
  }
  
  .news-info {
    min-height: 70px;
  }
  
  .news-title {
    font-size: 15px;
    line-height: 1.3;
    -webkit-line-clamp: 2;
    line-clamp: 2;
  }
  
  .news-meta {
    .news-source {
      .source-name {
        font-size: 11px;
      }
      
      .country-flag {
        font-size: 12px;
      }
    }
    
    .news-date {
      font-size: 10px;
    }
  }
  
  .news-category {
    .category-chip {
      font-size: 7px !important;
      padding: 1px 4px !important;
      border-radius: 6px !important;
      min-height: 16px !important;
      
      .category-icon {
        font-size: 8px !important;
        margin-right: 1px !important;
      }
    }
    
    .news-actions-mobile {
      .q-btn {
        min-height: 18px;
        min-width: 18px;
        border-radius: 2px;
        background: rgba(255, 255, 255, 0.1);
        color: rgba(255, 255, 255, 0.7);
        
        .q-icon {
          font-size: 8px !important;
        }
        
        &:active {
          transform: scale(0.9);
          background: rgba(255, 255, 255, 0.2);
        }
      }
    }
  }
  
  .news-actions {
    display: none; // Скрываем десктопные действия в мобильной версии
  }
  
  
  .mobile-stats-floating {
    bottom: 15px;
    padding: var(--spacing-xs) var(--spacing-sm);
    border-radius: 16px;
    font-size: 11px;
  }
}

// === ДИАЛОГИ И ФИЛЬТРЫ ===
.filters-dialog-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.filters-dialog-header {
  background: var(--gradient-primary) !important;
  color: white !important;
  padding: 20px 24px !important;
  
  .text-h6 {
    color: white !important;
    font-weight: 600 !important;
    display: flex;
    align-items: center;
  }
  
  .q-icon {
    color: white !important;
  }
  
  .q-btn {
    color: white !important;
  }
}

.filters-dialog-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px !important;
}

.filters-dialog-actions {
  padding: 16px 24px !important;
  background: var(--bg-secondary) !important;
  border-top: 1px solid var(--border-primary) !important;
  display: flex !important;
  justify-content: space-between !important;
  align-items: center !important;
}

.filters-dialog-actions .q-btn {
  display: block !important;
  visibility: visible !important;
  opacity: 1 !important;
}

.filters-dialog-actions .q-btn {
  transition: all 0.3s ease;
}

.reset-btn {
  background: #ff4444 !important;
  color: white !important;
  font-weight: 600 !important;
}

.reset-btn:hover {
  background: #ff6666 !important;
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(255, 68, 68, 0.3);
}

.filters-dialog-actions .q-btn:disabled {
  opacity: 0.5;
  color: var(--text-tertiary) !important;
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
      border-color: var(--accent-color) !important;
      box-shadow: var(--shadow-sm) !important;
    }
    
    &:focus-within {
      border-color: var(--accent-color) !important;
      box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.1) !important;
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
  color: var(--accent-color) !important;
  font-size: 1.5rem !important;
  transition: all 0.3s ease !important;
  filter: drop-shadow(0 0 4px rgba(245, 158, 11, 0.3)) !important;
  
  &:hover {
    color: var(--primary-light) !important;
    transform: scale(1.1) !important;
    filter: drop-shadow(0 0 8px rgba(245, 158, 11, 0.5)) !important;
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
    box-shadow: 0 0 0 2px var(--accent-color) inset !important;
    font-weight: 700 !important;
  }
  
  :deep(.q-date__range) {
    background: rgba(245, 158, 11, 0.1) !important;
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

// === ДИАЛОГ ПРОСМОТРА НОВОСТИ ===
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
    color: #ffffff !important;
  }
  
  .news-description {
    font-size: 1rem;
    line-height: 1.6;
    margin-bottom: 16px;
    word-break: break-word;
    overflow-wrap: break-word;
    color: #ffffff !important;
  }
  
  .news-content-text {
    font-size: 1.1rem;
    line-height: 1.8;
    word-break: break-word;
    overflow-wrap: break-word;
    color: #ffffff !important;
    font-weight: 400;
    letter-spacing: 0.02em;
  }
}

// Принудительные стили для диалога новости
.q-dialog .dialog-news-content .news-title {
  color: #ffffff !important;
}

.q-dialog .dialog-news-content .news-description {
  color: #ffffff !important;
}

.q-dialog .dialog-news-content .news-content-text {
  color: #ffffff !important;
}

// Десктопная версия - уменьшаем размер изображения
@media (min-width: 769px) {
  .dialog-image-wrapper {
    max-width: 600px;
    margin: 0 auto 20px auto;
    
    .dialog-news-image {
      max-height: 400px;
      object-fit: cover;
    }
  }
}

// Мобильная версия - оставляем как есть
@media (max-width: 768px) {
  .dialog-image-wrapper {
    width: 100%;
    
    .dialog-news-image {
      width: 100%;
      height: auto;
    }
  }
}

// Дополнительные стили для улучшения читаемости
.dialog-news-content {
  // Улучшаем контрастность и читаемость
  .news-meta {
    color: #e0e0e0 !important;
    
    .source-name {
      color: #FFA726 !important;
      font-weight: 600;
    }
    
    .country-flag {
      font-size: 1.2em;
    }
  }
  
  // Улучшаем стили для категории
  .category-chip-modern {
    font-weight: 600 !important;
    font-size: 0.9rem !important;
  }
  
  // Улучшаем стили для кнопки "Читать полностью"
  .q-btn {
    font-weight: 600;
    text-transform: none;
    border-radius: 8px;
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

// === ПРИНУДИТЕЛЬНЫЕ СТИЛИ ===
.news-page .news-image-wrapper,
.news-item .news-image-wrapper,
.news-content .news-image-wrapper {
  width: 160px !important;
  height: 100px !important;
  max-width: 160px !important;
  max-height: 100px !important;
  flex-shrink: 0 !important;
}

@media (max-width: 1024px) {
  .news-page .news-image-wrapper,
  .news-item .news-image-wrapper,
  .news-content .news-image-wrapper {
    width: 100% !important;
    height: 180px !important;
    max-width: 100% !important;
    max-height: 180px !important;
  }
}

@media (max-width: 768px) {
  .news-page .news-image-wrapper,
  .news-item .news-image-wrapper,
  .news-content .news-image-wrapper {
    width: 80px !important;
    height: 80px !important;
    max-width: 80px !important;
    max-height: 80px !important;
  }
}

@media (max-width: 480px) {
  .news-page .news-image-wrapper,
  .news-item .news-image-wrapper,
  .news-content .news-image-wrapper {
    width: 70px !important;
    height: 70px !important;
    max-width: 70px !important;
    max-height: 70px !important;
  }
}

/* ГЛОБАЛЬНЫЕ СТИЛИ ДЛЯ ПОИСКОВОЙ ПАНЕЛИ */
.news-page .search-section {
  width: 100%;
  max-width: none;
}
</style>

<!-- Принудительные стили без scoped для счетчика новостей -->
<style>
@media (min-width: 769px) {
  .news-counter {
    color: #00d1c1 !important;
    font-size: 14px !important;
    font-weight: 600 !important;
    background: none !important;
    padding: 0 !important;
    border: none !important;
    border-radius: 0 !important;
    text-shadow: none !important;
    transition: none !important;
  }
  
  .news-counter:hover {
    color: #00d1c1 !important;
    background: none !important;
    border: none !important;
    transform: none !important;
    box-shadow: none !important;
  }
}
</style>