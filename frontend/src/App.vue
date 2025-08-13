<template>
  <div id="app">
    <!-- Header -->
    <header class="app-header">
      <div class="header-content">
        <div class="logo-section">
          <div class="logo-icon">🎮</div>
          <h1>Game Save Backup Manager</h1>
        </div>
        <div class="header-actions">
          <button @click="scanGames" :disabled="scanning" class="btn btn-primary">
            <span v-if="scanning" class="loading-spinner"></span>
            {{ scanning ? 'Escaneando...' : 'Escanear Juegos' }}
          </button>
          <button @click="showSettings = true" class="btn btn-secondary">
            ⚙️ Configuración
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <!-- Stats Bar -->
      <div class="stats-bar" v-if="games.length > 0">
        <div class="stat-card">
          <h3>{{ games.length }}</h3>
          <p>Juegos Detectados</p>
        </div>
        <div class="stat-card">
          <h3>{{ totalBackups }}</h3>
          <p>Backups Creados</p>
        </div>
        <div class="stat-card">
          <h3>{{ formatSize(totalSize) }}</h3>
          <p>Tamaño Total</p>
        </div>
      </div>

      <!-- Games Section -->
      <div class="games-section">
        <div class="section-header">
          <h2>Juegos Detectados</h2>
          <button @click="showAddGame = true" class="btn btn-success">
            ➕ Agregar Juego
          </button>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="loading-state">
          <div class="loading-spinner large"></div>
          <p>Cargando juegos...</p>
        </div>

        <!-- Empty State -->
        <div v-else-if="games.length === 0" class="empty-state">
          <div class="empty-icon">🎮</div>
          <h3>No se encontraron juegos</h3>
          <p>Haz clic en "Escanear Juegos" para buscar automáticamente o agrega uno manualmente</p>
        </div>

        <!-- Games Grid -->
        <div v-else class="games-grid">
          <div 
            v-for="game in games" 
            :key="game.id" 
            class="game-card"
            :class="{ 'backing-up': backingUp.includes(game.id) }"
          >
            <div class="game-header">
              <h3>{{ game.name }}</h3>
              <div class="game-platform">{{ game.platform }}</div>
            </div>

            <div class="game-info">
              <div class="info-row">
                <span class="label">Archivos:</span>
                <span class="value">{{ game.file_count || 0 }}</span>
              </div>
              <div class="info-row">
                <span class="label">Tamaño:</span>
                <span class="value">{{ formatSize(game.total_size || 0) }}</span>
              </div>
              <div class="info-row">
                <span class="label">Último backup:</span>
                <span class="value">{{ formatDate(game.last_backup) }}</span>
              </div>
            </div>

            <div class="game-paths">
              <details>
                <summary>Rutas de guardado ({{ game.save_paths?.length || 0 }})</summary>
                <ul>
                  <li v-for="path in game.save_paths" :key="path">{{ path }}</li>
                </ul>
              </details>
            </div>

            <div class="game-actions">
              <button 
                @click="createBackup(game.id)" 
                :disabled="backingUp.includes(game.id)"
                class="btn btn-primary"
              >
                <span v-if="backingUp.includes(game.id)" class="loading-spinner"></span>
                {{ backingUp.includes(game.id) ? 'Creando...' : '💾 Backup' }}
              </button>
              <button @click="viewGameDetails(game)" class="btn btn-secondary">
                📋 Detalles
              </button>
              <button @click="removeGame(game.id)" class="btn btn-danger">
                🗑️ Eliminar
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Toast Notifications -->
    <div class="toast-container">
      <div 
        v-for="toast in toasts" 
        :key="toast.id" 
        class="toast"
        :class="toast.type"
      >
        <span>{{ toast.message }}</span>
        <button @click="removeToast(toast.id)" class="toast-close">✕</button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
// Comentar temporalmente las importaciones de Wails
// import { ScanGames, GetGameList, CreateBackup, AddCustomGame, GetConfig, UpdateConfig, RemoveGame } from '../wailsjs/go/main/App'

export default {
  name: 'App',
  setup() {
    // Reactive data
    const games = ref([])
    const loading = ref(false)
    const scanning = ref(false)
    const backingUp = ref([])
    const showAddGame = ref(false)
    const showSettings = ref(false)
    const selectedGame = ref(null)
    const toasts = ref([])
    const config = ref({
      backup_dir: './game_backups',
      max_backups: 10,
      compression_enabled: true,
      auto_backup: false
    })

    const newGame = ref({
      name: '',
      path: '',
      patterns: '*.sav,*.save,*.dat'
    })

    // Computed properties
    const totalBackups = computed(() => {
      return games.value.filter(g => g.last_backup && g.last_backup !== '0001-01-01T00:00:00Z').length
    })

    const totalSize = computed(() => {
      return games.value.reduce((sum, game) => sum + (game.total_size || 0), 0)
    })

    // Métodos temporales (placeholder)
    const loadGames = async () => {
      loading.value = true
      // Simulamos algunos juegos de prueba
      games.value = [
        {
          id: 'test-game-1',
          name: 'Juego de Prueba 1',
          platform: 'steam',
          file_count: 5,
          total_size: 1024000,
          last_backup: new Date().toISOString(),
          save_paths: ['/home/user/.steam/game1']
        }
      ]
      loading.value = false
    }

    const scanGames = async () => {
      scanning.value = true
      showToast('Función de escaneo temporal - esperando bindings de Wails', 'info')
      setTimeout(() => {
        scanning.value = false
      }, 2000)
    }

    const createBackup = async (gameId) => {
      backingUp.value.push(gameId)
      showToast('Función de backup temporal - esperando bindings de Wails', 'info')
      setTimeout(() => {
        backingUp.value = backingUp.value.filter(id => id !== gameId)
      }, 2000)
    }

    const addCustomGame = async () => {
      showToast('Función agregar juego temporal - esperando bindings de Wails', 'info')
      showAddGame.value = false
    }

    const removeGame = async (gameId) => {
      showToast('Función eliminar juego temporal - esperando bindings de Wails', 'info')
    }

    const saveSettings = async () => {
      showToast('Función guardar configuración temporal - esperando bindings de Wails', 'info')
      showSettings.value = false
    }

    const viewGameDetails = (game) => {
      selectedGame.value = game
    }

    const showToast = (message, type = 'info') => {
      const toast = {
        id: Date.now(),
        message,
        type
      }
      toasts.value.push(toast)
      setTimeout(() => removeToast(toast.id), 5000)
    }

    const removeToast = (id) => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    }

    const formatSize = (bytes) => {
      if (!bytes) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }

    const formatDate = (dateStr) => {
      if (!dateStr || dateStr === '0001-01-01T00:00:00Z') {
        return 'Nunca'
      }
      return new Date(dateStr).toLocaleString()
    }

    // Lifecycle
    onMounted(() => {
      loadGames()
    })

    return {
      games,
      loading,
      scanning,
      backingUp,
      showAddGame,
      showSettings,
      selectedGame,
      toasts,
      config,
      newGame,
      totalBackups,
      totalSize,
      scanGames,
      createBackup,
      addCustomGame,
      removeGame,
      saveSettings,
      viewGameDetails,
      showToast,
      removeToast,
      formatSize,
      formatDate
    }
  }
}
</script>