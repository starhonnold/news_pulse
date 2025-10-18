<template>
  <q-layout view="lHh Lpr lFf" class="modern-layout">
    <q-header class="modern-header glass-effect">
      <q-toolbar class="modern-toolbar">
        <!-- Логотип и название -->
        <div class="flex items-center q-mr-md">
          <q-icon name="trending_up" size="32px" class="text-white q-mr-sm modern-logo-icon" />
          <span class="text-h6 text-white font-weight-bold">News Pulse</span>
        </div>

        <q-space />

        <!-- Вкладки навигации -->
        <q-tabs
          v-model="activeTab"
          class="modern-tabs"
          active-color="white"
          indicator-color="white"
          align="center"
          no-caps
          dense
        >
          <q-tab 
            name="pulses" 
            :label="$q.screen.gt.sm ? 'Пульсы' : ''" 
            icon="analytics"
            @click="navigateToPulses"
            class="modern-tab"
          />
          <q-tab 
            name="news" 
            :label="$q.screen.gt.sm ? 'Новости' : ''" 
            icon="newspaper"
            @click="navigateToNews"
            class="modern-tab"
          />
        </q-tabs>

            <q-space />

            <!-- Действия -->
        <div class="flex items-center">
          <!-- Уведомления -->
          <q-btn 
            flat 
            round 
            dense 
            icon="notifications_active" 
            class="modern-header-btn q-mr-sm"
            size="sm"
          >
            <q-badge color="red" floating class="modern-badge">3</q-badge>
          </q-btn>

          <!-- Меню пользователя -->
          <q-btn 
            flat 
            round 
            dense 
            icon="person" 
            class="modern-header-btn"
            size="sm"
          >
            <q-menu class="modern-menu" anchor="bottom right" self="top right">
              <q-list class="modern-menu-list">
                <q-item clickable v-close-popup class="modern-menu-item">
                  <q-item-section avatar>
                    <q-icon name="account_circle" color="primary" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>Профиль</q-item-label>
                    <q-item-label caption>Управление аккаунтом</q-item-label>
                  </q-item-section>
                </q-item>
                <q-item clickable v-close-popup class="modern-menu-item">
                  <q-item-section avatar>
                    <q-icon name="tune" color="primary" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>Настройки</q-item-label>
                    <q-item-label caption>Персонализация</q-item-label>
                  </q-item-section>
                </q-item>
                <q-separator class="modern-separator" />
                <q-item clickable v-close-popup class="modern-menu-item">
                  <q-item-section avatar>
                    <q-icon name="exit_to_app" color="negative" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>Выйти</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-menu>
          </q-btn>
        </div>
      </q-toolbar>
    </q-header>

    <q-page-container class="modern-page-container">
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const activeTab = ref('pulses')

// Определяем активную вкладку на основе текущего маршрута
onMounted(() => {
  updateActiveTab()
})

// Отслеживаем изменения маршрута
watch(() => route.path, () => {
  updateActiveTab()
})

const updateActiveTab = () => {
  if (route.path === '/news') {
    activeTab.value = 'news'
  } else {
    activeTab.value = 'pulses'
  }
}

const navigateToPulses = () => {
  router.push('/')
  activeTab.value = 'pulses'
}

const navigateToNews = () => {
  router.push('/news')
  activeTab.value = 'news'
}

// Обработка поиска (пока не используется)
// const handleSearch = () => {
//   if (searchQuery.value.trim()) {
//     // Здесь можно добавить логику поиска
//     console.log('Поиск:', searchQuery.value)
//   }
// }
</script>

<style lang="scss" scoped>
// Современный layout
.modern-layout {
  background: var(--bg-primary) !important;
}

// Современный хедер
.modern-header {
  background: var(--gradient-primary) !important;
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: var(--shadow-lg);
}

.modern-toolbar {
  min-height: 64px !important;
  padding: 0 24px !important;
}

// Современные вкладки
.modern-tabs {
  .q-tab {
    color: white !important;
    font-weight: 500 !important;
    text-transform: none !important;
    font-size: 1rem !important;
    padding: 8px 16px !important;
    border-radius: 8px !important;
    margin: 0 4px !important;
    transition: all 0.3s ease !important;
    
    // Текст вкладки
    .q-tab__content {
      color: white !important;
    }
    
    .q-tab__label {
      color: white !important;
    }
    
    .q-icon {
      color: white !important;
    }
    
    &:hover {
      color: white !important;
      background: rgba(255, 255, 255, 0.1) !important;
      
      .q-tab__label {
        color: white !important;
      }
      
      .q-icon {
        color: white !important;
      }
    }
    
    &.q-tab--active {
      color: white !important;
      font-weight: 600 !important;
      background: rgba(255, 255, 255, 0.15) !important;
      
      .q-tab__content {
        color: white !important;
      }
      
      .q-tab__label {
        color: white !important;
      }
      
      .q-icon {
        color: white !important;
      }
    }
  }
  
  .q-tab__indicator {
    background: white !important;
    height: 3px !important;
    border-radius: 2px !important;
    box-shadow: 0 0 10px rgba(255, 255, 255, 0.5);
  }
}

