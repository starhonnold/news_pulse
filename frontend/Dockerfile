# Multi-stage build для Quasar приложения
FROM node:20-alpine AS build-stage

# Установка рабочей директории
WORKDIR /app

# Копирование всех файлов конфигурации Quasar
COPY package*.json ./
COPY quasar.config.js ./
COPY postcss.config.js ./
COPY eslint.config.js ./
COPY .editorconfig ./
COPY .prettierrc.json ./
COPY .npmrc ./
COPY jsconfig.json ./
COPY index.html ./

# Копирование исходного кода
COPY src/ ./src/

# Установка всех зависимостей (включая dev для сборки)
RUN npm ci

# Сборка приложения для продакшена
RUN npm run build

# Production stage
FROM nginx:alpine AS production-stage

# Копирование собранного приложения
COPY --from=build-stage /app/dist/spa /usr/share/nginx/html

# Копирование конфигурации Nginx
COPY nginx.conf /etc/nginx/nginx.conf

# Открытие порта 80
EXPOSE 80

# Запуск Nginx
CMD ["nginx", "-g", "daemon off;"]
