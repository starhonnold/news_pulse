import { api, createRequest } from 'boot/api'

// Сервис для работы с новостями
export const newsService = {
  // Получить список новостей с пагинацией
  getNews(params = {}) {
    return api.get('/news', { params })
  },

  // Получить новость по ID
  getNewsById(id) {
    return api.get(`/news/${id}`)
  },

  // Поиск новостей
  searchNews(query, params = {}) {
    return api.get('/news/search', { 
      params: { q: query, ...params } 
    })
  },

  // Получить последние новости
  getLatestNews(limit = 20) {
    return api.get('/news/latest', { 
      params: { limit } 
    })
  },

  // Получить трендовые новости
  getTrendingNews(limit = 20) {
    return api.get('/news/trending', { 
      params: { limit } 
    })
  }
}

// Сервис для работы с пульсами
export const pulseService = {
  // Получить пульсы пользователя
  getUserPulses() {
    return createRequest({ method: 'GET', url: '/pulses' })
  },

  // Создать новый пульс
  createPulse(pulseData) {
    return createRequest({ method: 'POST', url: '/pulses', data: pulseData })
  },

  // Обновить пульс
  updatePulse(id, pulseData) {
    return createRequest({ method: 'PUT', url: `/pulses/${id}`, data: pulseData })
  },

  // Удалить пульс
  deletePulse(id) {
    return createRequest({ method: 'DELETE', url: `/pulses/${id}` })
  },

  // Получить новости для пульса (с retry логикой)
  getPulseNews(id, params = {}) {
    return createRequest({ method: 'GET', url: `/pulses/${id}/news`, params })
  },

  // Обновить новости пульса (с retry логикой)
  refreshPulse(id) {
    return createRequest({ method: 'POST', url: `/pulses/${id}/refresh` })
  }
}

// Сервис для работы со справочниками
export const referenceService = {
  // Получить список категорий
  getCategories() {
    return createRequest({ method: 'GET', url: '/categories' })
  },

  // Получить список стран
  getCountries() {
    return createRequest({ method: 'GET', url: '/countries' })
  },

  // Получить список источников
  getSources(countryId = null) {
    const params = countryId ? { country_id: countryId } : {}
    return createRequest({ method: 'GET', url: '/sources', params })
  }
}

// Сервис для уведомлений
export const notificationService = {
  // Получить уведомления
  getNotifications(params = {}) {
    return api.get('/notifications', { params })
  },

  // Отметить уведомление как прочитанное
  markAsRead(id) {
    return api.patch(`/notifications/${id}/read`)
  },

  // Подписаться на WebSocket уведомления
  subscribeToNotifications() {
    // Используем относительный путь для WebSocket через Nginx прокси
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`
    
    return new WebSocket(wsUrl)
  }
}

// Утилита для обработки ошибок API
export const handleApiError = (error, defaultMessage = 'Произошла ошибка') => {
  if (error.response) {
    // Сервер вернул ошибку
    const message = error.response.data?.message || error.response.data?.error || defaultMessage
    return {
      message,
      status: error.response.status,
      data: error.response.data
    }
  } else if (error.request) {
    // Запрос был отправлен, но ответа не получено
    return {
      message: 'Сервер не отвечает. Проверьте подключение к интернету.',
      status: 0,
      data: null
    }
  } else {
    // Ошибка при настройке запроса
    return {
      message: error.message || defaultMessage,
      status: -1,
      data: null
    }
  }
}

// Экспорт API по умолчанию
export default api

