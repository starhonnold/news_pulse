<template>
  <button
    :class="[
      'base-button',
      `base-button--${variant}`,
      `base-button--${size}`,
      {
        'base-button--loading': loading,
        'base-button--disabled': disabled,
        'base-button--icon-only': iconOnly,
        'base-button--glass': glass,
        'base-button--muted': muted,
        'base-button--accent-border': accentBorder
      }
    ]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <!-- Иконка загрузки -->
    <q-spinner
      v-if="loading"
      :size="iconSize"
      color="currentColor"
      class="base-button__spinner"
    />
    
    <!-- Основная иконка -->
    <q-icon
      v-else-if="icon"
      :name="icon"
      :size="iconSize"
      class="base-button__icon"
    />
    
    <!-- Текст кнопки -->
    <span v-if="!iconOnly" class="base-button__text">
      <slot />
    </span>
    
    <!-- Дополнительная иконка справа -->
    <q-icon
      v-if="rightIcon && !loading"
      :name="rightIcon"
      :size="iconSize"
      class="base-button__right-icon"
    />
  </button>
</template>

<script setup>
import { defineProps, defineEmits, computed } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'primary', // primary, secondary, ghost, danger
    validator: (value) => ['primary', 'secondary', 'ghost', 'danger'].includes(value)
  },
  size: {
    type: String,
    default: 'medium', // small, medium, large
    validator: (value) => ['small', 'medium', 'large'].includes(value)
  },
  icon: {
    type: String,
    default: ''
  },
  rightIcon: {
    type: String,
    default: ''
  },
  loading: {
    type: Boolean,
    default: false
  },
  disabled: {
    type: Boolean,
    default: false
  },
  iconOnly: {
    type: Boolean,
    default: false
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

const iconSize = computed(() => {
  switch (props.size) {
    case 'small': return '16px'
    case 'large': return '24px'
    default: return '20px'
  }
})

const handleClick = (event) => {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<style lang="scss" scoped>
.base-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-xs);
  border: none;
  border-radius: var(--radius-md);
  font-family: inherit;
  font-weight: var(--font-weight-medium);
  text-decoration: none;
  cursor: pointer;
  transition: all var(--transition-normal);
  position: relative;
  overflow: hidden;
  outline: none;
  
  // Базовые стили для всех размеров
  &--small {
    padding: var(--spacing-xs) var(--spacing-sm);
    font-size: var(--font-size-sm);
    min-height: 32px;
  }
  
  &--medium {
    padding: var(--spacing-sm) var(--spacing-md);
    font-size: var(--font-size-base);
    min-height: 40px;
  }
  
  &--large {
    padding: var(--spacing-md) var(--spacing-lg);
    font-size: var(--font-size-lg);
    min-height: 48px;
  }
  
  // Иконка только
  &--icon-only {
    padding: var(--spacing-sm);
    min-width: 40px;
    
    &.base-button--small {
      padding: var(--spacing-xs);
      min-width: 32px;
    }
    
    &.base-button--large {
      padding: var(--spacing-md);
      min-width: 48px;
    }
  }
  
  // Варианты кнопок
  &--primary {
    background: var(--primary-color);
    color: var(--text-inverse);
    border: 1px solid var(--primary-color);
    
    &:hover:not(:disabled) {
      background: var(--primary-dark);
      border-color: var(--primary-dark);
      box-shadow: 0 0 12px rgba(139, 92, 246, 0.3);
      transform: scale(0.98);
    }
    
    &:active:not(:disabled) {
      transform: scale(0.96);
      box-shadow: 0 0 8px rgba(139, 92, 246, 0.2);
    }
  }
  
  &--secondary {
    background: var(--secondary-color);
    color: var(--text-inverse);
    border: 1px solid var(--secondary-color);
    
    &:hover:not(:disabled) {
      background: var(--secondary-dark);
      border-color: var(--secondary-dark);
      box-shadow: 0 0 12px rgba(6, 182, 212, 0.3);
      transform: scale(0.98);
    }
    
    &:active:not(:disabled) {
      transform: scale(0.96);
      box-shadow: 0 0 8px rgba(6, 182, 212, 0.2);
    }
  }
  
  &--ghost {
    background: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border-primary);
    
    &:hover:not(:disabled) {
      background: var(--bg-tertiary);
      border-color: var(--border-accent);
      box-shadow: 0 0 8px rgba(139, 92, 246, 0.15);
      transform: scale(0.98);
    }
    
    &:active:not(:disabled) {
      transform: scale(0.96);
      background: var(--bg-elevated);
    }
  }
  
  &--danger {
    background: var(--error-color);
    color: var(--text-inverse);
    border: 1px solid var(--error-color);
    
    &:hover:not(:disabled) {
      background: var(--error-600);
      border-color: var(--error-600);
      box-shadow: 0 0 12px rgba(239, 68, 68, 0.3);
      transform: scale(0.98);
    }
    
    &:active:not(:disabled) {
      transform: scale(0.96);
      box-shadow: 0 0 8px rgba(239, 68, 68, 0.2);
    }
  }
  
  // Состояния
  &--loading {
    cursor: not-allowed;
    opacity: 0.7;
  }
  
  &--disabled {
    cursor: not-allowed;
    opacity: 0.5;
    background: var(--bg-tertiary) !important;
    color: var(--text-disabled) !important;
    border-color: var(--border-secondary) !important;
    box-shadow: none !important;
    transform: none !important;
  }
  
  // Модификаторы
  &--glass {
    background: var(--bg-overlay) !important;
    backdrop-filter: blur(20px) !important;
    -webkit-backdrop-filter: blur(20px) !important;
    border: 1px solid rgba(255, 255, 255, 0.1) !important;
    box-shadow: var(--shadow-md) !important;
  }
  
  &--muted {
    opacity: 0.7;
    
    &:hover:not(:disabled) {
      opacity: 1;
    }
  }
  
  &--accent-border {
    border-color: var(--accent-color) !important;
    
    &:hover:not(:disabled) {
      box-shadow: 0 0 12px rgba(245, 158, 11, 0.3) !important;
    }
  }
  
  // Элементы кнопки
  &__spinner {
    animation: spin 1s linear infinite;
  }
  
  &__icon {
    flex-shrink: 0;
  }
  
  &__text {
    flex: 1;
    text-align: center;
  }
  
  &__right-icon {
    flex-shrink: 0;
  }
}

// Анимация вращения
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

// Фокус для доступности
.base-button:focus-visible {
  outline: 2px solid var(--accent-color) !important;
  outline-offset: 2px !important;
}

// Адаптивность
@media (max-width: 768px) {
  .base-button {
    &--small {
      min-height: 36px;
    }
    
    &--medium {
      min-height: 44px;
    }
    
    &--large {
      min-height: 52px;
    }
    
    &--icon-only {
      min-width: 44px;
      
      &.base-button--small {
        min-width: 36px;
      }
      
      &.base-button--large {
        min-width: 52px;
      }
    }
  }
}
</style>
