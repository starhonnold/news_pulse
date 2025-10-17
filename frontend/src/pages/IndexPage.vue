<template>
  <q-page class="modern-page">
    <div class="q-pa-sm q-pa-md-md q-pa-lg-lg">
      <!-- Главная страница с пульсами -->
      <div v-if="!selectedPulse">
        <!-- Если нет пульсов - показываем большую кнопку создания -->
        <div v-if="userPulses.length === 0" class="flex flex-center" style="min-height: 60vh;">
          <div class="text-center">
            <q-btn
              color="primary"
              size="xl"
              class="create-pulse-btn gradient-btn white-content"
              unelevated
              text-color="white"
              @click="showCreatePulse = true"
            >
              <i class="q-icon notranslate material-icons text-white" aria-hidden="true" role="img" style="color: white !important;">add_circle</i>
            </q-btn>
            <div class="text-h6 q-mt-md text-grey-7">
              Создайте свой первый пульс для отслеживания новостей
            </div>
          </div>
        </div>

        <!-- Если есть пульсы - показываем список пульсов -->
        <div v-else>
          <div class="row q-mb-md">
            <div class="col">
              <div class="text-subtitle1 text-grey-7">
                Выберите пульс для просмотра новостей или создайте новый
              </div>
            </div>
            <div class="col-auto">
              <q-btn
                color="primary"
                icon="add_circle"
                label="Новый пульс"
                class="modern-btn gradient-btn"
                unelevated
                @click="showCreatePulse = true"
              />
            </div>
          </div>

          <div class="row q-gutter-sm q-gutter-md-md">
            <div 
              v-for="pulse in userPulses" 
              :key="pulse?.id || Math.random()"
              class="col-12 col-sm-6 col-md-4 col-lg-3 col-xl-3"
            >
              <q-card 
                class="pulse-card equal-height-card cursor-pointer"
                @click="selectPulse(pulse)"
              >
                <q-card-section>
                  <div class="text-h6 text-weight-medium q-mb-sm">
                    {{ pulse?.name || 'Без названия' }}
                  </div>
                  <div class="text-body2 text-grey-7 q-mb-md">
                    {{ pulse?.description || 'Без описания' }}
                  </div>
                  
                  <!-- Ключевые слова пульса -->
                  <div v-if="pulse?.keywords" class="q-mb-sm">
                    <div class="text-caption text-grey-6 q-mb-xs">Ключевые слова:</div>
                    <div class="keywords-container">
                      <q-chip
                        v-for="keyword in pulse.keywords.split(',').slice(0, 3)"
                        :key="keyword.trim()"
                        dense
                        outline
                        color="primary"
                        class="q-mr-xs q-mb-xs"
                      >
                        {{ keyword.trim() }}
                      </q-chip>
                      <span v-if="pulse.keywords.split(',').length > 3" class="text-caption text-grey-6">
                        +{{ pulse.keywords.split(',').length - 3 }} еще
                      </span>
                    </div>
                  </div>
                  
                  <!-- Категории пульса -->
                  <div v-if="pulse?.categories && pulse.categories.length > 0" class="q-mb-sm">
                    <div class="text-caption text-grey-6 q-mb-xs">Категории:</div>
                      <q-chip
                        v-for="category in pulse.categories.slice(0, 3)"
                      :key="category.category_id"
                      :color="'primary'"
                      :style="`background-color: ${getCategoryColor(category.category_color)} !important; border-color: ${getCategoryColor(category.category_color)} !important;`"
                      text-color="white"
                      dense
                      class="q-mr-xs"
                    >
                      <q-icon :name="getCategoryIcon(category.category_icon || category.category_slug)" class="q-mr-xs" />
                      {{ category.category_name }}
                    </q-chip>
                    <span v-if="pulse.categories.length > 3" class="text-caption text-grey-6">
                      +{{ pulse.categories.length - 3 }} еще
                    </span>
                  </div>

                  <!-- Страны пульса -->
                  <div v-if="pulse?.sources && pulse.sources.length > 0" class="q-mb-sm">
                    <div class="text-caption text-grey-6 q-mb-xs">Страны:</div>
                    <q-chip
                      v-for="country in getUniqueCountries(pulse.sources).slice(0, 3)"
                      :key="country.id"
                      dense
                      outline
                      color="secondary"
                      class="q-mr-xs"
                    >
                      <q-icon name="public" class="q-mr-xs" />
                      {{ country.name }}
                    </q-chip>
                    <span v-if="getUniqueCountries(pulse.sources).length > 3" class="text-caption text-grey-6">
                      +{{ getUniqueCountries(pulse.sources).length - 3 }} еще
                    </span>
                  </div>

                  <!-- Статистика -->
                  <div class="row items-center text-caption text-grey-6">
                    <div class="col">
                      <q-icon name="article" class="q-mr-xs" />
                      {{ pulse?.news_count || 0 }} новостей
                    </div>
                    <div class="col-auto">
                      <q-icon name="schedule" class="q-mr-xs" />
                      {{ formatDate(pulse?.last_refreshed_at || pulse?.updated_at || pulse?.created_at) }}
                    </div>
                  </div>
                </q-card-section>

                <q-card-actions class="q-pt-none">
                  <q-btn 
                    flat 
                    color="primary" 
                    label="Открыть"
                    @click.stop="selectPulse(pulse)"
                  />
                  <q-space />
                  <q-btn 
                    flat 
                    round 
                    icon="more_vert"
                    @click.stop
                  >
                    <q-menu>
                      <q-list>
                        <q-item clickable @click="editPulse(pulse)">
                          <q-item-section avatar>
                            <q-icon name="edit" />
                          </q-item-section>
                          <q-item-section>Редактировать</q-item-section>
                        </q-item>
                        <q-item clickable @click="deletePulse(pulse)">
                          <q-item-section avatar>
                            <q-icon name="delete" color="negative" />
                          </q-item-section>
                          <q-item-section>Удалить</q-item-section>
                        </q-item>
                      </q-list>
                    </q-menu>
                  </q-btn>
                </q-card-actions>
              </q-card>
            </div>
          </div>
        </div>
      </div>

      <!-- Страница конкретного пульса с новостями -->
      <div v-else>
        <!-- Хедер пульса -->
        <div class="row items-center q-mb-md mobile-pulse-header">
          <div class="col-auto">
            <q-btn
              flat
              round
              icon="arrow_back"
              @click="selectedPulse = null"
              class="q-mr-sm mobile-back-btn"
              size="md"
            />
          </div>
          <div class="col mobile-pulse-title-wrapper">
            <div class="text-h5 text-weight-bold text-primary mobile-pulse-title">
              {{ selectedPulse.name }}
            </div>
            <div class="text-subtitle2 text-grey-7 mobile-pulse-desc">
              {{ selectedPulse.description }}
            </div>
          </div>
          <div class="col-12 col-sm-auto q-mt-sm q-mt-sm-none mobile-refresh-btn-wrapper">
            <q-btn
              :loading="isUpdating"
              color="primary"
              icon="refresh"
              :label="$q.screen.gt.xs ? 'Обновить' : ''"
              class="modern-btn gradient-btn mobile-refresh-btn"
              unelevated
              @click="updatePulseNews"
              size="md"
            />
          </div>
        </div>

        <!-- Статистика пульса -->
        <q-card class="status-card glass-effect q-mb-md">
          <q-card-section class="row items-center">
            <div class="col">
              <div class="text-caption text-secondary">
                Последнее обновление: <span class="text-primary">{{ lastUpdate }}</span>
              </div>
            </div>
            <div class="col-auto">
              <div class="text-caption text-secondary">
                Найдено: <span class="text-primary">{{ (pulseNews || []).length }} новостей</span>
              </div>
            </div>
          </q-card-section>
          
          <!-- Индикатор автообновления -->
          <q-linear-progress
            :value="updateProgress"
            color="primary"
            size="2px"
            class="q-mt-sm"
          />
        </q-card>

        <!-- Индикатор загрузки новостей -->
        <div v-if="isLoadingNews" class="row justify-center q-my-lg">
          <q-card class="loading-card glass-effect">
            <q-card-section class="text-center">
              <q-spinner-dots color="primary" size="40px" />
              <div class="text-h6 q-mt-md text-primary">{{ loadingMessage }}</div>
              <div v-if="retryCount > 0" class="text-caption text-grey-6 q-mt-sm">
                Попытка {{ retryCount }}/{{ maxRetries }}
              </div>
            </q-card-section>
          </q-card>
        </div>

        <!-- Новости пульса -->
        <div class="row">
          <div class="col-12">
            <q-infinite-scroll @load="loadMorePulseNews" :offset="250">
              <div class="news-grid stagger-animation">
                <q-card
                  v-for="news in (pulseNews || [])"
                  :key="news.id"
                  class="news-card q-mb-md cursor-pointer fade-in-up"
                  @click="openNews(news)"
                >
                  <!-- Изображение новости -->
                  <div v-if="news.image_url || news.image" class="news-image-wrapper">
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
                  <q-card-section class="q-pa-md mobile-pulse-news-section">
                        <!-- Мета информация -->
                        <div class="row items-center q-mb-sm">
                          <div class="col-auto">
                            <div class="news-meta">
                              <span class="country-flag q-mr-xs">{{ news.country?.flag || '🌍' }}</span>
                              <span class="source-name text-weight-medium text-primary">
                                {{ news.source?.name || 'Неизвестный источник' }}
                              </span>
                              <q-separator vertical class="q-mx-sm" />
                              <span class="text-grey-7">{{ formatDate(news.published_at) }}</span>
                            </div>
                          </div>
                          <div class="col-auto">
                            <q-chip
                              v-if="news.category && news.category.name"
                              :color="'primary'"
                              :style="`background-color: ${getCategoryColor(news.category.color)} !important; border-color: ${getCategoryColor(news.category.color)} !important;`"
                              text-color="white"
                              dense
                              class="q-ml-sm"
                            >
                              <q-icon :name="getCategoryIcon(news.category.icon || news.category.slug)" class="q-mr-xs" />
                              {{ news.category.name }}
                            </q-chip>
                          </div>
                        </div>

                        <!-- Заголовок -->
                        <div class="news-title text-h6 text-weight-medium q-mb-sm">
                          {{ news.title }}
                        </div>

                        <!-- Описание -->
                        <div class="news-description text-grey-8 q-mb-sm">
                          {{ news.description }}
                        </div>

                    <!-- Действия -->
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
                            dense
                            round
                            icon="share"
                            size="sm"
                            @click.stop="shareNews(news)"
                            class="q-mr-xs"
                          >
                            <q-tooltip>Поделиться</q-tooltip>
                          </q-btn>
                          <q-btn
                            flat
                            dense
                            round
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

                  <!-- Индикатор актуальности -->
                  <div
                    v-if="news.relevance_score > 0.8"
                    class="absolute-top-right q-ma-sm"
                  >
                    <q-badge color="red" floating>
                      <q-icon name="whatshot" />
                    </q-badge>
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

              <template v-slot:loading>
                <div class="row justify-center q-my-md">
                  <q-spinner-dots color="primary" size="40px" />
                </div>
              </template>
            </q-infinite-scroll>
          </div>
        </div>
      </div>
    </div>

    <!-- Диалог создания/редактирования пульса -->
    <q-dialog v-model="showCreatePulse" persistent :maximized="$q.platform.is.mobile">
      <q-card 
        :style="$q.platform.is.mobile ? '' : 'min-width: 600px; max-width: 800px'" 
        class="modern-card glass-effect"
      >
        <q-card-section>
          <div class="text-h6">{{ editingPulse ? 'Редактировать пульс' : 'Новый пульс' }}</div>
        </q-card-section>

        <q-card-section>
          <q-input
            v-model="newPulse.name"
            label="Название пульса *"
            outlined
            class="q-mb-md"
          />
          <q-input
            v-model="newPulse.keywords"
            label="Ключевые слова (через запятую)"
            outlined
            placeholder="ИИ, технологии, инновации"
            class="q-mb-md"
          />
          <q-input
            v-model="newPulse.description"
            label="Описание"
            outlined
            type="textarea"
            class="q-mb-md"
          />

          <div class="text-subtitle2 q-mb-sm row items-center">
            <span>Выберите страны: ({{ countries.length }} доступно)</span>
            <q-checkbox
              v-if="countries.length > 0"
              v-model="selectAllCountries"
              color="primary"
              size="sm"
              class="q-ml-sm"
            />
          </div>
          
          <div class="q-mb-md">
            <div v-if="countries.length === 0" class="text-grey-6 text-center q-py-md">
              Загрузка стран...
            </div>
            <q-chip
              v-else
              v-for="country in countries"
              :key="country.id"
              :selected="newPulse.countries.includes(country.id)"
              @click="togglePulseCountry(country.id)"
              clickable
              :color="newPulse.countries.includes(country.id) ? 'primary' : 'grey-3'"
              :text-color="newPulse.countries.includes(country.id) ? 'white' : 'black'"
              class="q-ma-xs"
            >
              <span class="q-mr-xs">{{ country.flag_emoji || country.flag }}</span>
              {{ country.name }}
            </q-chip>
          </div>

          <div class="text-subtitle2 q-mb-sm row items-center">
            <span>Выберите категории: ({{ categories.length }} доступно)</span>
            <q-checkbox
              v-if="categories.length > 0"
              v-model="selectAllCategories"
              color="primary"
              size="sm"
              class="q-ml-sm"
            />
          </div>
          
          <div class="q-mb-md">
            <div v-if="categories.length === 0" class="text-grey-6 text-center q-py-md">
              Загрузка категорий...
            </div>
            <q-chip
              v-else
              v-for="category in categories"
              :key="category.id"
              :selected="newPulse.categories.includes(category.id)"
              @click="togglePulseCategory(category.id)"
              clickable
              :color="newPulse.categories.includes(category.id) ? 'primary' : 'grey-3'"
              :text-color="newPulse.categories.includes(category.id) ? 'white' : 'black'"
              :style="newPulse.categories.includes(category.id) ? `background-color: ${getCategoryColor(category.color)} !important; border-color: ${getCategoryColor(category.color)} !important;` : ''"
              class="q-ma-xs"
            >
              <q-icon :name="getCategoryIcon(category.icon || category.slug)" class="q-mr-xs" />
              {{ category.name }}
            </q-chip>
          </div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Отмена" @click="cancelPulseDialog" />
          <q-btn color="primary" :label="editingPulse ? 'Сохранить' : 'Создать пульс'" @click="savePulseMain" />
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
                      :color="'primary'"
                      :style="`background-color: ${getCategoryColor(selectedNews.category.color)} !important; border-color: ${getCategoryColor(selectedNews.category.color)} !important;`"
                      text-color="white"
                      dense
                      class="q-ml-sm"
                    >
                      <q-icon :name="getCategoryIcon(selectedNews.category.icon || selectedNews.category.slug)" class="q-mr-xs" />
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
  </q-page>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useQuasar } from 'quasar'
