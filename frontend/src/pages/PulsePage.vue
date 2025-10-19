<template>
  <q-page class="pulse-page">
    <!-- Заголовок страницы -->
    <div class="page-header">
      <div class="header-content">
        <div class="breadcrumb">
          <q-btn 
            flat 
            icon="arrow_back" 
            @click="goBack"
            class="back-btn"
          >
            <q-tooltip>Назад к списку пульсов</q-tooltip>
          </q-btn>
          <span class="breadcrumb-text">Пульс</span>
        </div>
        <div class="title-with-edit">
          <h1 class="page-title">{{ pulse?.name || 'Загрузка...' }}</h1>
          <q-icon
            name="edit"
            size="20px"
            class="edit-icon"
            @click="editPulse"
          >
            <q-tooltip>Редактировать пульс</q-tooltip>
          </q-icon>
        </div>
        <p class="page-subtitle" v-if="pulse?.description">
          {{ pulse.description }}
        </p>
      </div>
    </div>

    <!-- Индикатор загрузки -->
    <div v-if="loading" class="loading-section">
      <q-spinner-dots color="primary" size="40px" />
      <p class="loading-text">Загружаем пульс...</p>
    </div>

    <!-- Информация о пульсе -->
    <div v-else-if="pulse" class="pulse-info-section">
      <div class="info-grid">
        <BaseCard class="info-card">
          <div class="info-content">
            <div class="info-icon">
              <q-icon name="public" size="20px" />
            </div>
            <div class="info-text">
              <div class="info-label">Страны</div>
              <div class="info-value">{{ uniqueCountriesCount }}</div>
            </div>
          </div>
        </BaseCard>

        <BaseCard class="info-card">
          <div class="info-content">
            <div class="info-icon">
              <q-icon name="category" size="20px" />
            </div>
            <div class="info-text">
              <div class="info-label">Категории</div>
              <div class="info-value">{{ uniqueCategoriesCount }}</div>
            </div>
          </div>
        </BaseCard>

        <BaseCard class="info-card">
          <div class="info-content">
            <div class="info-icon">
              <q-icon name="trending_up" size="20px" />
            </div>
            <div class="info-text">
              <div class="info-label">Новостей</div>
              <div class="info-value">{{ news.length }}</div>
            </div>
          </div>
        </BaseCard>
      </div>

      <!-- Ключевые слова -->
      <div v-if="pulse.keywords" class="keywords-section">
        <h3 class="section-title">Ключевые слова</h3>
        <div class="keywords-list">
          <q-chip
            v-for="keyword in pulse.keywords.split(',')"
            :key="keyword"
            size="md"
            color="primary"
            text-color="white"
            class="keyword-chip"
          >
            {{ keyword.trim() }}
          </q-chip>
        </div>
      </div>

      <!-- Новости -->
      <div class="news-section">
        <h3 class="section-title">Последние новости</h3>
        
        <div v-if="newsLoading" class="news-loading">
          <q-spinner-dots color="primary" size="32px" />
          <p class="loading-text">Загружаем новости...</p>
        </div>
        
        <div v-else-if="news.length > 0" class="news-list">
          <q-infinite-scroll @load="loadMoreNews" :offset="250">
            <div 
              v-for="(item, index) in news" 
              :key="item.id"
              :style="{ '--animation-delay': `${index * 0.1}s` }"
              class="news-item"
              @click="viewNewsItem(item)"
            >
            <!-- Полоска слева -->
            <div class="news-accent-bar"></div>
            
            <!-- Контент карточки -->
            <div class="news-content">
                <!-- Изображение новости -->
                <div v-if="isValidImageUrl(item.image_url || item.image)" class="news-image-wrapper">
                  <q-img
                    :src="item.image_url || item.image"
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
                    <span class="country-flag">{{ item.country?.flag_emoji || item.country?.flag || '🌍' }}</span>
                    <span class="source-name">{{ item.source?.name || item.source_name || 'Неизвестный источник' }}</span>
                          </div>
                  <div class="news-date">{{ formatDate(item.published_at) }}</div>
                        </div>

                <!-- Заголовок -->
                <h3 class="news-title">{{ cleanText(item.title) }}</h3>

                <!-- Описание -->
                <p class="news-description">{{ cleanText(item.description) }}</p>

                <!-- Категория и действия -->
                <div class="news-category">
                          <q-chip
                            v-if="item.category && item.category.name"
                            :style="{
                              background: `linear-gradient(135deg, ${getCategoryColor(item.category.color)} 0%, ${lightenColor(item.category.color, 20)} 100%) !important`,
                              color: 'white !important',
                              border: 'none !important'
                            }"
                    class="category-chip"
                          >
                    <q-icon :name="getCategoryIcon(item.category.icon || item.category.slug)" class="category-icon" />
                            {{ item.category.name }}
                          </q-chip>
                          
                          <!-- Действия для мобильной версии -->
                          <div class="news-actions-mobile">
                            <q-btn
                              flat
                              round
                              dense
                              icon="share"
                              size="sm"
                              @click.stop="shareNews(item)"
                            >
                              <q-tooltip>Поделиться</q-tooltip>
                            </q-btn>
                            <q-btn
                              flat
                              round
                              dense
                              icon="bookmark_border"
                              size="sm"
                              @click.stop="bookmarkNews(item)"
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
                          @click.stop="shareNews(item)"
                        >
                          <q-tooltip>Поделиться</q-tooltip>
                        </q-btn>
                        <q-btn
                          flat
                          round
                          dense
                          icon="bookmark_border"
                          size="sm"
                          @click.stop="bookmarkNews(item)"
                        >
                          <q-tooltip>В закладки</q-tooltip>
                        </q-btn>
                      </div>
                    </div>
                  </div>
            </div>

            </div>
            
            <template v-slot:loading>
              <div class="loading-more">
                <q-spinner-dots color="primary" size="32px" />
              </div>
            </template>
          </q-infinite-scroll>
        </div>
        
        <div v-else class="no-news">
          <q-icon name="newspaper" size="48px" class="no-news-icon" />
          <p class="no-news-text">Новости не найдены</p>
        </div>
      </div>

    </div>
    <div v-else-if="!loading && !pulse" class="error-state">
      <div class="error-content">
        <q-icon name="error" size="80px" class="error-icon" />
        <h3 class="error-title">Пульс не найден</h3>
        <p class="error-description">
          Возможно, пульс был удален или у вас нет доступа к нему
        </p>
        <BaseButton
          type="primary"
          size="lg"
          icon="arrow_back"
          label="Вернуться к списку"
          @click="goBack"
          class="error-action-btn"
        />
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
                              {{ cleanText(selectedNews.source?.name || selectedNews.source_name || 'Неизвестный источник') }}
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
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter, useRoute } from 'vue-router'
import { pulseService } from 'src/services/api'
import BaseCard from 'src/components/BaseCard.vue'

