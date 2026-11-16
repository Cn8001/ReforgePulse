import { createApp } from 'vue'
import App from './App.vue'
import { followAccent } from './lib/theme'
import './style.css'

followAccent()
createApp(App).mount('#app')
