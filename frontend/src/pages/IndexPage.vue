<template>
  <q-page class="pulses-page">
          <!-- Заголовок страницы -->
    <transition name="page-header" appear>
          <div class="page-header">
              <div class="header-content">
                <div class="title-section">
                  <q-icon
                    name="add_circle_outline"
                    size="28px"
                    class="add-pulse-icon"
                @click="showCreatePulse = true"
                  >
                    <q-tooltip>Создать новый пульс</q-tooltip>
                  </q-icon>
              <h1 class="page-title">Мои пульсы</h1>
                  </div>
              <p class="page-subtitle">
                  Управляйте своими персональными агрегаторами новостей
              </p>
                    </div>
              <div class="header-actions">
                <q-input
                  v-model="searchQuery"
                  placeholder="Поиск пульсов..."
                  outlined
                        dense
                  class="search-input"
                  @input="filterPulses"
                >
                  <template v-slot:prepend>
                    <q-icon name="search" />
                  </template>
                  <template v-slot:append v-if="searchQuery">
                    <q-icon 
                      name="clear" 
                      class="cursor-pointer" 
                      @click="clearSearch"
                    />
                  </template>
                </q-input>
            </div>
          </div>
    </transition>


    <!-- Индикатор загрузки -->
    <transition name="loading" appear>
      <div v-if="loading" class="loading-section">
              <q-spinner-dots color="primary" size="40px" />
        <p class="loading-text">Загружаем пульсы...</p>
              </div>
    </transition>

    <!-- Список пульсов -->
          <div v-if="!loading && filteredPulses.length > 0" class="pulses-container">
      <transition-group
        name="pulse-list"
        tag="div"
        class="pulses-grid"
        appear
      >
        <BaseCard
          v-for="(pulse, index) in filteredPulses"
          :key="pulse.id"
          class="pulse-card"
          hover
          clickable
          :style="{ '--animation-delay': `${index * 0.1}s` }"
          @click="viewPulse(pulse)"
        >
          <div class="pulse-content">
            <!-- Зона 1: Заголовок + действия -->
            <div class="pulse-header">
              <div class="pulse-title-section">
                <h3 class="pulse-title">{{ pulse.name }}</h3>
                        </div>
              <div class="pulse-actions">
                <q-icon
                  name="edit"
                  size="16px"
                  class="action-icon edit-icon"
                  @click.stop="editPulse(pulse)"
                >
                  <q-tooltip>Редактировать</q-tooltip>
                </q-icon>
                <q-icon
                  name="delete"
                  size="16px"
                  class="action-icon delete-icon"
                  @click.stop="deletePulse(pulse)"
                >
                  <q-tooltip>Удалить</q-tooltip>
                </q-icon>
              </div>
                  </div>

            <!-- Зона 2: Описание (1 строка) -->
            <div class="pulse-description">
              <p>{{ pulse.description || 'Нет описания' }}</p>
                            </div>

            <!-- Зона 3: Мета-информация (чипы в одну строку) -->
            <div class="pulse-meta-chips">
              <div class="meta-chips">
                <q-chip size="xs" color="primary" text-color="white">
                  <q-icon name="public" size="8px" class="q-mr-xs" />
                  {{ pulse.countries?.length || 0 }}
                </q-chip>
                <q-chip size="xs" color="secondary" text-color="white">
                  <q-icon name="category" size="8px" class="q-mr-xs" />
                  {{ pulse.categories?.length || 0 }}
                </q-chip>
                          </div>
              <div class="keyword-chips" v-if="pulse.keywords">
                            <q-chip
                  v-for="keyword in pulse.keywords.split(',').slice(0, 2)"
                  :key="keyword"
                  size="xs"
                  color="accent"
                              text-color="white"
                  class="keyword-chip"
                            >
                  {{ keyword.trim() }}
                            </q-chip>
                <span v-if="pulse.keywords.split(',').length > 2" class="more-keywords">
                  +{{ pulse.keywords.split(',').length - 2 }}
                </span>
                          </div>
                        </div>
                        </div>
        </BaseCard>
      </transition-group>
                        </div>

    <!-- Пустое состояние -->
    <transition name="empty-state" appear>
          <div v-if="!loading && filteredPulses.length === 0" class="empty-state">
        <div class="empty-content">
          <q-icon name="analytics" size="80px" class="empty-icon" />
          <h3 class="empty-title">У вас пока нет пульсов</h3>
          <p class="empty-description">
            Создайте свой первый пульс для отслеживания новостей по интересующим вас темам
          </p>
          <q-icon
            name="add_circle_outline"
            size="40px"
            class="empty-add-icon"
            @click="showCreatePulse = true"
          />
                        </div>
                      </div>
    </transition>

    <!-- Диалог создания пульса -->
    <q-dialog v-model="showCreatePulse" persistent :maximized="$q.platform.is.mobile">
      <q-card 
        :style="$q.platform.is.mobile ? '' : 'min-width: 600px; max-width: 800px'" 
        class="modern-card glass-effect"
      >
        <q-card-section>
          <div class="text-h6">{{ editingPulse ? 'Редактировать пульс' : 'Создать новый пульс' }}</div>
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
          <q-btn color="primary" :label="editingPulse ? 'Сохранить изменения' : 'Создать пульс'" @click="savePulseMain" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter, useRoute } from 'vue-router'
