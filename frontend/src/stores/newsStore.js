import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useNewsStore = defineStore('news', () => {
  // Состояние
  const allNews = ref([])
  const categories = ref([])
  const countries = ref([])
  const sources = ref([])
  const loading = ref(false)
  const error = ref(null)

  // Действия
  const fetchAllNews = async () => {
    try {
      loading.value = true
      error.value = null
      
      const response = await api.get('/api/news', {
        params: {
          page_size: 1000, // Загружаем много новостей для фильтрации
          sort_by: 'published_at',
          sort_order: 'desc'
        }
      })
      
      allNews.value = response.data.data || []
    } catch (err) {
      error.value = err.message
      console.error('Ошибка загрузки новостей:', err)
    } finally {
      loading.value = false
    }
  }

  const fetchCategories = async () => {
    try {
      const response = await api.get('/api/categories')
      categories.value = response.data.data || []
      return categories.value
    } catch (err) {
      console.error('Ошибка загрузки категорий:', err)
      return []
    }
  }

  const fetchCountries = async () => {
    try {
      const response = await api.get('/api/countries')
      countries.value = response.data.data || []
      return countries.value
    } catch (err) {
      console.error('Ошибка загрузки стран:', err)
      return []
    }
  }

  const fetchSources = async () => {
    try {
      const response = await api.get('/api/sources')
      sources.value = response.data.data || []
      return sources.value
    } catch (err) {
      console.error('Ошибка загрузки источников:', err)
      return []
    }
  }

  const searchNews = async (query, filters = {}) => {
    try {
      loading.value = true
      error.value = null
      
      const params = {
        q: query,
        ...filters,
        page_size: 1000
      }
      
      const response = await api.get('/api/news/search', { params })
      allNews.value = response.data.data || []
    } catch (err) {
      error.value = err.message
      console.error('Ошибка поиска новостей:', err)
    } finally {
      loading.value = false
    }
  }

  const getNewsById = (id) => {
    return allNews.value.find(news => news.id === id)
  }

  const getNewsByCategory = (categoryId) => {
    return allNews.value.filter(news => news.category_id === categoryId)
  }

  const getNewsByCountry = (countryId) => {
    return allNews.value.filter(news => news.country_id === countryId)
  }

  const getNewsBySource = (sourceId) => {
    return allNews.value.filter(news => news.source_id === sourceId)
  }

  // Очистка состояния
  const clearNews = () => {
    allNews.value = []
    categories.value = []
    countries.value = []
    sources.value = []
    error.value = null
  }

  return {
    // Состояние
    allNews,
    categories,
    countries,
    sources,
    loading,
    error,
    
    // Действия
    fetchAllNews,
    fetchCategories,
    fetchCountries,
    fetchSources,
    searchNews,
    getNewsById,
    getNewsByCategory,
    getNewsByCountry,
    getNewsBySource,
    clearNews
  }
})
