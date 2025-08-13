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

      <!-- Games Grid -->
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

    <!-- Add Game Modal -->
    <div v-if="showAddGame" class="modal-overlay" @click="showAddGame = false">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h2>Agregar Juego Personalizado</h2>
          <button @click="showAddGame = false" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <form @submit.prevent="addCustomGame">
            <div class="form-group">
              <label>Nombre del Juego</label>
              <input v-model="newGame.name" type="text" required class="form-input" />
            </div>
            <div class="form-group">
              <label>Ruta de Guardado</label>
              <input v-model="newGame.path" type="text" required class="form-input" />
              <small>Ejemplo: %USERPROFILE%/Documents/MiJuego/saves</small>
            </div>
            <div class="form-group">
              <label>Patrones de Archivos</label>
              <input v-model="newGame.patterns" type="text" class="form-input" />
              <small>Ejemplo: *.sav,*.save,*.dat (separados por comas)</small>
            </div>
            <div class="form-actions">
              <button type="button" @click="showAddGame = false" class="btn btn-secondary">
                Cancelar
              </button>
              <button type="submit" class="btn btn-primary">
                Agregar Juego
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Settings Modal -->
    <div v-if="showSettings" class="modal-overlay" @click="showSettings = false">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h2>Configuración</h2>
          <button @click="showSettings = false" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <form @submit.prevent="saveSettings">
            <div class="form-group">
              <label>Directorio de Backups</label>
              <input v-model="config.backup_dir" type="text" class="form-input" />
            </div>
            <div class="form-group">
              <label>Máximo de Backups por Juego</label>
              <input v-model="config.max_backups" type="number" min="1" max="50" class="form-input" />
            </div>
            <div class="form-group">
              <label class="checkbox-label">
                <input v-model="config.compression_enabled" type="checkbox" />
                Habilitar Compresión ZIP
              </label>
            </div>
            <div class="form-group">
              <label class="checkbox-label">
                <input v-model="config.auto_backup" type="checkbox" />
                Backup Automático
              </label>
            </div>
            <div class="form-actions">
              <button type="button" @click="showSettings = false" class="btn btn-secondary">
                Cancelar
              </button>
              <button type="submit" class="btn btn-primary">
                Guardar Configuración
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Game Details Modal -->
    <div v-if="selectedGame" class="modal-overlay" @click="selectedGame = null">
      <div class="modal large" @click.stop>
        <div class="modal-header">
          <h2>{{ selectedGame.name }} - Detalles</h2>
          <button @click="selectedGame = null" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <div class="details-grid">
            <div class="detail-section">
              <h3>Información General</h3>
              <div class="detail-item">
                <strong>ID:</strong> {{ selectedGame.id }}
              </div>
              <div class="detail-item">
                <strong>Plataforma:</strong> {{ selectedGame.platform }}
              </div>
              <div class="detail-item">
                <strong>Archivos:</strong> {{ selectedGame.file_count }}
              </div>
              <div class="detail-item">
                <strong>Tamaño Total:</strong> {{ formatSize(selectedGame.total_size) }}
              </div>
            </div>
            <div class="detail-section">
              <h3>Rutas de Guardado</h3>
              <ul class="paths-list">
                <li v-for="path in selectedGame.save_paths" :key="path">
                  {{ path }}
                </li>
              </ul>
            </div>
            <div class="detail-section">
              <h3>Patrones de Archivos</h3>
              <div class="patterns-list">
                <span v-for="pattern in selectedGame.patterns" :key="pattern" class="pattern-tag">
                  {{ pattern }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

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
import { ScanGames, GetGameList, CreateBackup, AddCustomGame, GetConfig, UpdateConfig, RemoveGame } from '../wailsjs/go/main/App'

export default {
  name: 'App',
  setup() {
    // Reactive data
    const games = ref([])
    const loading = ref(true)
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

    // Methods
    const loadGames = async () => {
      try {
        loading.value = true
        const gameList = await GetGameList()
        games.value = gameList || []
      } catch (error) {
        showToast('Error cargando juegos: ' + error, 'error')
      } finally {
        loading.value = false
      }
    }

    const loadConfig = async () => {
      try {
        const cfg = await GetConfig()
        config.value = cfg
      } catch (error) {
        console.error('Error cargando configuración:', error)
      }
    }

    const scanGames = async () => {
      try {
        scanning.value = true
        const result = await ScanGames()
        await loadGames()
        showToast(`Escaneo completado: ${result.total_games} juegos, ${result.new_games?.length || 0} nuevos`, 'success')
      } catch (error) {
        showToast('Error durante el escaneo: ' + error, 'error')
      } finally {
        scanning.value = false
      }
    }

    const createBackup = async (gameId) => {
      try {
        backingUp.value.push(gameId)
        await CreateBackup(gameId)
        await loadGames()
        showToast('Backup creado exitosamente', 'success')
      } catch (error) {
        showToast('Error creando backup: ' + error, 'error')
      } finally {
        backingUp.value = backingUp.value.filter(id => id !== gameId)
      }
    }

    const addCustomGame = async () => {
      try {
        const patterns = newGame.value.patterns.split(',').map(p => p.trim())
        await AddCustomGame(newGame.value.name, newGame.value.path, patterns)
        await loadGames()
        showAddGame.value = false
        newGame.value = { name: '', path: '', patterns: '*.sav,*.save,*.dat' }
        showToast('Juego agregado exitosamente', 'success')
      } catch (error) {
        showToast('Error agregando juego: ' + error, 'error')
      }
    }

    const removeGame = async (gameId) => {
      if (confirm('¿Estás seguro de que quieres eliminar este juego de la lista?')) {
        try {
          await RemoveGame(gameId)
          await loadGames()
          showToast('Juego eliminado de la lista', 'success')
        } catch (error) {
          showToast('Error eliminando juego: ' + error, 'error')
        }
      }
    }

    const saveSettings = async () => {
      try {
        await UpdateConfig(config.value)
        showSettings.value = false
        showToast('Configuración guardada', 'success')
      } catch (error) {
        showToast('Error guardando configuración: ' + error, 'error')
      }
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
      loadConfig()
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