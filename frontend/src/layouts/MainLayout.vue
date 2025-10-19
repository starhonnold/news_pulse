<template>
  <q-layout view="lHh Lpr lFf" class="minimal-layout">
    <!-- Минималистичный хедер -->
    <q-header class="minimal-header">
      <q-toolbar class="minimal-toolbar">
        <!-- Логотип (монохром) -->
        <div class="logo-container" @click="navigateToHome">
          <img 
            src="~assets/news-pulse-logo--white-color.png" 
            alt="News Pulse" 
            class="minimal-logo"
          />
        </div>

        <!-- Десктопная навигация -->
        <div v-if="$q.screen.gt.sm" class="desktop-nav">
          <q-tabs
            v-model="activeTab"
            class="desktop-tabs"
            active-color="primary"
            indicator-color="primary"
            align="left"
            no-caps
            dense
          >
            <q-tab 
              name="news" 
              label="Новости" 
              icon="newspaper"
              @click="navigateToNews"
              class="desktop-tab"
            />
            <q-tab 
              name="pulses" 
              label="Пульсы" 
              icon="analytics"
              @click="navigateToPulses"
              class="desktop-tab"
            />
          </q-tabs>
        </div>

        <q-space />

        <!-- Действия хедера -->
        <div class="header-actions">
          <!-- Уведомления -->
          <q-btn 
            flat 
            round 
            dense 
            icon="notifications" 
            class="header-icon"
            size="sm"
            aria-label="Уведомления"
          >
            <q-badge color="red" floating class="notification-badge">3</q-badge>
          </q-btn>

          <!-- Навигационное меню (всегда видимо) -->
          <q-btn 
            flat 
            round 
            dense 
            icon="menu" 
            class="header-icon"
            size="sm"
            @click="toggleMobileMenu"
            aria-label="Открыть меню"
          />
        </div>
      </q-toolbar>
    </q-header>

    <!-- Контейнер страницы с отступами и центровкой -->
    <q-page-container class="minimal-page-container">
      <div class="content-wrapper">
        <router-view />
      </div>
    </q-page-container>

    <!-- Мобильная нижняя навигация -->
    <q-footer v-if="$q.screen.lt.md" class="mobile-bottom-nav">
      <q-tabs
        v-model="activeTab"
        class="mobile-nav-tabs"
        active-color="primary"
        indicator-color="primary"
        dense
      >
        <q-tab
          name="news"
          @click="navigateToNews"
          class="mobile-nav-tab"
          tabindex="0"
        >
          <q-icon name="newspaper" size="24px" />
          <div class="mobile-nav-label">Новости</div>
        </q-tab>

        <q-tab
          name="pulses"
          @click="navigateToPulses"
          class="mobile-nav-tab"
          tabindex="0"
        >
          <q-icon name="analytics" size="24px" />
          <div class="mobile-nav-label">Пульсы</div>
        </q-tab>
      </q-tabs>
    </q-footer>

    <!-- Мобильное боковое меню -->
    <q-drawer
      v-model="mobileMenuOpen"
      side="right"
      overlay
      class="mobile-drawer"
    >
      <div class="mobile-nav">
        <div class="mobile-nav-header">
          <h4>Навигация</h4>
          <q-btn 
            flat 
            round 
            dense 
            icon="close" 
            @click="mobileMenuOpen = false"
            class="close-btn"
            size="sm"
            aria-label="Закрыть меню"
          />
        </div>
        
        <q-list class="mobile-nav-list">
          <q-item 
            clickable 
            @click="navigateToNews"
            class="mobile-nav-item"
            tabindex="0"
          >
            <q-item-section avatar>
              <q-icon name="newspaper" size="md" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Новости</q-item-label>
              <q-item-label caption>Просмотр новостей</q-item-label>
            </q-item-section>
          </q-item>
          
          <q-item 
            clickable 
            @click="navigateToPulses"
            class="mobile-nav-item"
            tabindex="0"
          >
            <q-item-section avatar>
              <q-icon name="analytics" size="md" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Пульсы</q-item-label>
              <q-item-label caption>Управление пульсами</q-item-label>
            </q-item-section>
          </q-item>

          <q-separator class="mobile-nav-separator" />

          <q-item 
            clickable 
            @click="navigateToProfile"
            class="mobile-nav-item"
            tabindex="0"
          >
            <q-item-section avatar>
              <q-icon name="person" size="md" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Профиль</q-item-label>
              <q-item-label caption>Настройки аккаунта</q-item-label>
            </q-item-section>
          </q-item>

          <q-item 
            clickable 
            @click="navigateToSettings"
            class="mobile-nav-item"
            tabindex="0"
          >
            <q-item-section avatar>
              <q-icon name="settings" size="md" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Настройки</q-item-label>
              <q-item-label caption>Персонализация</q-item-label>
            </q-item-section>
          </q-item>

          <q-separator class="mobile-nav-separator" />

          <q-item 
            clickable 
            @click="toggleAuth"
            class="mobile-nav-item"
            tabindex="0"
          >
            <q-item-section avatar>
              <q-icon name="login" size="md" />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ isAuthenticated ? 'Выйти' : 'Войти' }}</q-item-label>
              <q-item-label caption>{{ isAuthenticated ? 'Выйти из аккаунта' : 'Войти в аккаунт' }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </div>
    </q-drawer>
  </q-layout>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const mobileMenuOpen = ref(false)
const activeTab = ref('news')
const isAuthenticated = ref(false) // Временная переменная, в реальном приложении должна браться из store

// Определяем активную вкладку на основе текущего маршрута
onMounted(() => {
  updateActiveTab()
})

// Отслеживаем изменения маршрута
watch(() => route.path, () => {
  updateActiveTab()
})

const updateActiveTab = () => {
  if (route.path === '/' || route.path === '/news') {
    activeTab.value = 'news'
  } else if (route.path === '/pulses' || route.path.startsWith('/pulse/')) {
    activeTab.value = 'pulses'
  } else {
    activeTab.value = 'news'
  }
}

const toggleMobileMenu = () => {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

const navigateToHome = () => {
  router.push('/')
  activeTab.value = 'news'
  mobileMenuOpen.value = false
}

const navigateToNews = () => {
  router.push('/')
  activeTab.value = 'news'
  mobileMenuOpen.value = false
}

const navigateToPulses = () => {
  router.push('/pulses')
  activeTab.value = 'pulses'
  mobileMenuOpen.value = false
}

const navigateToProfile = () => {
  // router.push('/profile')
  mobileMenuOpen.value = false
}

const navigateToSettings = () => {
  // router.push('/settings')
  mobileMenuOpen.value = false
}

const toggleAuth = () => {
  if (isAuthenticated.value) {
    // Логика выхода
    isAuthenticated.value = false
    console.log('Пользователь вышел из системы')
  } else {
    // Логика входа
    isAuthenticated.value = true
    console.log('Пользователь вошел в систему')
  }
  mobileMenuOpen.value = false
}
</script>

<style lang="scss" scoped>
// === МИНИМАЛИСТИЧНЫЙ LAYOUT ===
.minimal-layout {
  background: var(--bg-main) !important;
}

// === МИНИМАЛИСТИЧНЫЙ ХЕДЕР ===
.minimal-header {
  background: var(--bg-card) !important;
  border-bottom: 1px solid var(--border-primary) !important;
  box-shadow: none !important;
}

.minimal-toolbar {
  min-height: 80px !important;
  padding: 0 var(--spacing-lg) !important;
}

// === ЛОГОТИП ===
.logo-container {
  display: flex;
  align-items: center;
  cursor: pointer;
  transition: all 0.3s ease;
}

.logo-container:hover {
  transform: scale(1.05);
}

.minimal-logo {
  height: 256px;
  width: auto;
  filter: brightness(0) invert(1); // Монохромный белый
  transition: all 0.3s ease;
  animation: logoPulse 2s ease-in-out infinite;
}

.minimal-logo:hover {
  filter: brightness(0) invert(1) drop-shadow(0 0 8px rgba(139, 92, 246, 0.6));
  animation: logoGlow 0.6s ease-in-out;
}

// Анимации логотипа
@keyframes logoPulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.8;
    transform: scale(1.02);
  }
}