import { pulseService, referenceService, handleApiError } from 'src/services/api'
import BaseCard from 'src/components/BaseCard.vue'

const $q = useQuasar()
const router = useRouter()
const route = useRoute()

// Состояние
const showCreatePulse = ref(false)
const editingPulse = ref(null)
const loading = ref(false)
const pulses = ref([])
const searchQuery = ref('')
const filteredPulses = ref([])

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
  if (newValue) {
    newPulse.value.countries = countries.value.map(country => country.id)
  } else {
    newPulse.value.countries = []
  }
})

watch(selectAllCategories, (newValue) => {
  if (newValue) {
    newPulse.value.categories = categories.value.map(category => category.id)
  } else {
    newPulse.value.categories = []
  }
})

// Методы для поиска
const filterPulses = () => {
  if (!searchQuery.value.trim()) {
    filteredPulses.value = [...pulses.value]
    return
  }
  
  const query = searchQuery.value.toLowerCase().trim()
  filteredPulses.value = pulses.value.filter(pulse => 
    pulse.name.toLowerCase().includes(query) ||
    (pulse.description && pulse.description.toLowerCase().includes(query)) ||
    (pulse.keywords && pulse.keywords.toLowerCase().includes(query))
  )
}

const clearSearch = () => {
  searchQuery.value = ''
  filteredPulses.value = [...pulses.value]
}

// Методы для работы с пульсами
const loadPulses = async () => {
  try {
    loading.value = true
    const response = await pulseService.getUserPulses()
    
    console.log('Response from getUserPulses:', response)
    
          if (response.data && response.data.success && response.data.data) {
            pulses.value = response.data.data
            filteredPulses.value = [...response.data.data]
            console.log('Loaded pulses:', pulses.value)
          } else {
            pulses.value = []
            filteredPulses.value = []
            console.log('No pulses found or invalid response structure')
          }
  } catch (error) {
    console.error('Ошибка загрузки пульсов:', error)
    pulses.value = []
    filteredPulses.value = []
    $q.notify({
      message: 'Не удалось загрузить пульсы',
      type: 'negative',
      position: 'bottom'
    })
  } finally {
    loading.value = false
  }
}

const viewPulse = (pulse) => {
  // Переход к странице пульса через Vue Router
  router.push(`/pulse/${pulse.id}`)
}

const editPulse = (pulse) => {
  editingPulse.value = pulse
  newPulse.value = {
    name: pulse.name,
    description: pulse.description || '',
    countries: pulse.countries || [],
    categories: pulse.categories || [],
    keywords: pulse.keywords || '',
  }
  showCreatePulse.value = true
}

