<template>
  <router-view />
</template>

<script setup>
import { Dark } from 'quasar'

// Включаем светлую тему
Dark.set(false)
</script>

<style>
@import './css/lato-fonts.css';
</style>
