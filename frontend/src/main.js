import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './wailsjs/runtime/runtime.js'
const app = createApp(App)

// Global error handler
app.config.errorHandler = (err, vm, info) => {
  console.error('Vue Error:', err)
  console.error('Component:', vm)
  console.error('Info:', info)
}

createApp(App).mount('#app')