@keyframes logoGlow {
  0% {
    filter: brightness(0) invert(1);
    transform: scale(1);
  }
  50% {
    filter: brightness(0) invert(1) drop-shadow(0 0 12px rgba(139, 92, 246, 0.8));
    transform: scale(1.1);
  }
  100% {
    filter: brightness(0) invert(1) drop-shadow(0 0 8px rgba(139, 92, 246, 0.6));
    transform: scale(1.05);
  }
}

// === ДЕСКТОПНАЯ НАВИГАЦИЯ ===
.desktop-nav {
  margin-left: var(--spacing-lg);
}

.desktop-tabs {
  .desktop-tab {
    color: var(--text-secondary) !important;
    font-weight: var(--font-weight-medium) !important;
    text-transform: none !important;
    font-size: var(--font-size-sm) !important;
    padding: var(--spacing-sm) var(--spacing-md) !important;
    border-radius: var(--radius-md) !important;
    margin: 0 var(--spacing-xs) !important;
    transition: all var(--transition-normal) !important;
    min-height: 36px !important;
    
    .q-icon {
      font-size: 1rem !important;
      margin-right: var(--spacing-xs) !important;
      color: var(--text-secondary) !important;
    }
    
    &:hover {
      color: var(--text-primary) !important;
      background: var(--bg-tertiary) !important;
      box-shadow: 0 0 8px rgba(139, 92, 246, 0.1) !important; // Легкий glow
      
      .q-icon {
        color: var(--text-primary) !important;
      }
    }
    
    &.q-tab--active {
      color: var(--primary-color) !important;
      font-weight: var(--font-weight-semibold) !important;
      background: var(--bg-tertiary) !important;
      
      .q-icon {
        color: var(--primary-color) !important;
      }
    }
  }
  
  .q-tab__indicator {
    background: var(--primary-color) !important;
    height: 2px !important;
    border-radius: var(--radius-full) !important;
  }
}

