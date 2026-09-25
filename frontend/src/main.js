import { createApp } from 'vue'
import App from './App.vue'
import { reveal } from './directives/reveal'
import { initTheme } from './composables/useTheme'
import { initThemePreset } from './composables/useThemePreset'
import './style.css'

// 首屏前应用深浅色与主题色，避免闪烁
initTheme()
initThemePreset()

createApp(App).directive('reveal', reveal).mount('#app')