const $q = useQuasar()
const router = useRouter()
const route = useRoute()

// Состояние
const loading = ref(false)
const pulse = ref(null)
const news = ref([])
const newsLoading = ref(false)

// Infinite scroll и модальное окно
const currentPage = ref(1)
const pageSize = ref(10)
const hasMoreNews = ref(true)
const showNewsDialog = ref(false)
const selectedNews = ref(null)

// Computed свойства для подсчета статистики
const uniqueCountriesCount = computed(() => {
  if (!pulse.value) return 0
  
  // Получаем уникальные страны из источников пульса
  if (pulse.value.sources && Array.isArray(pulse.value.sources)) {
    const countries = new Set()
    pulse.value.sources.forEach(source => {
      if (source.country_id) {
        countries.add(source.country_id)
      }
    })
    return countries.size
  }
  
  return 0
})

const uniqueCategoriesCount = computed(() => {
  if (!pulse.value) return 0
  
  // Получаем количество категорий из настроек пульса
  if (pulse.value.categories && Array.isArray(pulse.value.categories)) {
    return pulse.value.categories.length
  }
  
  return 0
})

// Методы
const loadNews = async (pulseId, page = 1, reset = false) => {
  try {
    if (reset) {
      newsLoading.value = true
      currentPage.value = 1
      news.value = []
      hasMoreNews.value = true
    }
    
    const params = {
      page: page,
      page_size: pageSize.value
    }
    
    const response = await pulseService.getPulseNews(pulseId, params)
    
    let newNews = []
    if (response.data && response.data.success && response.data.data) {
      newNews = response.data.data
    } else {
      newNews = []
    }
    
    if (reset) {
      news.value = newNews
    } else {
      news.value = [...news.value, ...newNews]
    }
    
    // Проверяем, есть ли еще новости для загрузки
    hasMoreNews.value = newNews.length === pageSize.value
    
  } catch (error) {
    console.error('Ошибка загрузки новостей:', error)
    if (reset) {
      news.value = []
    }
  } finally {
    newsLoading.value = false
  }
}