// === ДЕЙСТВИЯ ХЕДЕРА ===
.header-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.header-icon {
  color: var(--text-primary) !important;
  position: relative !important;
  overflow: visible !important;
  
  .q-icon {
    font-size: 1.1rem !important;
    color: var(--text-primary) !important;
  }
  
  &:hover {
    background: var(--bg-tertiary) !important;
    box-shadow: 0 0 8px rgba(139, 92, 246, 0.1) !important; // Легкий glow
  }
  
  // Специальные стили для кнопки уведомлений
  &.q-btn--round {
    overflow: visible !important;
  }
}

// === БЕЙДЖ УВЕДОМЛЕНИЙ ===
.notification-badge {
  font-size: 0.7rem !important;
  font-weight: 700 !important;
  padding: 2px 6px !important;
  min-width: 18px !important;
  height: 18px !important;
  border-radius: 9px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  line-height: 1 !important;
  top: -2px !important;
  right: -2px !important;
  z-index: 1000 !important;
}

// === МЕНЮ ПОЛЬЗОВАТЕЛЯ ===
.user-menu {
  border-radius: var(--radius-md) !important;
  box-shadow: var(--shadow-sm) !important;
  border: 1px solid var(--border-primary) !important;
}

.menu-list {
  padding: var(--spacing-xs) !important;
  background: var(--bg-card) !important;
}

.menu-item {
  border-radius: var(--radius-sm) !important;
  margin: 1px 0 !important;
  min-height: 44px !important; // Минимальная тап-зона
  
  &:hover {
    background: var(--bg-tertiary) !important;
    box-shadow: 0 0 6px rgba(139, 92, 246, 0.08) !important; // Легкий glow
  }
  
  &:focus {
    outline: 2px solid var(--accent-color) !important;
    outline-offset: 2px !important;
  }
}

.menu-separator {
  margin: var(--spacing-xs) 0 !important;
  background: var(--border-primary) !important;
}

// === КОНТЕЙНЕР СТРАНИЦЫ ===
.minimal-page-container {
  background: var(--bg-main) !important;
  min-height: calc(100vh - 48px) !important;
}

.content-wrapper {
  max-width: none !important;
  margin: 0 !important;
  padding: var(--spacing-lg);
  min-height: calc(100vh - 48px);
}

// === МОБИЛЬНАЯ НИЖНЯЯ НАВИГАЦИЯ ===
.mobile-bottom-nav {
  background: var(--bg-card) !important;
  border-top: 1px solid var(--border-primary) !important;
  box-shadow: none !important;
}

.mobile-nav-tabs {
  width: 100%;
  height: 64px;
  background: transparent !important;
  
  .q-tabs__content {
    justify-content: space-evenly;
  }
}

.mobile-nav-tab {
  flex-direction: column !important;
  padding: var(--spacing-sm) var(--spacing-md) !important;
  min-height: 64px !important;
  transition: all var(--transition-normal) !important;
  
  .q-tab__content {
    flex-direction: column !important;
    gap: var(--spacing-xs) !important;
  }
  
  .q-icon {
    margin: 0 !important;
    font-size: 24px !important;
    transition: all var(--transition-normal) !important;
  }
  
  &.q-tab--active {
    background: var(--bg-tertiary) !important;
    border-radius: var(--radius-md) !important;
    
    .q-icon {
      color: var(--primary-color) !important;
      transform: scale(1.05) !important;
    }
    
    .mobile-nav-label {
      color: var(--primary-color) !important;
      font-weight: var(--font-weight-semibold) !important;
    }
  }
  
  &:not(.q-tab--active) {
    .q-icon {
      color: var(--text-tertiary) !important;
    }
    
    .mobile-nav-label {
      color: var(--text-tertiary) !important;
    }
  }
  
  &:focus {
    outline: 2px solid var(--accent-color) !important;
    outline-offset: 2px !important;
  }
}