import { pulseService, referenceService, handleApiError } from 'src/services/api'

const $q = useQuasar()

// Состояние
const userPulses = ref([])
const selectedPulse = ref(null)
const pulseNews = ref([])
const isUpdating = ref(false)
const updateProgress = ref(0)
const lastUpdate = ref('')
const showNewsDialog = ref(false)
const selectedNews = ref(null)
const showCreatePulse = ref(false)
const editingPulse = ref(null)

// Состояние для улучшенных индикаторов загрузки
const isLoadingNews = ref(false)
const loadingMessage = ref('')
const retryCount = ref(0)
const maxRetries = ref(3)

// Данные для создания пульса
const newPulse = ref({
  name: '',
  description: '',
  countries: [],
  categories: [],
  keywords: '',
})

// Состояние для выбора всех элементов
const selectAllCountries = ref(false)
const selectAllCategories = ref(false)

// Справочные данные
const countries = ref([])
const categories = ref([])

// Watch для отслеживания изменений в чекбоксах "Выбрать все"
watch(selectAllCountries, (newValue) => {
  console.log('selectAllCountries changed to:', newValue)
  if (newValue) {
    newPulse.value.countries = countries.value.map(country => country.id)
    console.log('Selected all countries:', newPulse.value.countries)
  } else {
    newPulse.value.countries = []
    console.log('Deselected all countries')
  }
})

