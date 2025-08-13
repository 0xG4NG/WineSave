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
          <button @click="showGameSelectionWizard = true" class="btn btn-primary">
            🎯 Seleccionar Juegos
          </button>
          <button @click="showSettings = true" class="btn btn-secondary">
            ⚙️ Configuración
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <!-- Welcome Section -->
      <div class="welcome-section">
        <div class="welcome-card">
          <div class="welcome-icon">🎮</div>
          <h2>Backup de Partidas de Juegos</h2>
          <p>Selecciona los juegos que quieres respaldar y el programa buscará automáticamente dónde se guardan las partidas usando PCGamingWiki</p>
          <button @click="showGameSelectionWizard = true" class="btn btn-primary btn-large">
            🎯 Comenzar Selección de Juegos
          </button>
        </div>
      </div>

      <!-- Selected Games Section (if any) -->
      <div v-if="selectedGamesForBackup.length > 0" class="selected-games-section">
        <div class="section-header">
          <h2>Juegos Seleccionados para Backup ({{ selectedGamesForBackup.length }})</h2>
          <button @click="startBatchBackup" :disabled="creatingBackups" class="btn btn-success">
            <span v-if="creatingBackups" class="loading-spinner"></span>
            {{ creatingBackups ? 'Creando Backups...' : '💾 Crear Backups' }}
          </button>
        </div>

        <div class="selected-games-grid">
          <div 
            v-for="game in selectedGamesForBackup" 
            :key="game.name"
            class="selected-game-card"
            :class="{ 
              'available': game.available, 
              'unavailable': !game.available,
              'backing-up': backingUpGames.includes(game.name)
            }"
          >
            <div class="game-status">
              <span v-if="game.available" class="status-icon success">✓</span>
              <span v-else class="status-icon error">✗</span>
            </div>
            
            <div class="game-info">
              <h3>{{ game.name }}</h3>
              <p v-if="game.available" class="save-paths-info">
                📁 {{ game.save_paths?.length || 0 }} ruta(s) de guardado encontrada(s)
              </p>
              <p v-else class="error-reason">{{ game.reason }}</p>
              
              <div v-if="game.save_paths && game.save_paths.length > 0" class="paths-preview">
                <details>
                  <summary>Ver rutas de guardado</summary>
                  <ul>
                    <li v-for="path in game.save_paths" :key="path">{{ path }}</li>
                  </ul>
                </details>
              </div>
            </div>

            <div class="game-actions">
              <button @click="removeSelectedGame(game.name)" class="btn btn-danger btn-small">
                🗑️ Quitar
              </button>
            </div>
          </div>
        </div>

        <!-- Backup Destination -->
        <div class="backup-destination">
          <h3>📁 Destino del Backup</h3>
          <div class="destination-input">
            <input 
              v-model="backupDestination" 
              type="text" 
              class="form-input"
              placeholder="~/WineSaveBackups"
            />
            <button @click="selectBackupDestination" class="btn btn-secondary">
              📂 Seleccionar Carpeta
            </button>
          </div>
          <small>Los backups se guardarán como archivos ZIP en esta ubicación</small>
        </div>
      </div>

      <!-- Recent Backups Section -->
      <div v-if="recentBackups.length > 0" class="recent-backups-section">
        <h2>📋 Backups Recientes</h2>
        <div class="backups-list">
          <div v-for="backup in recentBackups" :key="backup.id" class="backup-item">
            <div class="backup-info">
              <h4>{{ backup.games.join(', ') }}</h4>
              <p>{{ formatDate(backup.created) }} - {{ backup.success_count }}/{{ backup.total_games }} exitosos</p>
            </div>
            <div class="backup-actions">
              <button @click="openBackupLocation(backup.path)" class="btn btn-secondary btn-small">
                📂 Abrir
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Game Selection Wizard Modal -->
    <div v-if="showGameSelectionWizard" class="modal-overlay" @click="closeGameSelectionWizard">
      <div class="modal large" @click.stop>
        <div class="modal-header">
          <h2>🎯 Seleccionar Juegos para Backup</h2>
          <button @click="closeGameSelectionWizard" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <div class="game-selection-step">
            <h3>📝 Ingresa los nombres de los juegos</h3>
            <p>Escribe los nombres de los juegos que quieres respaldar, uno por línea. El programa buscará automáticamente en PCGamingWiki dónde guardan las partidas.</p>
            
            <div class="form-group">
              <label>Lista de Juegos</label>
              <textarea 
                v-model="gameNamesInput" 
                class="form-input textarea-large"
                placeholder="Ej:
