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
                      :color="category.category_color || 'grey'"
                      text-color="white"
                      dense
                      class="q-mr-xs"
                    >
                      <q-icon :name="category.category_icon || 'info'" class="q-mr-xs" />
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
        <div class="row items-center q-mb-md">
          <div class="col-auto">
            <q-btn
              flat
              round
              icon="arrow_back"
              @click="selectedPulse = null"
              class="q-mr-md"
            />
          </div>
          <div class="col">
            <div class="text-h5 text-weight-bold text-primary">
              {{ selectedPulse.name }}
            </div>
            <div class="text-subtitle2 text-grey-7">
              {{ selectedPulse.description }}
            </div>
          </div>
          <div class="col-auto">
            <q-btn
              :loading="isUpdating"
              color="primary"
              icon="refresh"
              label="Обновить"
              class="modern-btn gradient-btn"
              unelevated
              @click="updatePulseNews"
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
                  <div class="row no-wrap">
                    <!-- Изображение новости -->
                    <div v-if="news.image_url || news.image" class="col-auto">
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
                              :color="news.category.color || 'grey'"
                              text-color="white"
                              dense
                              class="q-ml-sm"
                            >
                              <q-icon :name="news.category.icon || 'info'" class="q-mr-xs" />
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

                        <!-- Теги и действия -->
                        <div class="row items-center justify-between">
                          <div class="col">
                            <div class="news-tags">
                              <q-chip
                                v-for="tag in (news.tags || []).slice(0, 3)"
                                :key="tag"
                                dense
                                outline
                                color="grey-7"
                                class="q-mr-xs"
                              >
                                {{ tag }}
                              </q-chip>
                              <span v-if="(news.tags || []).length > 3" class="text-grey-6 text-caption">
                                +{{ (news.tags || []).length - 3 }} еще
                              </span>
                            </div>
                          </div>
                          <div class="col-auto">
                            <div class="news-actions">
                              <q-btn
                                flat
                                dense
                                round
                                icon="visibility"
                                color="grey-6"
                                @click.stop="viewNews(news)"
                              >
                                <q-tooltip>Просмотров: {{ news.view_count }}</q-tooltip>
                              </q-btn>
                              <q-btn
                                flat
                                dense
                                round
                                icon="share"
                                color="grey-6"
                                @click.stop="shareNews(news)"
                              />
                              <q-btn
                                flat
                                dense
                                round
                                icon="bookmark_border"
                                color="grey-6"
                                @click.stop="bookmarkNews(news)"
                              />
                            </div>
                          </div>
                        </div>
                      </q-card-section>
                    </div>
                  </div>

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

          <div class="text-subtitle2 q-mb-sm">Выберите страны: ({{ countries.length }} доступно)</div>
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

          <div class="text-subtitle2 q-mb-sm">Выберите категории: ({{ categories.length }} доступно)</div>
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
              :color="newPulse.categories.includes(category.id) ? category.color : 'grey-3'"
              :text-color="newPulse.categories.includes(category.id) ? 'white' : 'black'"
              class="q-ma-xs"
            >
              <q-icon :name="category.icon" class="q-mr-xs" />
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
    <q-dialog v-model="showNewsDialog" :maximized="$q.platform.is.mobile">
      <q-card style="min-width: 800px; max-width: 1000px">
        <q-card-section class="row items-center q-pb-none">
          <div class="col">
            <div class="text-h6">{{ selectedNews?.title }}</div>
          </div>
          <div class="col-auto">
            <q-btn icon="close" flat round dense @click="showNewsDialog = false" />
          </div>
        </q-card-section>

        <q-card-section v-if="selectedNews">
          <!-- Контент новости -->
          <div class="text-body1 q-mb-md" style="line-height: 1.6">
            {{ selectedNews.content }}
          </div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn
            color="primary"
            :href="selectedNews?.url"
            target="_blank"
            icon="open_in_new"
            label="Читать на источнике"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
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

// Данные для создания пульса
const newPulse = ref({
  name: '',
  description: '',
  countries: [],
  categories: [],
  keywords: '',
})

// Справочные данные
const countries = ref([])
const categories = ref([])



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
}

function togglePulseCategory(categoryId) {
  const index = newPulse.value.categories.indexOf(categoryId)
  if (index > -1) {
    newPulse.value.categories.splice(index, 1)
  } else {
    newPulse.value.categories.push(categoryId)
  }
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
  selectedNews.value = news
  showNewsDialog.value = true
}

function viewNews(news) {
  console.log('Просмотр новости:', news.id)
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
    
    // Убираем проверку существования пульса в userPulses.value
    // так как это может вызывать проблемы с вновь созданными пульсами
    
    const response = await pulseService.getPulseNews(pulse.id, { limit: 20 })
    console.log('Pulse news response:', response)
    console.log('Pulse news response.data:', response.data)
    
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
          color: news.category_color
        },
        country: {
          flag: getCountryFlagBySource(news.source_domain)
        },
        tags: news.tags || []
      }))
      pulseNews.value = apiNews
    } else if (Array.isArray(response.data)) {
      // Обрабатываем данные как массив, убеждаемся что у каждой новости есть теги
      const apiNews = response.data.map(news => ({
        ...news,
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
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка загрузки новостей пульса')
    console.error('Error loading pulse news:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
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
    }, 1000)
    
  } catch (error) {
    const apiError = handleApiError(error, 'Ошибка обновления новостей')
    console.error('Error updating pulse news:', apiError)
    
    $q.notify({
      message: apiError.message,
      type: 'negative',
      position: 'bottom'
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
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.news-description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
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
</style>