watch(selectAllCategories, (newValue) => {
  console.log('selectAllCategories changed to:', newValue)
  if (newValue) {
    newPulse.value.categories = categories.value.map(category => category.id)
    console.log('Selected all categories:', newPulse.value.categories)
  } else {
    newPulse.value.categories = []
    console.log('Deselected all categories')
  }
})



// Computed свойства

// Функция для получения уникальных стран из источников
function getUniqueCountries(sources) {
  if (!sources || !Array.isArray(sources)) return []
  
  const uniqueCountries = new Map()
  sources.forEach(source => {
    if (source.country_id && source.country_name) {
      uniqueCountries.set(source.country_id, {
        id: source.country_id,
        name: source.country_name
      })
    }
  })
  
  return Array.from(uniqueCountries.values())
}

// Функция для определения флага страны по домену источника
function getCountryFlagBySource(domain) {
  if (!domain) return '🌍'
  
  const domainToFlag = {
    'ria.ru': '🇷🇺',
    'tass.ru': '🇷🇺',
    'interfax.ru': '🇷🇺',
    'lenta.ru': '🇷🇺',
    'gazeta.ru': '🇷🇺',
    'rbc.ru': '🇷🇺',
    'kommersant.ru': '🇷🇺',
    'vedomosti.ru': '🇷🇺',
    'russian.rt.com': '🇷🇺',
    'sputniknews.ru': '🇷🇺',
    'belta.by': '🇧🇾',
    'sb.by': '🇧🇾',
    'belarusnews.by': '🇧🇾',
    'inform.kz': '🇰🇿',
    'tengrinews.kz': '🇰🇿',
    'nur.kz': '🇰🇿',
    'unian.net': '🇺🇦',
    'korrespondent.net': '🇺🇦',
    'uza.uz': '🇺🇿',
    'gazeta.uz': '🇺🇿',
    'kabar.kg': '🇰🇬',
    '24.kg': '🇰🇬',
    'armenpress.am': '🇦🇲',
    'news.am': '🇦🇲',
    'azertag.az': '🇦🇿',
    'trend.az': '🇦🇿',
    'khovar.tj': '🇹🇯',
    'news.tj': '🇹🇯',
    'moldpres.md': '🇲🇩',
    'newsmaker.md': '🇲🇩',
    'sputnik-georgia.ru': '🇬🇪',
    '1tv.ge': '🇬🇪'
  }
  
  return domainToFlag[domain] || '🌍'
}