const loadMoreNews = async (index, done) => {
  if (!hasMoreNews.value || newsLoading.value) {
    done()
    return
  }
  
  try {
    currentPage.value++
    const pulseId = route.params.id
    await loadNews(pulseId, currentPage.value, false)
    done()
  } catch (error) {
    console.error('Ошибка загрузки дополнительных новостей:', error)
    done()
  }
}

const loadPulse = async () => {
  try {
    loading.value = true
    const pulseId = route.params.id
    
    const response = await pulseService.getPulseById(pulseId)
    
    if (response.data && response.data.success && response.data.data) {
      pulse.value = response.data.data
      // Автоматически загружаем новости для этого пульса
      await loadNews(pulseId, 1, true)
    } else {
      pulse.value = null
    }
  } catch (error) {
    console.error('Ошибка загрузки пульса:', error)
    pulse.value = null
    $q.notify({
      message: 'Не удалось загрузить пульс',
      type: 'negative',
      position: 'bottom'
    })
  } finally {
    loading.value = false
  }
}

const viewNewsItem = (item) => {
  selectedNews.value = item
  showNewsDialog.value = true
}

const openOriginalNews = (url) => {
  if (url) {
    window.open(url, '_blank')
  }
}

const formatDate = (dateString) => {
  if (!dateString) return 'Неизвестно'
  
  const date = new Date(dateString)
  const now = new Date()
  const diffInHours = Math.floor((now - date) / (1000 * 60 * 60))
  
  if (diffInHours < 1) return 'Только что'
  if (diffInHours < 24) return `${diffInHours}ч назад`
  
  const diffInDays = Math.floor(diffInHours / 24)
  if (diffInDays < 7) return `${diffInDays}д назад`
  
  return date.toLocaleDateString('ru-RU')
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

const goBack = () => {
  router.push('/pulses')
}

const editPulse = () => {
  if (!pulse.value) return
  
  // Переходим на страницу списка пульсов с параметром редактирования
  router.push({
    path: '/pulses',
    query: { edit: pulse.value.id }
  })
}

onMounted(() => {
  loadPulse()
})
</script>

<style lang="scss" scoped>
// === СТРАНИЦА ПУЛЬСА ===
.pulse-page {
  min-height: 100vh;
  background: var(--bg-main);
  padding: var(--spacing-xl);
}

// === ЗАГОЛОВОК СТРАНИЦЫ ===
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-3xl);
  flex-wrap: wrap;
  gap: var(--spacing-lg);
}

.title-with-edit {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-sm);
}

.edit-icon {
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-normal);
  
  &:hover {
    color: var(--accent-color);
    transform: scale(1.1);
  }
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-md);
  
  .back-btn {
    color: var(--text-secondary);
    
    &:hover {
      color: var(--primary-color);
    }
  }
  
  .breadcrumb-text {
    color: var(--text-secondary);
    font-size: var(--font-size-sm);
  }
}

.header-content {
  flex: 1;
  min-width: 300px;
  
  .page-title {
    font-size: var(--font-size-4xl);
    font-weight: var(--font-weight-extrabold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-sm);
    background: var(--gradient-primary);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  
  .page-subtitle {
    font-size: var(--font-size-lg);
    color: var(--text-secondary);
    line-height: var(--line-height-relaxed);
  }
}


// === СЕКЦИЯ ЗАГРУЗКИ ===
.loading-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-5xl);
  
  .loading-text {
    margin-top: var(--spacing-lg);
    color: var(--text-secondary);
    font-size: var(--font-size-lg);
  }
}

// === ИНФОРМАЦИЯ О ПУЛЬСЕ ===
.pulse-info-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3xl);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--spacing-md);
  width: 100%;
  max-width: 800px;
}

.info-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border: 1px solid var(--border-primary);
  background: var(--bg-elevated);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    border-color: var(--accent-color);
  }
}

.info-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-lg);
  min-height: 80px;
}

.info-icon {
  width: 40px;
  height: 40px;
  background: var(--gradient-primary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(139, 143, 241, 0.3);
  
  .q-icon {
    color: white;
    font-size: 20px;
  }
}

.info-text {
  flex: 1;
  min-width: 0;
  
  .info-label {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    margin-bottom: 4px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  
  .info-value {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1;
  }
}

// === КЛЮЧЕВЫЕ СЛОВА ===
.keywords-section {
  .section-title {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-lg);
  }
}

.keywords-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
}

.keyword-chip {
  font-size: var(--font-size-sm);
}

// === ДЕЙСТВИЯ ===
.actions-section {
  display: flex;
  gap: var(--spacing-lg);
  justify-content: center;
  flex-wrap: wrap;
}

