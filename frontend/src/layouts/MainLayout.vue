<template>
  <q-layout view="lHh Lpr lFf" class="bg-grey-1">
    <q-header class="dark-blue-header">
      <q-toolbar class="mobile-toolbar">

        <q-toolbar-title class="flex items-center mobile-title">
          <!-- Вкладки навигации -->
          <q-tabs
            v-model="activeTab"
            class="text-white mobile-tabs"
            active-color="white"
            indicator-color="white"
            align="left"
            no-caps
          >
            <q-tab 
              name="pulses" 
              :label="$q.screen.gt.xs ? 'Пульсы' : ''" 
              icon="dashboard"
              @click="navigateToPulses"
              class="mobile-tab"
            />
            <q-tab 
              name="news" 
              :label="$q.screen.gt.xs ? 'Новости' : ''" 
              icon="article"
              @click="navigateToNews"
              class="mobile-tab"
            />
          </q-tabs>
        </q-toolbar-title>

        <q-space />

        <!-- Уведомления -->
        <q-btn flat round dense icon="notifications" class="modern-btn mobile-header-btn" size="sm">
          <q-badge color="red" floating class="neon-glow mobile-badge">3</q-badge>
        </q-btn>

        <!-- Меню пользователя -->
        <q-btn flat round dense icon="account_circle" class="q-ml-xs modern-btn mobile-header-btn" size="sm">
          <q-menu>
            <q-list style="min-width: 200px">
              <q-item clickable v-close-popup>
                <q-item-section avatar>
                  <q-icon name="person" />
                </q-item-section>
                <q-item-section>Профиль</q-item-section>
              </q-item>
              <q-item clickable v-close-popup>
                <q-item-section avatar>
                  <q-icon name="settings" />
                </q-item-section>
                <q-item-section>Настройки</q-item-section>
              </q-item>
              <q-separator />
              <q-item clickable v-close-popup>
                <q-item-section avatar>
                  <q-icon name="logout" />
                </q-item-section>
                <q-item-section>Выйти</q-item-section>
              </q-item>
            </q-list>
          </q-menu>
        </q-btn>
      </q-toolbar>
    </q-header>


    <q-page-container>
      <router-view />
    </q-page-container>

  </q-layout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const activeTab = ref('pulses')

// Определяем активную вкладку на основе текущего маршрута
onMounted(() => {
  if (route.path === '/news') {
    activeTab.value = 'news'
  } else {
    activeTab.value = 'pulses'
  }
})

const navigateToPulses = () => {
  router.push('/')
  activeTab.value = 'pulses'
}

const navigateToNews = () => {
  router.push('/news')
  activeTab.value = 'news'
}
</script>

<style lang="scss" scoped>
.search-input {
  .q-field__control {
    border-radius: 20px;
  }
}

// Стили для вкладок в хедере
.q-tabs {
  .q-tab {
    color: rgba(255, 255, 255, 0.7) !important;
    font-weight: 500 !important;
    text-transform: none !important;
    font-size: 1rem !important;
    
    &:hover {
      color: rgba(255, 255, 255, 0.9) !important;
    }
    
    &.q-tab--active {
      color: white !important;
      font-weight: 600 !important;
    }
  }
  
  .q-tab__indicator {
    background: white !important;
    height: 3px !important;
    border-radius: 2px !important;
  }
}

// Стили для иконок вкладок
.q-tab .q-icon {
  font-size: 1.2rem !important;
  margin-right: 8px !important;
}

// Мобильная адаптация хедера
@media (max-width: 599px) {
  .mobile-toolbar {
    min-height: 56px !important;
    padding: 0 8px !important;
  }
  
  .mobile-title {
    flex: 1;
    overflow-x: auto;
    
    .mobile-tabs {
      .q-tab {
        min-width: auto !important;
        padding: 0 12px !important;
        font-size: 0.875rem !important;
        
        .q-icon {
          font-size: 1.1rem !important;
          margin-right: 4px !important;
        }
      }
    }
  }
  
  .mobile-header-btn {
    .q-btn__content {
      font-size: 1.1rem !important;
    }
  }
  
  .mobile-badge {
    font-size: 0.65rem !important;
    padding: 2px 4px !important;
  }
}

@media (max-width: 400px) {
  .mobile-toolbar {
    min-height: 52px !important;
    padding: 0 4px !important;
  }
  
  .mobile-tabs {
    .q-tab {
      padding: 0 8px !important;
      
      .q-icon {
        font-size: 1rem !important;
        margin-right: 2px !important;
      }
    }
  }
  
  .q-ml-xs {
    margin-left: 4px !important;
  }
}
</style>