const deletePulse = async (pulse) => {
  $q.dialog({
    title: 'Подтверждение удаления',
    message: `Вы уверены, что хотите удалить пульс "${pulse.name}"?`,
    persistent: true,
    ok: {
      label: 'Удалить',
      color: 'negative'
    },
    cancel: {
      label: 'Отмена',
      color: 'grey'
    }
  }).onOk(async () => {
    try {
      const response = await pulseService.deletePulse(pulse.id)
      
            if (response.data && response.data.success) {
              pulses.value = pulses.value.filter(p => p.id !== pulse.id)
              filteredPulses.value = filteredPulses.value.filter(p => p.id !== pulse.id)
              $q.notify({
                message: 'Пульс удален',
                type: 'positive',
                position: 'bottom'
              })
            }
    } catch (error) {
      console.error('Ошибка удаления пульса:', error)
      $q.notify({
        message: 'Не удалось удалить пульс',
        type: 'negative',
        position: 'bottom'
      })
    }
  })
}

// Методы управления пульсами (из оригинального файла)
const togglePulseCountry = (countryId) => {
  const index = newPulse.value.countries.indexOf(countryId)
  if (index > -1) {
    newPulse.value.countries.splice(index, 1)
  } else {
    newPulse.value.countries.push(countryId)
  }
  updateSelectAllCountries()
}

const togglePulseCategory = (categoryId) => {
  const index = newPulse.value.categories.indexOf(categoryId)
  if (index > -1) {
    newPulse.value.categories.splice(index, 1)
  } else {
    newPulse.value.categories.push(categoryId)
  }
  updateSelectAllCategories()
}

const updateSelectAllCountries = () => {
  selectAllCountries.value = newPulse.value.countries.length === countries.value.length && countries.value.length > 0
}

const updateSelectAllCategories = () => {
  selectAllCategories.value = newPulse.value.categories.length === categories.value.length && categories.value.length > 0
}

const getCategoryIcon = (iconName) => {
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
  
  if (iconMap[iconName]) {
    return iconMap[iconName]
  }
  
  const lowerIconName = iconName.toLowerCase()
  for (const [key, value] of Object.entries(iconMap)) {
    if (lowerIconName.includes(key)) {
      return value
    }
  }
  
  return 'info'
}

