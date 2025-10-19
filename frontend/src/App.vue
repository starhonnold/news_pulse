<template>
  <router-view />
</template>

<script setup>
import { Dark } from 'quasar'

// Включаем темную тему по умолчанию
Dark.set(true)
</script>

<style>
@import './css/lato-fonts.css';
@import './css/global.scss';
</style>