Elden Ring
Cyberpunk 2077
The Witcher 3
Dark Souls III"
                rows="10"
              ></textarea>
              <small>Un juego por línea. Puedes copiar y pegar desde cualquier lugar.</small>
            </div>

            <div class="wizard-actions">
              <button @click="closeGameSelectionWizard" class="btn btn-secondary">
                Cancelar
              </button>
              <button 
                @click="processSelectedGames" 
                :disabled="processing || !gameNamesInput.trim()"
                class="btn btn-primary"
              >
                <span v-if="processing" class="loading-spinner"></span>
                {{ processing ? 'Buscando en PCGamingWiki...' : '🔍 Buscar Juegos' }}
              </button>
            </div>

            <!-- Progress indicator -->
            <div v-if="processing" class="progress-info">
              <p>Buscando información de guardado para cada juego...</p>
              <div class="progress-details">
                <p>Procesando: {{ currentlyProcessing }}</p>
                <p>Progreso: {{ processedGames }}/{{ totalGamesToProcess }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Settings Modal -->
    <div v-if="showSettings" class="modal-overlay" @click="showSettings = false">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h2>⚙️ Configuración</h2>
          <button @click="showSettings = false" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <form @submit.prevent="saveSettings">
            <div class="form-group">
              <label>Directorio de Backups por Defecto</label>
              <div class="path-input-group">
                <input v-model="config.backup_dir" type="text" class="form-input" />
                <button type="button" @click="selectDefaultBackupFolder" class="btn btn-secondary">
                  📁 Seleccionar
                </button>
              </div>
              <small>Ubicación donde se guardarán todos los backups por defecto</small>
            </div>
            
            <div class="form-group">
              <label>Máximo de Backups por Juego</label>
              <input v-model="config.max_backups" type="number" min="1" max="50" class="form-input" />
              <small>Número máximo de backups a mantener por juego (los más antiguos se eliminan automáticamente)</small>
            </div>
            
            <div class="form-group">
              <label class="checkbox-label">
                <input v-model="config.compression_enabled" type="checkbox" />
                Habilitar Compresión ZIP
              </label>
              <small>Los backups se guardarán comprimidos en formato ZIP</small>
            </div>
            
            <div class="form-group">
              <label class="checkbox-label">
                <input v-model="config.auto_backup" type="checkbox" />
                Backup Automático (Experimental)
              </label>
              <small>Crear backups automáticamente según el intervalo configurado</small>
            </div>
            
            <div class="form-actions">
              <button type="button" @click="showSettings = false" class="btn btn-secondary">
                Cancelar
              </button>
              <button type="submit" class="btn btn-primary">
                💾 Guardar Configuración
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
          <h2>📋 {{ selectedGame.name }} - Detalles</h2>
          <button @click="selectedGame = null" class="close-btn">✕</button>
        </div>
        <div class="modal-content">
          <div class="details-grid">
            <div class="detail-section">
              <h3>ℹ️ Información General</h3>
              <div class="detail-item">
                <strong>ID:</strong> {{ selectedGame.id }}
              </div>
              <div class="detail-item">
                <strong>Plataforma:</strong> {{ selectedGame.platform }}
              </div>
              <div class="detail-item">
                <strong>Archivos detectados:</strong> {{ selectedGame.file_count || 0 }}
              </div>
              <div class="detail-item">
                <strong>Tamaño total:</strong> {{ formatSize(selectedGame.total_size || 0) }}
              </div>
              <div class="detail-item">
                <strong>Último backup:</strong> {{ formatDate(selectedGame.last_backup) }}
              </div>
              <div v-if="selectedGame.metadata?.steam_app_id" class="detail-item">
                <strong>Steam App ID:</strong> {{ selectedGame.metadata.steam_app_id }}
              </div>
              <div v-if="selectedGame.metadata?.pcgw_page_id" class="detail-item">
                <strong>PCGamingWiki:</strong> 
                <a :href="`https://www.pcgamingwiki.com/wiki/Special:CargoExport?tables=Infobox_game&where=_pageID=${selectedGame.metadata.pcgw_page_id}`" 
                   target="_blank" class="external-link">
                  Ver en PCGW
                </a>
              </div>
            </div>
            
            <div class="detail-section">
              <h3>📁 Rutas de Guardado</h3>
              <ul class="paths-list">
                <li v-for="path in selectedGame.save_paths" :key="path">
                  {{ path }}
                </li>
              </ul>
              <button @click="validatePaths(selectedGame.id)" class="btn btn-info" style="margin-top: 1rem;">
                ✓ Validar Rutas
              </button>
            </div>
            
            <div class="detail-section">
              <h3>🔍 Patrones de Archivos</h3>
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
import { 
  GetConfig, 
  UpdateConfig, 
  GetDefaultBackupPath,
  SetBackupPath,
  GetAvailableGamesForBackup,
  CreateBackupForSelectedGames
} from '../wailsjs/go/main/App'

export default {
  name: 'App',
  setup() {
    // Reactive data
    const showSettings = ref(false)
    const toasts = ref([])
    const config = ref({
      backup_dir: './game_backups',
      max_backups: 10,
      compression_enabled: true,
      auto_backup: false
    })

    // Game selection workflow
    const showGameSelectionWizard = ref(false)
    const gameNamesInput = ref('')
    const processing = ref(false)
    const currentlyProcessing = ref('')
    const processedGames = ref(0)
    const totalGamesToProcess = ref(0)
    const selectedGamesForBackup = ref([])
    
    // Backup workflow
    const backupDestination = ref('')
    const defaultBackupPath = ref('~/WineSaveBackups')
    const creatingBackups = ref(false)
    const backingUpGames = ref([])
    const recentBackups = ref([])

    // Computed properties
    const availableGamesCount = computed(() => {
      return selectedGamesForBackup.value.filter(g => g.available).length
    })

    // Methods
    const loadConfig = async () => {
      try {
        const cfg = await GetConfig()
        config.value = cfg
        const defaultPath = await GetDefaultBackupPath()
        defaultBackupPath.value = defaultPath
        backupDestination.value = defaultPath
      } catch (error) {
        console.error('Error cargando configuración:', error)
      }
    }

    const processSelectedGames = async () => {
      if (!gameNamesInput.value.trim()) {
        showToast('Ingresa al menos un nombre de juego', 'warning')
        return
      }

      try {
        processing.value = true
        
        // Parse game names
        const gameNames = gameNamesInput.value
          .split('\n')
          .map(name => name.trim())
          .filter(name => name.length > 0)

        totalGamesToProcess.value = gameNames.length
        processedGames.value = 0
        
        showToast(`Buscando información para ${gameNames.length} juego(s)...`, 'info')
        
        // Process games in batches to show progress
        const results = []
        for (const gameName of gameNames) {
          currentlyProcessing.value = gameName
          
          try {
            const gameResults = await GetAvailableGamesForBackup([gameName])
            if (gameResults && gameResults.length > 0) {
              results.push(gameResults[0])
            }
          } catch (error) {
            console.error(`Error processing ${gameName}:`, error)
            results.push({
              name: gameName,
              available: false,
              reason: 'Error al procesar: ' + error
            })
          }
          
          processedGames.value++
          
          // Small delay to show progress
          await new Promise(resolve => setTimeout(resolve, 100))
        }

        selectedGamesForBackup.value = results
        const availableCount = results.filter(g => g.available).length
        
        showToast(
          `Procesamiento completado: ${availableCount}/${gameNames.length} juegos con rutas de guardado encontradas`, 
          availableCount > 0 ? 'success' : 'warning'
        )
        
        closeGameSelectionWizard()
        
      } catch (error) {
        showToast('Error procesando juegos: ' + error, 'error')
      } finally {
        processing.value = false
        currentlyProcessing.value = ''
      }
    }

    const closeGameSelectionWizard = () => {
      showGameSelectionWizard.value = false
      gameNamesInput.value = ''
      processing.value = false
      currentlyProcessing.value = ''
      processedGames.value = 0
      totalGamesToProcess.value = 0
    }

    const removeSelectedGame = (gameName) => {
      selectedGamesForBackup.value = selectedGamesForBackup.value.filter(g => g.name !== gameName)
      showToast(`${gameName} eliminado de la lista`, 'info')
    }

    const startBatchBackup = async () => {
      const availableGames = selectedGamesForBackup.value.filter(g => g.available)
      
      if (availableGames.length === 0) {
        showToast('No hay juegos disponibles para backup', 'warning')
        return
      }

      try {
        creatingBackups.value = true
        backingUpGames.value = availableGames.map(g => g.name)
        
        // Create backup path if needed
        if (backupDestination.value && backupDestination.value !== defaultBackupPath.value) {
          await SetBackupPath(backupDestination.value)
        }

        showToast(`Iniciando backup de ${availableGames.length} juego(s)...`, 'info')
        
        // Pasar los nombres de los juegos directamente
        const gameNames = availableGames.map(game => game.name)
        
        const result = await CreateBackupForSelectedGames(gameNames, backupDestination.value)
        
        // Add to recent backups
        const backupRecord = {
          id: Date.now(),
          games: availableGames.map(g => g.name),
          created: new Date(),
          path: backupDestination.value,
          total_games: result.total_games,
          success_count: result.success_count,
          error_count: result.error_count
        }
        
        recentBackups.value.unshift(backupRecord)
        // Keep only last 10 backups
        if (recentBackups.value.length > 10) {
          recentBackups.value = recentBackups.value.slice(0, 10)
        }
        
        if (result.error_count > 0) {
          showToast(
            `Backup completado con errores: ${result.success_count}/${result.total_games} exitosos`, 
            'warning'
          )
        } else {
          showToast(`Backup completado exitosamente: ${result.success_count} juegos`, 'success')
        }
        
      } catch (error) {
        showToast('Error creando backups: ' + error, 'error')
      } finally {
        creatingBackups.value = false
        backingUpGames.value = []
      }
    }

    const saveSettings = async () => {
      try {
        await UpdateConfig(config.value)
        showSettings.value = false
        showToast('Configuración guardada exitosamente', 'success')
      } catch (error) {
        showToast('Error guardando configuración: ' + error, 'error')
      }
    }

    const selectBackupDestination = async () => {
      // En una implementación real, esto abriría un diálogo de carpetas
      showToast('Funcionalidad de selección de carpeta pendiente de implementar', 'info')
    }

    const selectDefaultBackupFolder = async () => {
      // En una implementación real, esto abriría un diálogo de carpetas
      showToast('Funcionalidad de selección de carpeta pendiente de implementar', 'info')
    }

    const openBackupLocation = (path) => {
      showToast(`Abriendo: ${path}`, 'info')
      // En una implementación real, esto abriría el explorador de archivos
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
      return new Date(dateStr).toLocaleDateString() + ' ' + new Date(dateStr).toLocaleTimeString()
    }

    // Lifecycle
    onMounted(() => {
      loadConfig()
    })

    return {
      // State
      showSettings,
      toasts,
      config,
      showGameSelectionWizard,
      gameNamesInput,
      processing,
      currentlyProcessing,
      processedGames,
      totalGamesToProcess,
      selectedGamesForBackup,
      backupDestination,
      defaultBackupPath,
      creatingBackups,
      backingUpGames,
      recentBackups,
      availableGamesCount,
      
      // Methods
      processSelectedGames,
      closeGameSelectionWizard,
      removeSelectedGame,
      startBatchBackup,
      saveSettings,
      selectBackupDestination,
      selectDefaultBackupFolder,
      openBackupLocation,
      showToast,
      removeToast,
      formatSize,
      formatDate
    }
  }
}
</script>