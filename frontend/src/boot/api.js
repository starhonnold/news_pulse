import { boot } from 'quasar/wrappers'
import axios from 'axios'

// Конфигурация retry
const RETRY_CONFIG = {
  retries: 3,
  retryDelay: (retryCount) => {
    // Exponential backoff: 1s, 2s, 4s
    return Math.pow(2, retryCount) * 1000
  },
  retryCondition: (error) => {
    // Повторяем для сетевых ошибок и 5xx статусов
    return (
      !error.response || // Сетевые ошибки
      (error.response.status >= 500 && error.response.status < 600) || // 5xx ошибки
      error.code === 'ECONNABORTED' || // Таймауты
      error.code === 'NETWORK_ERROR' // Другие сетевые ошибки
    )
  }
}

// Создаем экземпляр axios для API
const api = axios.create({
  baseURL: '/api', // Используем относительный путь для проксирования через Nginx
  timeout: 30000, // Увеличиваем таймаут до 30 секунд
  headers: {
    'Content-Type': 'application/json',
  }
})

// Map для хранения активных запросов (для отмены)
const activeRequests = new Map()

// Функция для retry логики
const retryRequest = async (config, retryCount = 0) => {
  try {
    return await api(config)
  } catch (error) {
    if (retryCount < RETRY_CONFIG.retries && RETRY_CONFIG.retryCondition(error)) {
      const delay = RETRY_CONFIG.retryDelay(retryCount)
      console.warn(`Retry ${retryCount + 1}/${RETRY_CONFIG.retries} after ${delay}ms for ${config.url}`, error.message)
      
      await new Promise(resolve => setTimeout(resolve, delay))
      return retryRequest(config, retryCount + 1)
    }
    throw error
  }
}

// Функция для создания запроса с AbortController
const createRequest = (config) => {
  const requestKey = `${config.method || 'GET'}_${config.url}`
  
  // Отменяем предыдущий запрос с тем же ключом
  if (activeRequests.has(requestKey)) {
    activeRequests.get(requestKey).abort()
  }
  
  // Создаем новый AbortController
  const controller = new AbortController()
  activeRequests.set(requestKey, controller)
  
  // Добавляем signal в конфиг
  config.signal = controller.signal
  
  // Очищаем из map после завершения запроса
  const cleanup = () => {
    activeRequests.delete(requestKey)
  }
  
  return retryRequest(config).finally(cleanup)
}

// Интерсептор для запросов
api.interceptors.request.use(
  (config) => {
    // Добавляем токен авторизации если есть
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Интерсептор для ответов
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    // Не логируем ошибки отмены запросов
    if (error.name === 'AbortError' || error.code === 'ERR_CANCELED') {
      return Promise.reject(error)
    }
    
    console.error('API Error:', error)
    
    // Обработка ошибок авторизации
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token')
      // Можно добавить редирект на страницу входа
    }
    
    // Обработка таймаутов
    if (error.code === 'ECONNABORTED') {
      console.warn('Request timeout:', error.config?.url)
    }
    
    return Promise.reject(error)
  }
)

export default boot(({ app }) => {
  // Делаем API доступным глобально
  app.config.globalProperties.$axios = axios
  app.config.globalProperties.$api = api
  app.config.globalProperties.$createRequest = createRequest
})

export { api, createRequest, RETRY_CONFIG }