// Методы управления пульсами
function selectPulse(pulse) {
  console.log('Selecting pulse:', pulse)
  console.log('Pulse ID:', pulse.id, 'Type:', typeof pulse.id)
  console.log('Pulse name:', pulse.name)
  selectedPulse.value = pulse
  loadPulseNewsFromApi(pulse)
}


function loadMorePulseNews(index, done) {
  // Пока что отключаем автоматическую загрузку больше новостей
  // чтобы избежать ошибки 429 Too Many Requests
  console.log('Load more requested, but disabled to prevent rate limiting')
  done()
}

function updatePulseNews() {
  if (selectedPulse.value) {
    updatePulseNewsFromApi()
  }
}

// Методы управления диалогом создания пульса
function togglePulseCountry(countryId) {
  const index = newPulse.value.countries.indexOf(countryId)
  if (index > -1) {
    newPulse.value.countries.splice(index, 1)
  } else {
    newPulse.value.countries.push(countryId)
  }
  updateSelectAllCountries()
}

function togglePulseCategory(categoryId) {
  const index = newPulse.value.categories.indexOf(categoryId)
  if (index > -1) {
    newPulse.value.categories.splice(index, 1)
  } else {
    newPulse.value.categories.push(categoryId)
  }
  updateSelectAllCategories()
}


// Обновление состояния "Выбрать все" для стран
function updateSelectAllCountries() {
  selectAllCountries.value = newPulse.value.countries.length === countries.value.length && countries.value.length > 0
}

// Обновление состояния "Выбрать все" для категорий
function updateSelectAllCategories() {
  selectAllCategories.value = newPulse.value.categories.length === categories.value.length && categories.value.length > 0
}

// Функция для получения правильной иконки категории
function getCategoryIcon(iconName) {
  if (!iconName) return 'info'
  
  const iconMap = {
    'politics': 'gavel',
    'politika': 'gavel',
    'trending-up': 'trending_up',
    'economy': 'trending_up',
    'ekonomika': 'trending_up',
    'sports': 'sports_soccer',
    'sport': 'sports_soccer',
    'cpu': 'computer',
    'technology': 'computer',
    'tech': 'computer',
    'tehnologii': 'computer',
    'palette': 'palette',
    'culture': 'palette',
    'kultura': 'palette',
    'flask': 'science',
    'science': 'science',
    'nauka': 'science',
    'users': 'people',
    'society': 'people',
    'obschestvo': 'people',
    'alert-triangle': 'warning',
    'incidents': 'warning',
    'proisshestviya': 'warning',
    'health': 'local_hospital',
    'zdorove': 'local_hospital',
    'education': 'school',
    'obrazovanie': 'school',
    'international': 'public',
    'mezhdunarodnye': 'public',
    'business': 'business',
    'biznes': 'business'
  }
  
  // Ищем точное совпадение
  if (iconMap[iconName]) {
    return iconMap[iconName]
  }
  
  // Ищем по частичному совпадению (для slug)
  const lowerIconName = iconName.toLowerCase()
  for (const [key, value] of Object.entries(iconMap)) {
    if (lowerIconName.includes(key)) {
      return value
    }
  }
  
  return 'info'
}

// Функция для преобразования цветов Quasar в CSS цвета
function getCategoryColor(quasarColor) {
  if (!quasarColor) return '#1976d2' // primary по умолчанию
  
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
    'brown-6': '#795548'
  }
  
  return colorMap[quasarColor] || '#1976d2'
}


// Основной метод сохранения через API
function savePulseMain() {
  savePulseToApi()
}

function editPulse(pulse) {
  editingPulse.value = pulse
  newPulse.value = {
    name: pulse.name || '',
    description: pulse.description || '',
    countries: pulse.countries ? [...pulse.countries] : [],
    categories: pulse.categories ? [...pulse.categories] : [],
    keywords: pulse.keywords || ''
  }
  // Обновляем состояние "Выбрать все" после загрузки данных
  setTimeout(() => {
    updateSelectAllCountries()
    updateSelectAllCategories()
  }, 100)
  showCreatePulse.value = true
}

function deletePulse(pulse) {
  if (confirm(`Вы уверены, что хотите удалить пульс "${pulse.name}"?`)) {
    deletePulseFromApi(pulse)
  }
}

function cancelPulseDialog() {
  showCreatePulse.value = false
  editingPulse.value = null
  newPulse.value = {
    name: '',
    description: '',
    countries: [],
    categories: [],
    keywords: '',
  }
  selectAllCountries.value = false
  selectAllCategories.value = false
}