.mobile-nav-label {
  font-size: var(--font-size-xs) !important;
  line-height: 1.2 !important;
  text-transform: none !important;
  margin-top: 2px !important;
  transition: all var(--transition-normal) !important;
}

// === МОБИЛЬНОЕ БОКОВОЕ МЕНЮ ===
.mobile-drawer {
  background: var(--bg-card) !important;
  border-left: 1px solid var(--border-primary) !important;
}

.mobile-nav {
  padding: var(--spacing-lg);
  height: 100%;
}

.mobile-nav-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-lg);
  padding-bottom: var(--spacing-md);
  border-bottom: 1px solid var(--border-primary);
  
  h4 {
    margin: 0;
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-medium);
    color: var(--text-primary);
  }
}

.close-btn {
  color: var(--text-secondary) !important;
  
  &:hover {
    background: var(--bg-tertiary) !important;
  }
  
  &:focus {
    outline: 2px solid var(--accent-color) !important;
    outline-offset: 2px !important;
  }
}

.mobile-nav-list {
  padding: 0;
}

.mobile-nav-item {
  border-radius: var(--radius-md) !important;
  margin-bottom: var(--spacing-sm) !important;
  min-height: 56px !important; // Большие кликабельные элементы
  padding: var(--spacing-md) !important;
  
  &:hover {
    background: var(--bg-tertiary) !important;
    box-shadow: 0 0 8px rgba(139, 92, 246, 0.1) !important; // Легкий glow
  }
  
  &:focus {
    outline: 2px solid var(--accent-color) !important;
    outline-offset: 2px !important;
  }
  
  .q-icon {
    color: var(--text-secondary) !important;
  }
  
  &:hover .q-icon {
    color: var(--accent-color) !important;
  }
  
  .q-item-label {
    font-weight: var(--font-weight-medium) !important;
    color: var(--text-primary) !important;
  }
  
  .q-item-label[data-caption] {
    font-size: var(--font-size-sm) !important;
    color: var(--text-tertiary) !important;
  }
}

.mobile-nav-separator {
  margin: var(--spacing-md) 0 !important;
  background: var(--border-primary) !important;
}

// === АДАПТИВНОСТЬ ===
@media (max-width: 1024px) {
  .minimal-toolbar {
    padding: 0 var(--spacing-md) !important;
  }
  
  .content-wrapper {
    padding: var(--spacing-md);
  }
  
  .desktop-nav {
    margin-left: var(--spacing-md);
  }
  
  .minimal-logo {
    height: 240px;
  }
}

// Десктопная версия - еще больше логотип
@media (min-width: 1025px) {
  .minimal-logo {
    height: 48px;
  }
  
  .logo-container:hover {
    transform: scale(1.08);
  }
}

@media (max-width: 768px) {
  .minimal-toolbar {
    padding: 0 var(--spacing-sm) !important;
    min-height: 44px !important;
  }
  
  .minimal-logo {
    height: 120px;
  }
  
  .content-wrapper {
    padding: var(--spacing-sm);
    padding-bottom: 80px !important; // Отступ для нижней навигации
  }
  
  .header-icon {
    .q-icon {
      font-size: 1rem !important;
    }
  }
}

@media (max-width: 480px) {
  .minimal-toolbar {
    padding: 0 var(--spacing-xs) !important;
    min-height: 40px !important;
  }
  
  .minimal-logo {
    height: 60px;
  }
  
  .content-wrapper {
    padding: var(--spacing-xs);
    padding-bottom: 80px !important; // Отступ для нижней навигации
  }
  
  .header-icon {
    .q-icon {
      font-size: 0.9rem !important;
    }
  }
  
  .mobile-nav-tab {
    padding: var(--spacing-xs) var(--spacing-sm) !important;
    
    .q-icon {
      font-size: 20px !important;
    }
  }
  
  .mobile-nav-label {
    font-size: 0.7rem !important;
  }
}

// === УТИЛИТЫ ===
.gap-xs {
  gap: var(--spacing-xs);
}

.gap-sm {
  gap: var(--spacing-sm);
}

.gap-md {
  gap: var(--spacing-md);
}

// === ФОКУС ДЛЯ ДОСТУПНОСТИ ===
.header-icon:focus-visible,
.desktop-tab:focus-visible,
.mobile-nav-tab:focus-visible,
.menu-item:focus-visible,
.mobile-nav-item:focus-visible,
.close-btn:focus-visible {
  outline: 2px solid var(--accent-color) !important;
  outline-offset: 2px !important;
}

// === КЛАВИАТУРНАЯ НАВИГАЦИЯ ===
.q-tab[tabindex="0"]:focus,
.q-item[tabindex="0"]:focus {
  outline: 2px solid var(--accent-color) !important;
  outline-offset: 2px !important;
}
</style>