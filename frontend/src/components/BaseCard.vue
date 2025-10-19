<template>
  <div 
    :class="[
      'base-card',
      {
        'base-card--clickable': clickable,
        'base-card--hover': hover,
        'base-card--glass': glass,
        'base-card--muted': muted,
        'base-card--accent-border': accentBorder
      }
    ]"
    @click="handleClick"
  >
    <slot />
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  clickable: {
    type: Boolean,
    default: false
  },
  hover: {
    type: Boolean,
    default: true
  },
  glass: {
    type: Boolean,
    default: false
  },
  muted: {
    type: Boolean,
    default: false
  },
  accentBorder: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  if (props.clickable) {
    emit('click', event)
  }
}
</script>

<style lang="scss" scoped>
.base-card {
  background: var(--bg-elevated) !important;
  border-radius: var(--radius-lg) !important;
  box-shadow: var(--shadow-sm) !important;
  border: 1px solid var(--border-primary) !important;
  transition: all var(--transition-normal) !important;
  overflow: hidden;
  position: relative;
  
  // Базовые стили для всех карточек
  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: var(--gradient-primary);
    opacity: 0;
    transition: opacity var(--transition-normal);
  }
}

// Кликабельная карточка
.base-card--clickable {
  cursor: pointer;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-md) !important;
    border-color: var(--border-accent) !important;
    
    &::before {
      opacity: 1;
    }
  }
  
  &:active {
    transform: translateY(0);
    box-shadow: var(--shadow-sm) !important;
  }
}

// Карточка с hover эффектами
.base-card--hover {
  &:hover {
    transform: translateY(-1px);
    box-shadow: var(--shadow-md) !important;
    border-color: var(--border-accent) !important;
    
    &::before {
      opacity: 0.5;
    }
  }
}

// Стеклянный эффект
.base-card--glass {
  background: var(--bg-overlay) !important;
  backdrop-filter: blur(20px) !important;
  -webkit-backdrop-filter: blur(20px) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  box-shadow: var(--shadow-md) !important;
}

// Приглушенная карточка
.base-card--muted {
  background: var(--bg-tertiary) !important;
  border-color: var(--border-secondary) !important;
  opacity: 0.8;
  
  &:hover {
    opacity: 1;
  }
}

// Карточка с акцентной границей
.base-card--accent-border {
  border-color: var(--accent-color) !important;
  
  &::before {
    background: var(--accent-color);
    opacity: 1;
  }
}

// Адаптивность
@media (max-width: 768px) {
  .base-card {
    border-radius: var(--radius-md) !important;
    
    &--clickable:hover {
      transform: translateY(-1px);
    }
    
    &--hover:hover {
      transform: none;
    }
  }
}

// Фокус для доступности
.base-card--clickable:focus-visible {
  outline: 2px solid var(--accent-color) !important;
  outline-offset: 2px !important;
}
</style>