// Общие методы для новостей
function formatDate(dateString) {
  if (!dateString) return 'Неизвестно'
  
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return 'Неизвестно'
  
  const now = new Date()
  const diffInMinutes = Math.floor((now - date) / (1000 * 60))
  
  if (diffInMinutes < 1) return 'только что'
  if (diffInMinutes < 60) return `${diffInMinutes} мин. назад`
  if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)} ч. назад`
  
  return date.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function isNewNews(news) {
  const newsDate = new Date(news.published_at)
  const now = new Date()
  return (now - newsDate) < (1000 * 60 * 60)
}

function openNews(news) {
  console.log('Opening news:', news)
  console.log('News content:', news.content)
  console.log('News content length:', news.content ? news.content.length : 0)
  selectedNews.value = news
  showNewsDialog.value = true
}

function shareNews(news) {
  if (navigator.share) {
    navigator.share({
      title: news.title,
      text: news.description,
      url: news.url
    })
  } else {
    navigator.clipboard.writeText(news.url)
    $q.notify({
      message: 'Ссылка скопирована в буфер обмена',
      type: 'positive',
      position: 'bottom'
    })
  }
}

function bookmarkNews() {
  $q.notify({
    message: 'Новость добавлена в закладки',
    type: 'positive',
    position: 'bottom'
  })
}

// Загрузка данных из API
async function loadUserPulses() {
  try {
    console.log('Loading user pulses...')
    const response = await pulseService.getUserPulses()
    console.log('User pulses response:', response)
    console.log('User pulses response.data:', response.data)
    
    // Проверяем, что response.data существует и является массивом
    if (response && response.data && response.data.success && Array.isArray(response.data.data)) {
      userPulses.value = response.data.data
      console.log('User pulses loaded:', userPulses.value.length)
    } else {
      console.warn('Invalid user pulses response, using empty array')
      console.log('User pulses response structure:', JSON.stringify(response, null, 2))
      userPulses.value = []
    }
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка загрузки пульсов')
    console.error('Error loading pulses:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
    })
  }
}

async function loadReferenceData() {
  try {
    console.log('Loading reference data...')
    
    // Загружаем категории
    const categoriesResponse = await referenceService.getCategories()
    console.log('Categories response:', categoriesResponse)
    console.log('Categories response.data:', categoriesResponse.data)
    console.log('Categories response.data type:', typeof categoriesResponse.data)
    console.log('Categories response.data isArray:', Array.isArray(categoriesResponse.data))
    
    if (categoriesResponse && categoriesResponse.data && categoriesResponse.data.success && Array.isArray(categoriesResponse.data.data)) {
      categories.value = categoriesResponse.data.data
      console.log('Categories loaded:', categories.value.length)
      console.log('First category example:', categories.value[0])
      console.log('Category fields:', categories.value[0] ? Object.keys(categories.value[0]) : 'No categories')
    } else {
      console.warn('Categories data is not valid')
      console.log('Categories response structure:', JSON.stringify(categoriesResponse, null, 2))
    }
    
    // Загружаем страны
    const countriesResponse = await referenceService.getCountries()
    console.log('Countries response:', countriesResponse)
    console.log('Countries response.data:', countriesResponse.data)
    console.log('Countries response.data type:', typeof countriesResponse.data)
    console.log('Countries response.data isArray:', Array.isArray(countriesResponse.data))
    
    if (countriesResponse && countriesResponse.data && countriesResponse.data.success && Array.isArray(countriesResponse.data.data)) {
      countries.value = countriesResponse.data.data
      console.log('Countries loaded:', countries.value.length)
    } else {
      console.warn('Countries data is not valid')
      console.log('Countries response structure:', JSON.stringify(countriesResponse, null, 2))
    }
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка загрузки справочных данных')
    console.error('Error loading reference data:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
    })
  }
}

async function loadPulseNewsFromApi(pulse) {
  try {
    console.log('Loading pulse news for pulse:', pulse)
    console.log('Pulse ID:', pulse.id, 'Type:', typeof pulse.id)
    
    // Проверяем, что ID пульса существует и валиден
    if (!pulse.id || pulse.id === undefined || pulse.id === null) {
      console.warn('Pulse ID is undefined/null')
      $q.notify({
        message: 'Неверный ID пульса',
        type: 'negative',
        position: 'bottom'
      })
      return
    }
    
    // Проверяем, что ID пульса является валидным UUID
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
    if (!uuidRegex.test(pulse.id)) {
      console.warn('Pulse ID is not a valid UUID')
      $q.notify({
        message: 'Неверный формат ID пульса',
        type: 'negative',
        position: 'bottom'
      })
      return
    }
    
    // Устанавливаем состояние загрузки
    isLoadingNews.value = true
    loadingMessage.value = 'Загружаем новости...'
    retryCount.value = 0
    
    // Убираем проверку существования пульса в userPulses.value
    // так как это может вызывать проблемы с вновь созданными пульсами
    
    const response = await pulseService.getPulseNews(pulse.id, { limit: 20 })
    console.log('Pulse news response:', response)
    console.log('Pulse news response.data:', response.data)
    
    // Проверяем первую новость из ответа
    if (response.data && response.data.data && response.data.data.length > 0) {
      console.log('First news from API:', response.data.data[0])
      console.log('First news content:', response.data.data[0].content)
    }
    
    // Проверяем структуру ответа
    if (response.data && response.data.success && response.data.data) {
      // Обрабатываем данные с API, преобразуем плоскую структуру в вложенную
      const apiNews = (response.data.data || []).map(news => ({
        ...news,
        // Преобразуем плоскую структуру в вложенную для совместимости с UI
        source: {
          id: news.source_id,
          name: news.source_name,
          domain: news.source_domain,
          logo_url: news.source_logo_url
        },
        category: {
          id: news.category_id,
          name: news.category_name,
          slug: news.category_slug,
          color: news.category_color,
          icon: news.category_icon
        },
        country: {
          flag: getCountryFlagBySource(news.source_domain),
          flag_emoji: news.country_flag_emoji
        },
        tags: news.tags || []
      }))
      
      
      pulseNews.value = apiNews
    } else if (Array.isArray(response.data)) {
      // Обрабатываем данные как массив, убеждаемся что у каждой новости есть теги
      const apiNews = response.data.map(news => ({
        ...news,
        // Преобразуем плоскую структуру в вложенную для совместимости с UI
        source: {
          id: news.source_id,
          name: news.source_name,
          domain: news.source_domain,
          logo_url: news.source_logo_url
        },
        category: {
          id: news.category_id,
          name: news.category_name,
          slug: news.category_slug,
          color: news.category_color,
          icon: news.category_icon
        },
        country: {
          flag: getCountryFlagBySource(news.source_domain),
          flag_emoji: news.country_flag_emoji
        },
        tags: news.tags || []
      }))
      pulseNews.value = apiNews
    } else {
      console.warn('Unexpected response structure')
      $q.notify({
        message: 'Неожиданная структура ответа сервера',
        type: 'negative',
        position: 'bottom'
      })
      return
    }
    
    lastUpdate.value = new Date().toLocaleTimeString('ru-RU')
    isLoadingNews.value = false
    loadingMessage.value = ''
  } catch (error) {
    isLoadingNews.value = false
    loadingMessage.value = ''
    
    const apiError = handleApiError(error, 'Ошибка загрузки новостей пульса')
    console.error('Error loading pulse news:', apiError)
    
    // Показываем более информативное сообщение об ошибке
    let errorMessage = apiError.message
    if (error.code === 'ECONNABORTED') {
      errorMessage = 'Превышено время ожидания. Попробуйте обновить страницу.'
    } else if (error.response?.status >= 500) {
      errorMessage = 'Ошибка сервера. Попробуйте позже.'
    } else if (!error.response) {
      errorMessage = 'Проблема с подключением. Проверьте интернет-соединение.'
    }
    
    $q.notify({
      message: errorMessage,
      type: 'negative',
      position: 'bottom',
      timeout: 5000,
      actions: [
        {
          label: 'Повторить',
          color: 'white',
          handler: () => {
            if (selectedPulse.value) {
              loadPulseNewsFromApi(selectedPulse.value)
            }
          }
        }
      ]
    })
  }
}

async function savePulseToApi() {
  if (!newPulse.value.name.trim()) {
    $q.notify({
      message: 'Введите название пульса',
      type: 'negative',
      position: 'bottom'
    })
    return
  }

  try {
    // Преобразуем реактивные массивы в обычные массивы
    const countryIds = Array.isArray(newPulse.value.countries) ? [...newPulse.value.countries] : []
    const categoryIds = Array.isArray(newPulse.value.categories) ? [...newPulse.value.categories] : []
    
    // Проверяем, что есть хотя бы одна страна
    if (countryIds.length === 0) {
      $q.notify({
        message: 'Выберите хотя бы одну страну',
        type: 'negative',
        position: 'bottom'
      })
      return
    }
    
    // Маппим страны на их источники
    const countryToSources = {
      1: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10], // Россия
      2: [11, 12, 13], // Беларусь
      3: [14, 15, 16], // Казахстан
      4: [17, 18], // Украина
      5: [19, 20], // Узбекистан
      6: [21, 22], // Кыргызстан
      7: [23, 24], // Армения
      8: [25, 26], // Азербайджан
    }
    
    // Получаем источники для выбранных стран
    const sourceIds = []
    countryIds.forEach(countryId => {
      if (countryToSources[countryId]) {
        sourceIds.push(...countryToSources[countryId])
      }
    })
    
    // Если источники не найдены, используем российские по умолчанию
    if (sourceIds.length === 0) {
      sourceIds.push(1, 2, 3) // РИА Новости, ТАСС, Интерфакс
    }
    
    const pulseData = {
      name: newPulse.value.name,
      description: newPulse.value.description,
      keywords: newPulse.value.keywords,
      refresh_interval_min: parseInt(30), // Убеждаемся, что это число
      source_ids: sourceIds, // Используем маппинг стран на источники
      category_ids: categoryIds,
      is_active: true,
      is_default: false
    }
    
    console.log('Sending pulse data:', pulseData) // Для отладки
    console.log('Selected countries:', countryIds)
    console.log('Mapped source IDs:', sourceIds)

    let response
    if (editingPulse.value) {
      response = await pulseService.updatePulse(editingPulse.value.id, pulseData)
      const index = userPulses.value.findIndex(p => p.id === editingPulse.value.id)
      if (index > -1) {
        // Проверяем структуру ответа: response.data.data содержит объект пульса
        if (response.data && response.data.success && response.data.data) {
          userPulses.value[index] = response.data.data
        } else {
          console.error('Invalid update response structure:', response.data)
        }
      }
      $q.notify({
        message: 'Пульс успешно обновлен',
        type: 'positive',
        position: 'bottom'
      })
    } else {
      response = await pulseService.createPulse(pulseData)
      console.log('Create pulse response:', response)
      console.log('Create pulse response.data:', response.data)
      console.log('Create pulse response.data.data:', response.data?.data)
      console.log('Create pulse response.data.data.id:', response.data?.data?.id)
      
      // Проверяем структуру ответа: response.data.data содержит объект пульса
      if (response.data && response.data.success && response.data.data && response.data.data.id) {
        userPulses.value.push(response.data.data)
        console.log('Pulse added to userPulses:', response.data.data)
        console.log('Current userPulses count:', userPulses.value.length)
        console.log('All userPulses:', userPulses.value)
      } else {
        console.error('Invalid response data structure:', response.data)
        $q.notify({
          message: 'Ошибка: неправильная структура ответа сервера',
          type: 'negative',
          position: 'bottom'
        })
        return
      }
      
      $q.notify({
        message: 'Пульс успешно создан',
        type: 'positive',
        position: 'bottom'
      })
    }

    cancelPulseDialog()
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка сохранения пульса')
    console.error('Error saving pulse:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
    })
  }
}

async function deletePulseFromApi(pulse) {
  try {
    await pulseService.deletePulse(pulse.id)
    const index = userPulses.value.findIndex(p => p.id === pulse.id)
    if (index > -1) {
      userPulses.value.splice(index, 1)
      $q.notify({
        message: 'Пульс удален',
        type: 'positive',
        position: 'bottom'
      })
    }
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка удаления пульса')
    console.error('Error deleting pulse:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
    })
  }
}

async function updatePulseNewsFromApi() {
  if (!selectedPulse.value) return
  
  isUpdating.value = true
  updateProgress.value = 0
  loadingMessage.value = 'Обновляем новости...'
  
  try {
    // Запускаем обновление пульса
    await pulseService.refreshPulse(selectedPulse.value.id)
    
    // Имитируем прогресс
    const interval = setInterval(() => {
      updateProgress.value += 0.1
      if (updateProgress.value >= 1) {
        clearInterval(interval)
        updateProgress.value = 0
      }
    }, 100)
    
    // Загружаем обновленные новости
    setTimeout(async () => {
      await loadPulseNewsFromApi(selectedPulse.value)
      isUpdating.value = false
      loadingMessage.value = ''
    }, 1000)
    
  } catch (error) {
    isUpdating.value = false
    loadingMessage.value = ''
    
    const apiError = handleApiError(error, 'Ошибка обновления новостей')
    console.error('Error updating pulse news:', apiError)
    
    // Показываем более информативное сообщение об ошибке
    let errorMessage = apiError.message
    if (error.code === 'ECONNABORTED') {
      errorMessage = 'Превышено время ожидания при обновлении.'
    } else if (error.response?.status >= 500) {
      errorMessage = 'Ошибка сервера при обновлении.'
    } else if (!error.response) {
      errorMessage = 'Проблема с подключением при обновлении.'
    }
    
    $q.notify({
      message: errorMessage,
      type: 'negative',
      position: 'bottom',
      timeout: 5000,
      actions: [
        {
          label: 'Повторить',
          color: 'white',
          handler: () => {
            updatePulseNewsFromApi()
          }
        }
      ]
    })
  }
}


onMounted(async () => {
  console.log('Component mounted, starting initialization...')
  
  // Инициализируем pulseNews как пустой массив
  pulseNews.value = []
  
  
  // Загружаем справочные данные
  console.log('Loading reference data...')
  await loadReferenceData()
  console.log('Reference data loaded. Countries:', countries.value.length, 'Categories:', categories.value.length)
  
  // Загружаем пульсы пользователя
  console.log('Loading user pulses...')
  await loadUserPulses()
  console.log('User pulses loaded:', userPulses.value.length)
  
  lastUpdate.value = new Date().toLocaleTimeString('ru-RU')
  console.log('Initialization completed')
})

// Функции для работы с новостями (скопированы из NewsPage.vue)
const cleanText = (text) => {
  if (!text) return ''
  
  // Удаляем HTML теги
  let cleaned = text.replace(/<[^>]*>/g, '')
  
  // Удаляем множественные пробелы и переносы строк
  cleaned = cleaned.replace(/\s+/g, ' ').trim()
  
  return cleaned
}

const cleanNewsContent = (content) => {
  if (!content) return ''
  
  // Удаляем HTML теги
  let cleaned = content.replace(/<[^>]*>/g, '')
  
  // Удаляем множественные пробелы и переносы строк
  cleaned = cleaned.replace(/\s+/g, ' ').trim()
  
  // Если контент слишком короткий или содержит много непечатаемых символов, показываем сообщение
  if (cleaned.length < 50) {
    return 'Полный текст новости недоступен. Рекомендуется прочитать оригинальную статью.'
  }
  
  return cleanText(cleaned)
}

const isValidImageUrl = (url) => {
  if (!url) return false
  
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

const openOriginalNews = (url) => {
  if (url) {
    window.open(url, '_blank')
  }
}

const isContentCorrupted = (content) => {
  if (!content) return false
  
  // Проверяем процент непечатаемых символов
  let nonPrintableCount = 0
  for (let i = 0; i < content.length; i++) {
    const charCode = content.charCodeAt(i)
    if (charCode < 32 && charCode !== 9 && charCode !== 10 && charCode !== 13) {
      nonPrintableCount++
    }
  }
  
  const nonPrintablePercentage = (nonPrintableCount / content.length) * 100
  return nonPrintablePercentage > 20 || content.length < 100
}
</script>

<style lang="scss" scoped>
.create-pulse-btn {
  min-height: 120px;
  min-width: 120px;
  border-radius: 50%;
  font-size: 1.1em;
  
  // Принудительно применяем белый цвет ко всем элементам
  &,
  & * {
    color: white !important;
  }
  
  .q-icon,
  .material-icons {
    font-size: 4em;
    color: white !important;
  }
  
  &:hover {
    transform: translateY(-4px) scale(1.05);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
    
    &,
    & * {
      color: white !important;
    }
  }
  
  &:focus,
  &:active,
  &.q-btn--active {
    &,
    & * {
      color: white !important;
    }
  }
}

// Дополнительный класс для принудительного белого цвета
.white-content {
  &,
  & *,
  & .q-btn__content,
  & .q-btn__content *,
  & .q-icon,
  & .material-icons,
  & span {
    color: white !important;
  }
}

.pulse-card {
  transition: all 0.3s ease;
  border-radius: 16px;
  
  &:hover {
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
    transform: translateY(-4px);
  }
}

.news-card {
  transition: all 0.3s ease;
  
  &:hover {
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }
}

.news-title {
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.news-description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
}

.news-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.country-flag {
  font-size: 1.2em;
}

.source-name {
  font-size: 0.9em;
}

.news-actions {
  display: flex;
  gap: 4px;
}

// Общие адаптивные стили
.modern-page {
  min-height: 100vh;
  overflow-x: hidden;
}

// Анимации для появления элементов
.fade-in-up {
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stagger-animation > * {
  animation-delay: calc(var(--animation-order, 0) * 0.1s);
}

// Адаптивные утилиты
.responsive-text {
  font-size: clamp(0.9rem, 2.5vw, 1.2rem);
}

.responsive-title {
  font-size: clamp(1.5rem, 4vw, 2.5rem);
}

// Адаптивные стили для разных размеров экрана
@media (max-width: 1200px) {
  .pulse-card {
    margin-bottom: 16px;
  }
}

@media (max-width: 900px) {
  .create-pulse-btn {
    min-width: 110px;
    min-height: 110px;
    
    &,
    & * {
      color: white !important;
    }
    
    .q-icon,
    .material-icons {
      font-size: 3.5em;
      color: white !important;
    }
  }
}

@media (max-width: 768px) {
  .create-pulse-btn {
    min-width: 100px;
    min-height: 100px;
    
    &,
    & * {
      color: white !important;
    }
    
    .q-icon,
    .material-icons {
      font-size: 3em;
      color: white !important;
    }
  }
  
  // Карточки пульсов на планшетах
  .pulse-card {
    margin-bottom: 12px;
  }
  
  // Новости - адаптация для планшетов
  .news-card {
    margin-bottom: 12px;
    
    .row.no-wrap {
      flex-direction: row;
    }
    
    .q-img {
      width: 100px !important;
      height: 100px !important;
    }
  }
}

@media (max-width: 600px) {
  .create-pulse-btn {
    min-width: 90px;
    min-height: 90px;
    
    &,
    & * {
      color: white !important;
    }
    
    .q-icon,
    .material-icons {
      font-size: 2.5em;
      color: white !important;
    }
  }
  
  // Карточки пульсов - одна колонка на мобильных
  .pulse-card {
    margin-bottom: 10px;
  }
  
  // Новости - вертикальная раскладка на мобильных
  .news-card {
    .row.no-wrap {
      flex-direction: column;
    }
    
    .q-img {
      width: 100% !important;
      height: 180px !important;
    }
    
    .news-title {
      font-size: 1.1em;
      line-height: 1.3;
    }
    
    .news-description {
      font-size: 0.9em;
      line-height: 1.4;
    }
  }
  
  // Заголовки и текст
  .text-h4 {
    font-size: 1.8em !important;
  }
  
  .text-h5 {
    font-size: 1.5em !important;
  }
  
  .text-h6 {
    font-size: 1.2em !important;
  }
}

@media (max-width: 480px) {
  .create-pulse-btn {
    min-width: 80px;
    min-height: 80px;
    
    &,
    & * {
      color: white !important;
    }
    
    .q-icon,
    .material-icons {
      font-size: 2.2em;
      color: white !important;
    }
  }
  
  // Уменьшаем отступы на маленьких экранах
  .q-pa-md {
    padding: 12px !important;
  }
  
  // Карточки новостей - компактнее
  .news-card {
    .q-img {
      height: 150px !important;
    }
    
    .q-card-section {
      padding: 12px !important;
    }
    
    .news-title {
      font-size: 1em;
    }
    
    .news-description {
      font-size: 0.85em;
    }
    
    .news-actions {
      gap: 2px;
    }
  }
  
  // Чипы - меньше размер
  .q-chip {
    font-size: 0.8em;
    padding: 2px 8px;
  }
}

@media (max-width: 360px) {
  .create-pulse-btn {
    min-width: 70px;
    min-height: 70px;
    
    &,
    & * {
      color: white !important;
    }
    
    .q-icon,
    .material-icons {
      font-size: 2em;
      color: white !important;
    }
  }
  
  // Очень маленькие экраны - минимальные отступы
  .q-pa-md {
    padding: 8px !important;
  }
  
  .news-card .q-img {
    height: 120px !important;
  }
}

// Дополнительная мобильная адаптация для IndexPage
@media (max-width: 599px) {
  .mobile-pulse-header {
    flex-wrap: wrap;
    
    .mobile-back-btn {
      margin-right: 8px !important;
    }
    
    .mobile-pulse-title-wrapper {
      flex: 1;
      min-width: 0;
      
      .mobile-pulse-title {
        font-size: 1.25rem !important;
        line-height: 1.3 !important;
        margin-bottom: 4px;
      }
      
      .mobile-pulse-desc {
        font-size: 0.875rem !important;
        line-height: 1.4 !important;
      }
    }
    
    .mobile-refresh-btn-wrapper {
      width: 100%;
      
      .mobile-refresh-btn {
        width: 100%;
        justify-content: center;
      }
    }
  }
  
  // Статистика пульса
  .status-card {
    .q-card-section {
      padding: 12px !important;
      flex-direction: column !important;
      
      .col, .col-auto {
        width: 100%;
        text-align: center;
        margin-bottom: 8px;
        
        &:last-child {
          margin-bottom: 0;
        }
      }
      
      .text-caption {
        font-size: 0.8rem !important;
      }
    }
  }
  
  // Карточки новостей в пульсе
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
    
    .mobile-pulse-news-section {
      padding: 16px !important;
    }
    
    .news-meta {
      font-size: 0.875rem !important;
      flex-wrap: wrap;
      gap: 8px;
      display: flex;
      align-items: center;
      
      .country-flag {
        font-size: 1rem !important;
      }
      
      .source-name {
        font-size: 0.9rem !important;
        font-weight: 600 !important;
      }
      
      .q-separator {
        display: none;
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
    
    .q-chip {
      font-size: 0.8rem !important;
      height: 28px !important;
    }
  }
  
  // Диалог создания пульса
  .q-dialog .q-card {
    .q-card-section {
      padding: 16px !important;
      
      .text-h6 {
        font-size: 1.2rem !important;
      }
    }
    
    .q-input, .q-select {
      margin-bottom: 12px !important;
    }
    
    .q-chip {
      font-size: 0.75rem !important;
      margin: 4px 2px !important;
    }
  }
}

@media (max-width: 400px) {
  .mobile-pulse-header {
    .mobile-pulse-title {
      font-size: 1.1rem !important;
    }
    
    .mobile-pulse-desc {
      font-size: 0.8rem !important;
    }
  }
  
  .news-card {
    .news-meta {
      font-size: 0.75rem !important;
      
      .source-name {
        font-size: 0.8rem !important;
      }
    }
    
    .news-title {
      font-size: 0.95rem !important;
    }
    
    .news-description {
      font-size: 0.8rem !important;
    }
    
    .q-chip {
      font-size: 0.7rem !important;
      height: 22px !important;
    }
  }
}


// Стили для изображений новостей
.news-image-wrapper {
  width: 100%;
  overflow: hidden;
  
  .news-image {
    width: 100%;
    transition: transform 0.3s ease;
  }
}

// Стили для новостей
.news-card {
  transition: all 0.3s ease;
  border: 1px solid var(--border-primary);
  overflow: hidden;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
    
    .news-image {
      transform: scale(1.05);
    }
  }
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

.news-title {
  font-weight: 600;
  line-height: 1.3;
  color: var(--text-primary);
}

.news-description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
}

.news-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.country-flag {
  font-size: 1.2em;
}

.source-name {
  font-size: 0.9em;
}

.news-actions {
  display: flex;
  gap: 4px;
}

// Стили для полного текста новости
.news-content {
  border-top: 1px solid var(--border-primary);
  padding-top: 16px;
  margin-top: 16px;
}

.news-content-text {
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.error-content {
  color: var(--q-negative);
  font-style: italic;
  background-color: var(--q-negative-light);
  padding: 8px;
  border-radius: 4px;
  border-left: 4px solid var(--q-negative);
}

// Стили для индикатора загрузки
.loading-card {
  min-width: 300px;
  max-width: 400px;
  border-radius: 16px;
  padding: 24px;
  
  .q-spinner-dots {
    animation: pulse 1.5s ease-in-out infinite;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
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
  }
  
  .news-description {
    font-size: 1.1rem;
    line-height: 1.6;
    margin-bottom: 16px;
  }
  
  .news-content-text {
    line-height: 1.8;
    white-space: pre-wrap;
    word-wrap: break-word;
  }
}

// Мобильная адаптация диалога
@media (max-width: 599px) {
  .dialog-news-content {
    padding: 16px;
    
    .news-title {
      font-size: 1.2rem;
    }
    
    .news-description {
      font-size: 1rem;
    }
  }
}
</style>