const getCategoryColor = (quasarColor) => {
  if (!quasarColor) return '#1976d2'
  
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

const savePulseMain = async () => {
  if (!newPulse.value.name.trim()) {
    $q.notify({
      message: 'Введите название пульса',
      type: 'negative',
      position: 'bottom'
    })
    return
  }

  try {
    const countryIds = Array.isArray(newPulse.value.countries) ? [...newPulse.value.countries] : []
    const categoryIds = Array.isArray(newPulse.value.categories) ? [...newPulse.value.categories] : []
    
    if (countryIds.length === 0) {
      $q.notify({
        message: 'Выберите хотя бы одну страну',
        type: 'negative',
        position: 'bottom'
      })
      return
    }
    
    const countryToSources = {
      1: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
      2: [11, 12, 13],
      3: [14, 15, 16],
      4: [17, 18],
      5: [19, 20],
      6: [21, 22],
      7: [23, 24],
      8: [25, 26],
    }
    
    const sourceIds = []
    countryIds.forEach(countryId => {
      if (countryToSources[countryId]) {
        sourceIds.push(...countryToSources[countryId])
      }
    })
    
    if (sourceIds.length === 0) {
      sourceIds.push(1, 2, 3)
    }
    
    const pulseData = {
      name: newPulse.value.name,
      description: newPulse.value.description,
      keywords: newPulse.value.keywords,
      refresh_interval_min: 30,
      source_ids: sourceIds,
      category_ids: categoryIds,
      is_active: true,
      is_default: false
    }

    let response
    if (editingPulse.value) {
      // Редактирование существующего пульса
      response = await pulseService.updatePulse(editingPulse.value.id, pulseData)
        } else {
      // Создание нового пульса
      response = await pulseService.createPulse(pulseData)
        }
    
    if (response.data && response.data.success) {
      $q.notify({
        message: editingPulse.value ? 'Пульс обновлен!' : 'Пульс успешно создан!',
        type: 'positive',
        position: 'bottom'
      })
      
      // Перезагружаем список пульсов
      await loadPulses()
      cancelPulseDialog()
      } else {
        $q.notify({
          message: 'Ошибка: неправильная структура ответа сервера',
          type: 'negative',
          position: 'bottom'
        })
    }
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

const cancelPulseDialog = () => {
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

// Загрузка справочных данных
async function loadReferenceData() {
  try {
    const categoriesResponse = await referenceService.getCategories()
    if (categoriesResponse && categoriesResponse.data && categoriesResponse.data.success && Array.isArray(categoriesResponse.data.data)) {
      categories.value = categoriesResponse.data.data
    }
    
    const countriesResponse = await referenceService.getCountries()
    if (countriesResponse && countriesResponse.data && countriesResponse.data.success && Array.isArray(countriesResponse.data.data)) {
      countries.value = countriesResponse.data.data
    }
  } catch (error) {
    console.error('Error loading reference data:', error)
  }
}

const editPulseById = async (pulseId) => {
  try {
    const response = await pulseService.getPulseById(pulseId)
    if (response.data && response.data.success && response.data.data) {
      const pulse = response.data.data
      editingPulse.value = pulse
      newPulse.value = {
        name: pulse.name,
        description: pulse.description || '',
        countries: pulse.countries?.map(c => c.id) || [],
        categories: pulse.categories?.map(c => c.id) || [],
        keywords: pulse.keywords || '',
      }
      showCreatePulse.value = true
    } else {
    $q.notify({
        message: 'Пульс не найден',
      type: 'negative',
        position: 'bottom'
      })
    }
  } catch (error) {
    console.error('Ошибка загрузки пульса для редактирования:', error)
    $q.notify({
      message: 'Не удалось загрузить пульс для редактирования',
      type: 'negative',
      position: 'bottom'
    })
  }
}

onMounted(async () => {
  console.log('IndexPage mounted, loading data...')
  await Promise.all([
    loadPulses(),
    loadReferenceData()
  ])
  console.log('Data loading completed')
  
  // Проверяем, есть ли параметр редактирования в URL
  const editPulseId = route.query.edit
  if (editPulseId) {
    await editPulseById(editPulseId)
    // Очищаем URL от параметра редактирования
    router.replace({ path: '/pulses' })
  }
})
</script>

<style lang="scss" scoped>
// === СТРАНИЦА ПУЛЬСОВ ===
.pulses-page {
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

// Анимация заголовка
.page-header-enter-active {
  transition: all 0.6s ease;
}

.page-header-enter-from {
  opacity: 0;
  transform: translateY(-20px);
}

.header-content {
  flex: 1;
  min-width: 300px;
  
  .title-section {
  display: flex;
    align-items: center;
    gap: var(--spacing-md);
    margin-bottom: var(--spacing-sm);
  }
  
  .page-title {
    font-size: var(--font-size-4xl);
    font-weight: var(--font-weight-extrabold);
    color: var(--text-primary);
    margin: 0;
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

.header-actions {
  flex-shrink: 0;
  
  .search-input {
    min-width: 300px;
    max-width: 400px;
    
    .q-field__control {
      background: var(--bg-elevated);
      border-color: var(--border-primary);
  
  &:hover {
        border-color: var(--accent-color);
      }
    }
    
    .q-field__native {
  color: var(--text-primary);
    }
  }
}

.add-pulse-icon {
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-normal);
  flex-shrink: 0;
  
  &:hover {
    color: var(--accent-color);
    transform: scale(1.1);
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

// Анимация загрузки
.loading-enter-active {
  transition: all 0.4s ease;
}

.loading-enter-from {
  opacity: 0;
  transform: scale(0.8);
}

// === СПИСОК ПУЛЬСОВ ===
.pulses-container {
  width: 100%;
  margin-top: var(--spacing-lg);
  }
  
  .pulses-grid {
  display: grid !important;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)) !important;
  gap: var(--spacing-lg) !important;
  width: 100% !important;
  align-items: start !important;
}

.pulse-card {
  transition: all var(--transition-normal);
  animation: pulseSlideIn 0.6s ease-out forwards;
  animation-delay: var(--animation-delay, 0s);
  opacity: 0;
  transform: translateY(30px);
  height: 100px !important;
  min-height: 100px !important;
  max-height: 120px !important;
  width: 100% !important;
  max-width: 100% !important;
  position: relative;
  
  &:hover {
    transform: translateY(-4px);
    box-shadow: var(--shadow-lg);
  }
}

// === АНИМАЦИИ ===
@keyframes pulseSlideIn {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// Transition-group анимации
.pulse-list-enter-active,
.pulse-list-leave-active {
  transition: all 0.4s ease;
}

.pulse-list-enter-from {
  opacity: 0;
  transform: translateY(30px);
}

.pulse-list-leave-to {
  opacity: 0;
  transform: translateY(-30px);
}

.pulse-list-move {
  transition: transform 0.4s ease;
}

.pulse-content {
  padding: 6px !important;
  height: 80px !important;
  min-height: 80px !important;
  max-height: 80px !important;
  display: flex;
  flex-direction: column;
  gap: 4px;
  position: relative;
}

.pulse-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  height: 40px !important;
  min-height: 40px !important;
  max-height: 40px !important;
}

.pulse-title-section {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  height: 20px !important;
  
  .pulse-title {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin: 0;
    line-height: 20px !important;
    height: 20px !important;
  }
}

.pulse-description {
  flex-shrink: 0;
  height: 16px !important;
  min-height: 16px !important;
  max-height: 16px !important;
  
  p {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    line-height: 16px !important;
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    height: 16px !important;
  }
}

.pulse-meta-chips {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  overflow: hidden;
  height: 24px !important;
  min-height: 24px !important;
  max-height: 24px !important;
}

.meta-chips {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
  height: 24px !important;
}

.keyword-chips {
  display: flex;
  gap: 1px;
  align-items: center;
  flex: 1;
  overflow-x: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
  
  &::-webkit-scrollbar {
    display: none;
  }
  
  .keyword-chip {
    font-size: var(--font-size-xs);
    flex-shrink: 0;
    height: 16px !important;
    min-height: 16px !important;
    max-height: 16px !important;
  }
  
  .more-keywords {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    font-style: italic;
    flex-shrink: 0;
  }
}

// Принудительные стили для чипов
:deep(.q-chip) {
  height: 20px !important;
  min-height: 20px !important;
  max-height: 20px !important;
  font-size: var(--font-size-xs) !important;
  padding: 0 6px !important;
  
  .q-chip__content {
    height: 20px !important;
    min-height: 20px !important;
    max-height: 20px !important;
    line-height: 20px !important;
  }
  
  .q-icon {
    font-size: 12px !important;
    height: 12px !important;
    width: 12px !important;
  }
}

.pulse-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;

  .action-icon {
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-normal);
    
    &:hover {
      transform: scale(1.1);
    }
    
    &.edit-icon:hover {
      color: var(--accent-color);
    }
    
    &.delete-icon:hover {
      color: var(--error-color);
    }
  }
}

// === ПУСТОЕ СОСТОЯНИЕ ===
.empty-state {
      display: flex;
      align-items: center;
        justify-content: center;
  min-height: 400px;
  padding: var(--spacing-5xl);
}

.empty-content {
        text-align: center;
  max-width: 500px;
  
  .empty-icon {
    color: var(--text-tertiary);
    margin-bottom: var(--spacing-xl);
    animation: emptyIconFloat 3s ease-in-out infinite;
  }
  
  .empty-title {
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin-bottom: var(--spacing-md);
  }
  
  .empty-description {
    font-size: var(--font-size-lg);
    color: var(--text-secondary);
    line-height: var(--line-height-relaxed);
    margin-bottom: var(--spacing-3xl);
  }
  
.empty-add-icon {
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-normal);
  
  &:hover {
    color: var(--accent-color);
    transform: scale(1.1);
  }
}
}

// Анимация для пустого состояния
.empty-state-enter-active {
  transition: all 0.8s ease;
}

.empty-state-enter-from {
  opacity: 0;
  transform: scale(0.9);
}

@keyframes emptyIconFloat {
  0%, 100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-10px);
  }
}

// === АДАПТИВНОСТЬ ===
@media (max-width: 1024px) {
  .pulses-page {
    padding: var(--spacing-lg);
  }
  
  .pulses-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--spacing-md);
  }
  
  .pulse-card {
    height: 55px !important;
    min-height: 55px !important;
    max-height: 55px !important;
  }
  
  .pulse-content {
    height: 55px !important;
    min-height: 55px !important;
    max-height: 55px !important;
  }
  
  .header-actions .search-input {
    min-width: 250px;
    max-width: 300px;
  }
}

