import { boot } from 'quasar/wrappers'
import axios from 'axios'

// Создаем экземпляр axios для API
const api = axios.create({
  baseURL: process.env.NODE_ENV === 'production' 
    ? 'http://localhost:8080/api' 
    : 'http://localhost:8080/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  }
})

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
    console.error('API Error:', error)
    
    // Обработка ошибок авторизации
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token')
      // Можно добавить редирект на страницу входа
    }
    
    return Promise.reject(error)
  }
)

export default boot(({ app }) => {
  // Делаем API доступным глобально
  app.config.globalProperties.$axios = axios
  app.config.globalProperties.$api = api
})

export { api }

