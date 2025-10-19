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
              <q-icon name="public" size="24px" />
            </div>
            <div class="info-text">
              <div class="info-label">Страны</div>
              <div class="info-value">{{ pulse.countries?.length || 0 }}</div>
            </div>
          </div>
        </BaseCard>

        <BaseCard class="info-card">
          <div class="info-content">
            <div class="info-icon">
              <q-icon name="category" size="24px" />
            </div>
            <div class="info-text">
              <div class="info-label">Категории</div>
              <div class="info-value">{{ pulse.categories?.length || 0 }}</div>
            </div>
          </div>
        </BaseCard>

        <BaseCard class="info-card">
          <div class="info-content">
            <div class="info-icon">
              <q-icon name="trending_up" size="24px" />
            </div>
            <div class="info-text">
              <div class="info-label">Новостей</div>
              <div class="info-value">{{ pulse.news_count || 0 }}</div>
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
          <BaseCard
            v-for="item in news.slice(0, 5)"
            :key="item.id"
            class="news-item"
            hover
            clickable
            @click="viewNewsItem(item)"
          >
            <div class="news-content">
              <h4 class="news-title">{{ item.title }}</h4>
              <p class="news-description">{{ item.description || item.content?.substring(0, 150) + '...' }}</p>
              <div class="news-meta">
                <span class="news-source">{{ item.source?.name || 'Неизвестный источник' }}</span>
                <span class="news-date">{{ formatDate(item.published_at) }}</span>
              </div>
            </div>
          </BaseCard>
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
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter, useRoute } from 'vue-router'
import { pulseService, newsService } from 'src/services/api'
import BaseCard from 'src/components/BaseCard.vue'

const $q = useQuasar()
const router = useRouter()
const route = useRoute()

// Состояние
const loading = ref(false)
const pulse = ref(null)
const news = ref([])
const newsLoading = ref(false)

// Методы
const loadNews = async (pulseId) => {
  try {
    newsLoading.value = true
    const response = await newsService.getNews({ pulse: pulseId })
    
    if (response.data && response.data.success && response.data.data) {
      news.value = response.data.data
    } else {
      news.value = []
    }
  } catch (error) {
    console.error('Ошибка загрузки новостей:', error)
    news.value = []
  } finally {
    newsLoading.value = false
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
      await loadNews(pulseId)
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
  if (item.url) {
    window.open(item.url, '_blank')
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
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--spacing-lg);
}

.info-card {
  transition: all var(--transition-normal);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
  }
}

.info-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
}

.info-icon {
  width: 48px;
  height: 48px;
  background: var(--gradient-primary);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  
  .q-icon {
    color: white;
  }
}

.info-text {
  flex: 1;
  
  .info-label {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    margin-bottom: var(--spacing-xs);
  }
  
  .info-value {
    font-size: var(--font-size-xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
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
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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
  
  .news-item {
    transition: all var(--transition-normal);
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: var(--shadow-md);
    }
  }
  
  .news-content {
    padding: var(--spacing-lg);
    
    .news-title {
      font-size: var(--font-size-lg);
      font-weight: var(--font-weight-semibold);
      color: var(--text-primary);
      margin-bottom: var(--spacing-sm);
      line-height: var(--line-height-tight);
    }
    
    .news-description {
      font-size: var(--font-size-base);
      color: var(--text-secondary);
      line-height: var(--line-height-relaxed);
      margin-bottom: var(--spacing-md);
    }
    
    .news-meta {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: var(--font-size-sm);
      color: var(--text-tertiary);
      
      .news-source {
        font-weight: var(--font-weight-medium);
      }
      
      .news-date {
        opacity: 0.8;
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
</style>