@media (max-width: 768px) {
  .pulses-page {
    padding: var(--spacing-md);
  }
  
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: var(--spacing-lg);
  }
  
  .header-actions {
    align-self: stretch;
    
    .search-input {
      min-width: 100%;
      max-width: 100%;
    }
  }
  
  .title-section {
    gap: var(--spacing-sm);
  }
  
  .page-title {
    font-size: var(--font-size-3xl) !important;
  }
  
  .page-subtitle {
    font-size: var(--font-size-base) !important;
  }
  
  .add-pulse-icon {
    font-size: 24px !important;
  }
  
  .pulses-grid {
    grid-template-columns: 1fr;
    gap: var(--spacing-md);
  }
  
  .pulse-card {
    height: 65px !important;
    min-height: 65px !important;
    max-height: 65px !important;
  }
  
  .pulse-content {
    height: 65px !important;
    min-height: 65px !important;
    max-height: 65px !important;
  }
  
  .pulse-meta-chips {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--spacing-xs);
  }
  
  .keyword-chips {
  width: 100%;
  }
}

@media (max-width: 480px) {
  .pulses-page {
    padding: var(--spacing-sm);
  }
  
  .page-title {
    font-size: var(--font-size-2xl) !important;
  }
  
  .page-subtitle {
    font-size: var(--font-size-sm) !important;
  }
  
  .pulse-card {
    height: 100px !important;
    min-height: 100px !important;
    max-height: 100px !important;
  }
  
  .pulse-content {
    height: 100px !important;
    min-height: 100px !important;
    max-height: 100px !important;
  }
  
  .pulse-title {
    font-size: var(--font-size-base) !important;
  }
  
  .pulse-description p {
    font-size: var(--font-size-sm) !important;
  }
  
  .pulse-header {
    height: 30px !important;
    min-height: 30px !important;
    max-height: 30px !important;
  }
  
  .pulse-title-section {
    height: 30px !important;
    
    .pulse-title {
      line-height: 30px !important;
      height: 30px !important;
    }
  }
  
  .pulse-description {
    height: 25px !important;
    min-height: 25px !important;
    max-height: 25px !important;
    
    p {
      line-height: 25px !important;
      height: 25px !important;
    }
  }
  
  .pulse-meta-chips {
    height: 35px !important;
    min-height: 35px !important;
    max-height: 35px !important;
  }
  
  .meta-chips {
    height: 35px !important;
  }
  
  // Увеличиваем размеры чипов для мобильных
  :deep(.q-chip) {
    height: 22px !important;
    min-height: 22px !important;
    max-height: 22px !important;
    font-size: var(--font-size-sm) !important;
    padding: 0 8px !important;
    
    .q-chip__content {
      height: 28px !important;
      min-height: 28px !important;
      max-height: 28px !important;
      line-height: 28px !important;
    }
    
    .q-icon {
      font-size: 16px !important;
      height: 16px !important;
      width: 16px !important;
    }
  }
}
</style>