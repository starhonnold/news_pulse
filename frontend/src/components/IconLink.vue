<template>
  <a
    :href="link"
    :target="external ? '_blank' : undefined"
    :rel="external ? 'noopener noreferrer' : undefined"
    :class="[
      'icon-link',
      {
        'icon-link--external': external,
        'icon-link--hover': hover,
        'icon-link--muted': muted,
        'icon-link--accent-border': accentBorder
      }
    ]"
    @click="handleClick"
  >
    <q-icon
      :name="icon"
      :size="iconSize"
      class="icon-link__icon"
    />
    
    <span v-if="label" class="icon-link__label">{{ label }}</span>
    
    <q-icon
      v-if="external"
      name="open_in_new"
      :size="externalIconSize"
      class="icon-link__external-icon"
    />
  </a>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

defineProps({
  icon: {
    type: String,
    required: true
  },
  label: {
    type: String,
    default: ''
  },
  link: {
    type: String,
    default: '#'
  },
  external: {
    type: Boolean,
    default: false
  },
  hover: {
    type: Boolean,
    default: true
  },
  muted: {
    type: Boolean,
    default: false
  },
  accentBorder: {
    type: Boolean,
    default: false
  },
  iconSize: {
    type: String,
    default: '20px'
  },
  externalIconSize: {
    type: String,
    default: '14px'
  }
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  emit('click', event)
}
</script>

<style lang="scss" scoped>
.icon-link {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: var(--text-primary);
  transition: all var(--transition-normal);
  border: 1px solid transparent;
  position: relative;
  overflow: hidden;
  min-height: 32px;
  
  // Базовые стили
  &__icon {
    color: var(--text-secondary);
    flex-shrink: 0;
    transition: all var(--transition-normal);
  }
  
  &__label {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    color: var(--text-primary);
    line-height: var(--line-height-normal);
    transition: all var(--transition-normal);
    white-space: nowrap;
  }
  
  &__external-icon {
    color: var(--text-quaternary);
    flex-shrink: 0;
    opacity: 0;
    transition: all var(--transition-normal);
  }
  
  // Hover эффекты
  &--hover:hover {
    background: var(--bg-tertiary);
    border-color: var(--border-accent);
    box-shadow: 0 0 6px rgba(139, 92, 246, 0.1);
    transform: translateY(-1px);
    
    .icon-link__icon {
      color: var(--accent-color);
      transform: scale(1.1);
    }
    
    .icon-link__label {
      color: var(--text-primary);
    }
    
    .icon-link__external-icon {
      opacity: 1;
      color: var(--text-tertiary);
    }
  }
  
  // Внешняя ссылка
  &--external {
    .icon-link__external-icon {
      opacity: 0.5;
    }
    
    &:hover .icon-link__external-icon {
      opacity: 1;
    }
  }
  
  // Приглушенная ссылка
  &--muted {
    opacity: 0.7;
    
    &:hover {
      opacity: 1;
    }
  }
  
  // С акцентной границей
  &--accent-border {
    border-color: var(--accent-color);
    
    &:hover {
      box-shadow: 0 0 6px rgba(245, 158, 11, 0.2);
    }
  }
  
  // Фокус для доступности
  &:focus-visible {
    outline: 2px solid var(--accent-color) !important;
    outline-offset: 2px !important;
  }
}

// Адаптивность
@media (max-width: 768px) {
  .icon-link {
    padding: var(--spacing-xs);
    min-height: 36px;
    
    &__label {
      font-size: var(--font-size-xs);
    }
    
    &__external-icon {
      font-size: 12px;
    }
  }
}

// Компактная версия для маленьких экранов
@media (max-width: 480px) {
  .icon-link {
    &__label {
      display: none;
    }
    
    &__external-icon {
      display: none;
    }
  }
}
</style>
