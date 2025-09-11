<template>
  <q-layout view="lHh Lpr lFf" class="bg-grey-1">
    <q-header class="dark-blue-header">
      <q-toolbar>

        <q-toolbar-title class="flex items-center">
          <!-- Вкладки навигации -->
          <q-tabs
            v-model="activeTab"
            class="text-white"
            active-color="white"
            indicator-color="white"
            align="left"
            no-caps
          >
            <q-tab 
              name="pulses" 
              label="Пульсы" 
              icon="dashboard"
              @click="navigateToPulses"
            />
            <q-tab 
              name="news" 
              label="Новости" 
              icon="article"
              @click="navigateToNews"
            />
          </q-tabs>
        </q-toolbar-title>

        <q-space />

        <!-- Уведомления -->
        <q-btn flat round dense icon="notifications" class="modern-btn">
          <q-badge color="red" floating class="neon-glow">3</q-badge>
        </q-btn>

        <!-- Меню пользователя -->
        <q-btn flat round dense icon="account_circle" class="q-ml-sm modern-btn">
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
</style>