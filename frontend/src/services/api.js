import { api } from 'boot/api'

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
    return api.get('/pulses')
  },

  // Создать новый пульс
  createPulse(pulseData) {
    return api.post('/pulses', pulseData)
  },

  // Обновить пульс
  updatePulse(id, pulseData) {
    return api.put(`/pulses/${id}`, pulseData)
  },

  // Удалить пульс
  deletePulse(id) {
    return api.delete(`/pulses/${id}`)
  },

  // Получить новости для пульса
  getPulseNews(id, params = {}) {
    return api.get(`/pulses/${id}/news`, { params })
  },

  // Обновить новости пульса
  refreshPulse(id) {
    return api.post(`/pulses/${id}/refresh`)
  }
}

// Сервис для работы со справочниками
export const referenceService = {
  // Получить список категорий
  getCategories() {
    return api.get('/categories')
  },

  // Получить список стран
  getCountries() {
    return api.get('/countries')
  },

  // Получить список источников
  getSources(countryId = null) {
    const params = countryId ? { country_id: countryId } : {}
    return api.get('/sources', { params })
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
    const wsUrl = process.env.NODE_ENV === 'production' 
      ? 'ws://localhost:8080/ws' 
      : 'ws://localhost:8080/ws'
    
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