.primary-action,
.secondary-action {
  min-width: 200px;
}

// === ОШИБКА ===
.error-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  padding: var(--spacing-5xl);
}

.error-content {
  text-align: center;
  max-width: 500px;
  
  .error-icon {
    color: var(--error-color);
    margin-bottom: var(--spacing-xl);
  }
  
  .error-title {
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-md);
  }
  
  .error-description {
    font-size: var(--font-size-lg);
    color: var(--text-secondary);
    line-height: var(--line-height-relaxed);
    margin-bottom: var(--spacing-3xl);
  }
  
  .error-action-btn {
    box-shadow: var(--shadow-md);
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: var(--shadow-lg);
    }
  }
}

// === АДАПТИВНОСТЬ ===
@media (max-width: 1024px) {
  .pulse-page {
    padding: var(--spacing-lg);
  }
  
  .page-header {
    flex-direction: column;
    align-items: stretch;
    text-align: center;
  }
  
  .info-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: var(--spacing-sm);
    max-width: 100%;
  }
  
  .info-content {
    padding: var(--spacing-md);
    min-height: 70px;
  }
  
  .info-icon {
    width: 36px;
    height: 36px;
    
    .q-icon {
      font-size: 18px;
    }
  }
  
  .info-text {
    .info-label {
      font-size: var(--font-size-xs);
      margin-bottom: 2px;
    }
    
    .info-value {
      font-size: 1.5rem;
    }
  }
}

@media (max-width: 768px) {
  .pulse-page {
    padding: var(--spacing-md);
  }
  
  .page-title {
    font-size: var(--font-size-3xl) !important;
  }
  
  .page-subtitle {
    font-size: var(--font-size-base) !important;
  }
  
  .info-grid {
    grid-template-columns: 1fr;
    gap: var(--spacing-sm);
    max-width: 100%;
  }
  
  .info-content {
    padding: var(--spacing-md);
    min-height: 60px;
  }
  
  .info-icon {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    
    .q-icon {
      font-size: 16px;
    }
  }
  
  .info-text {
    .info-label {
      font-size: var(--font-size-xs);
      margin-bottom: 2px;
    }
    
    .info-value {
      font-size: 1.25rem;
    }
  }
  
  .actions-section {
    flex-direction: column;
    align-items: center;
  }
  
  .primary-action,
  .secondary-action {
    min-width: auto;
    width: 100%;
    max-width: 300px;
  }
}

@media (max-width: 480px) {
  .pulse-page {
    padding: var(--spacing-sm);
  }
  
  .page-title {
    font-size: var(--font-size-2xl) !important;
  }
  
  .info-grid {
    gap: var(--spacing-xs);
  }
  
  .info-content {
    padding: var(--spacing-sm);
    min-height: 50px;
  }
  
  .info-icon {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    
    .q-icon {
      font-size: 14px;
    }
  }
  
  .info-text {
    .info-label {
      font-size: 10px;
      margin-bottom: 1px;
    }
    
    .info-value {
      font-size: 1.1rem;
    }
  }
  
  .page-subtitle {
    font-size: var(--font-size-sm) !important;
  }
  
  .info-content {
    padding: var(--spacing-md);
  }
  
  .error-content {
    padding: var(--spacing-lg);
    
    .error-title {
      font-size: var(--font-size-xl) !important;
    }
    
    .error-description {
      font-size: var(--font-size-base) !important;
    }
  }
}

// === СЕКЦИЯ НОВОСТЕЙ ===
.news-section {
  margin-top: var(--spacing-3xl);
  
  .section-title {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-lg);
  }
  
  .news-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--spacing-3xl);
    
    .loading-text {
      margin-top: var(--spacing-md);
      color: var(--text-secondary);
      font-size: var(--font-size-base);
    }
  }
  
  .news-list {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
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

.news-image-wrapper {
  flex-shrink: 0;
  width: 160px;
  height: 100px;
  border-radius: var(--radius-md);
  overflow: hidden;
  position: relative;
  background: var(--bg-secondary);
  
  .news-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform var(--transition-normal);
  }
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
  
  .no-news {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--spacing-3xl);
    text-align: center;
    
    .no-news-icon {
      color: var(--text-tertiary);
      margin-bottom: var(--spacing-md);
    }
    
    .no-news-text {
      font-size: var(--font-size-lg);
      color: var(--text-secondary);
    }
  }
}

// === LOADING MORE ===
.loading-more {
  display: flex;
  justify-content: center;
  padding: var(--spacing-xl);
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
</style>
