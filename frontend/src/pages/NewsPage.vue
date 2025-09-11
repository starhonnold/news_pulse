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
    <q-card class="q-mb-md modern-card glass-effect">
      <q-card-section>
        <div class="row q-gutter-sm">
          <!-- Поиск -->
          <div class="col-12 col-sm-6 col-md-3 col-lg-3">
            <q-input
              v-model="searchQuery"
              placeholder="Поиск новостей..."
              dense
              outlined
              class="modern-input"
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
          <div class="col-12 col-sm-6 col-md-3 col-lg-3">
            <q-select
              v-model="selectedCategories"
              :options="categoryOptions"
              label="Категории"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="modern-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCategoriesClear"
            />
          </div>

          <!-- Фильтр по странам -->
          <div class="col-12 col-sm-6 col-md-3 col-lg-3">
            <q-select
              v-model="selectedCountries"
              :options="countryOptions"
              label="Страны"
              dense
              outlined
              multiple
              use-chips
              clearable
              class="modern-input"
              emit-value
              map-options
              @update:model-value="onFilterChange"
              @clear="onCountriesClear"
            />
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
                <div class="row no-wrap">
                  <!-- Изображение новости -->
                  <div v-if="isValidImageUrl(news.image_url || news.image)" class="col-auto">
                    <q-img
                      :src="news.image_url || news.image"
                      style="width: 120px; height: 120px"
                      class="rounded-borders news-image"
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
                  <div class="col">
                    <q-card-section class="q-pa-md">
                      <!-- Мета информация -->
                      <div class="row items-center q-mb-sm">
                        <div class="col-auto">
                          <div class="news-meta">
                            <span class="country-flag q-mr-xs">{{ news.country?.flag_emoji || news.country?.flag || '🌍' }}</span>
                            <span class="source-name text-weight-medium text-primary">
                              {{ news.source?.name || news.source_name || 'Неизвестный источник' }}
                            </span>
                            <q-separator vertical class="q-mx-sm" />
                            <span class="text-grey-7">{{ formatDate(news.published_at) }}</span>
                          </div>
                        </div>
                        <div class="col-auto">
                          <q-chip
                            v-if="news.category && news.category.name"
                            :color="news.category.color || 'primary'"
                            :style="news.category.color ? `background-color: ${news.category.color} !important; border-color: ${news.category.color} !important;` : ''"
                            text-color="white"
                            dense
                            class="q-ml-sm"
                          >
                            <q-icon :name="getCategoryIcon(news.category.icon)" class="q-mr-xs" />
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
                      <div class="row items-center justify-between">
                        <div class="col">
                          <div class="news-actions">
                            <q-icon name="visibility" class="q-mr-xs" />
                            {{ news.view_count || 0 }}
                          </div>
                        </div>
                        <div class="col-auto">
                          <div class="news-action-buttons">
                            <q-btn
                              flat
                              round
                              dense
                              icon="share"
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
                              @click.stop="bookmarkNews(news)"
                            >
                              <q-tooltip>В закладки</q-tooltip>
                            </q-btn>
                          </div>
                        </div>
                      </div>
                    </q-card-section>
                  </div>
                </div>

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

        <q-card-section v-if="selectedNews">
          <div class="row no-wrap">
                    <!-- Изображение новости -->
                    <div v-if="isValidImageUrl(selectedNews.image_url || selectedNews.image)" class="col-auto">
                      <q-img
                        :src="selectedNews.image_url || selectedNews.image"
                        style="width: 200px; height: 200px"
                        class="rounded-borders news-image"
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
            <div class="col">
              <q-card-section class="q-pa-md">
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
                      :color="selectedNews.category.color || 'grey'"
                      :style="selectedNews.category.color ? `background-color: ${selectedNews.category.color} !important; border-color: ${selectedNews.category.color} !important;` : ''"
                      text-color="white"
                      dense
                      class="q-ml-sm"
                    >
                      <q-icon :name="getCategoryIcon(selectedNews.category.icon)" class="q-mr-xs" />
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
                <div class="row justify-end">
                  <div class="col-auto">
                    <q-btn
                      color="secondary"
                      label="Закрыть"
                      @click="showNewsDialog = false"
                      flat
                    />
                  </div>
                </div>
              </q-card-section>
            </div>
          </div>
        </q-card-section>
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
         (selectedCountries.value && selectedCountries.value.length > 0)
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
      icon: cat.icon
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

// Функция для получения правильной иконки категории
const getCategoryIcon = (iconName) => {
  const iconMap = {
    'politics': 'account_balance',
    'trending-up': 'trending_up',
    'sports': 'sports_soccer',
    'cpu': 'computer',
    'palette': 'palette',
    'flask': 'science',
    'users': 'group',
    'alert-triangle': 'warning'
  }
  return iconMap[iconName] || 'info'
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

// Жизненный цикл
onMounted(() => {
  loadNews(1, true)
})
</script>

<style lang="scss" scoped>
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
</style>