// Логотип
.modern-logo-icon {
  color: white !important;
  filter: drop-shadow(0 0 8px rgba(255, 255, 255, 0.3)) !important;
  animation: pulse 2s infinite !important;
}

// Иконки и текст вкладок
.modern-tab {
  .q-icon {
    font-size: 1.2rem !important;
    margin-right: 8px !important;
    color: white !important;
    filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.2)) !important;
    transition: all 0.3s ease !important;
    
    &:hover {
      filter: drop-shadow(0 0 8px rgba(255, 255, 255, 0.4)) !important;
      transform: scale(1.1) !important;
    }
  }
  
  // Текст под иконками
  .q-tab__label {
    color: white !important;
    font-weight: 500 !important;
  }
}

// Поиск
.search-input {
  min-width: 300px !important;
  
  .q-field__control {
    background: rgba(255, 255, 255, 0.1) !important;
    border: 1px solid rgba(255, 255, 255, 0.2) !important;
    border-radius: 24px !important;
    color: white !important;
    
    &:hover {
      background: rgba(255, 255, 255, 0.15) !important;
      border-color: rgba(255, 255, 255, 0.3) !important;
    }
    
    &.q-field--focused {
      background: rgba(255, 255, 255, 0.2) !important;
      border-color: white !important;
      box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.2) !important;
    }
  }
  
  // Убираем синий фон при фокусе на мобильных
  .q-field__control:before,
  .q-field__control:after {
    display: none !important;
  }
  
  .q-field__native {
    color: white !important;
    
    &::placeholder {
      color: rgba(255, 255, 255, 0.7) !important;
    }
  }
  
  .q-field__prepend {
    color: rgba(255, 255, 255, 0.8) !important;
  }
  
  .search-icon {
    color: white !important;
    filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.3)) !important;
    transition: all 0.3s ease !important;
    
    &:hover {
      filter: drop-shadow(0 0 8px rgba(255, 255, 255, 0.5)) !important;
      transform: scale(1.1) !important;
    }
  }
}

// Кнопки хедера
.modern-header-btn {
  color: rgba(255, 255, 255, 0.8) !important;
  transition: all 0.3s ease !important;
  
  .q-icon {
    color: white !important;
    filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.2)) !important;
    transition: all 0.3s ease !important;
  }
  
  &:hover {
    color: white !important;
    background: rgba(255, 255, 255, 0.1) !important;
    transform: scale(1.05) !important;
    
    .q-icon {
      filter: drop-shadow(0 0 8px rgba(255, 255, 255, 0.4)) !important;
      transform: scale(1.1) !important;
    }
  }
}

// Бейдж уведомлений
.modern-badge {
  font-size: 0.7rem !important;
  padding: 2px 6px !important;
  border-radius: 10px !important;
  box-shadow: 0 0 10px rgba(244, 67, 54, 0.5) !important;
}

// Современное меню
.modern-menu {
  border-radius: 12px !important;
  box-shadow: var(--shadow-xl) !important;
  border: 1px solid var(--border-primary) !important;
  overflow: hidden !important;
}

.modern-menu-list {
  padding: 8px !important;
  background: var(--bg-card) !important;
}

.modern-menu-item {
  border-radius: 8px !important;
  margin: 2px 0 !important;
  transition: all 0.2s ease !important;
  
  &:hover {
    background: var(--bg-tertiary) !important;
  }
}

.modern-separator {
  margin: 8px 0 !important;
  background: var(--border-primary) !important;
}

// Контейнер страницы
.modern-page-container {
  background: var(--bg-primary) !important;
  min-height: calc(100vh - 64px) !important;
}

// Адаптивность
@media (max-width: 1024px) {
  .modern-toolbar {
    padding: 0 16px !important;
  }
  
  .search-input {
    min-width: 200px !important;
  }
}

@media (max-width: 768px) {
  .modern-toolbar {
    padding: 0 12px !important;
    min-height: 56px !important;
  }
  
  .search-input {
    display: none !important;
  }
  
  .modern-tabs {
    .q-tab {
      padding: 6px 12px !important;
      font-size: 0.9rem !important;
      
      .q-icon {
        font-size: 1.1rem !important;
        margin-right: 6px !important;
      }
    }
  }
}

@media (max-width: 480px) {
  .modern-toolbar {
    padding: 0 8px !important;
    min-height: 52px !important;
  }
  
  .modern-tabs {
    .q-tab {
      padding: 4px 8px !important;
      font-size: 0.85rem !important;
      margin: 0 2px !important;
      
      .q-icon {
        font-size: 1rem !important;
        margin-right: 4px !important;
      }
    }
  }
  
  .modern-header-btn {
    padding: 4px !important;
  }
}
</style>