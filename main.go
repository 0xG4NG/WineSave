package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// App struct
type App struct {
	ctx           context.Context
	backupManager *BackupManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// OnStartup is called when the app starts up
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Inicializar el backup manager
	bm, err := NewBackupManager("config.json")
	if err != nil {
		log.Printf("Error inicializando backup manager: %v", err)
		// Crear uno nuevo con configuración por defecto
		bm = &BackupManager{
			Config: BackupConfig{
				BackupDir:          "./game_backups",
				MaxBackups:         10,
				CompressionEnabled: true,
				ExcludePatterns:    []string{"*.tmp", "*.log", "*.cache"},
				AutoBackup:         false,
			},
			DetectedGames: make(map[string]*GameInfo),
			DatabasePath:  "game_saves.json",
		}
	}

	a.backupManager = bm
	log.Println("Aplicación iniciada correctamente")
}

// OnDomReady is called after front-end resources have been loaded
func (a *App) OnDomReady(ctx context.Context) {
	// Cargar base de datos al inicio
	if err := a.backupManager.LoadDatabase(); err != nil {
		log.Printf("Error cargando base de datos: %v", err)
	}
}

// OnBeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
func (a *App) OnBeforeClose(ctx context.Context) (prevent bool) {
	// Guardar configuración y base de datos antes de cerrar
	if err := a.backupManager.SaveConfig("config.json"); err != nil {
		log.Printf("Error guardando configuración: %v", err)
	}

	if err := a.backupManager.SaveDatabase(); err != nil {
		log.Printf("Error guardando base de datos: %v", err)
	}

	return false
}

// OnShutdown is called when the application is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	log.Println("Aplicación cerrada")
}

// Métodos expuestos al frontend

// ScanGames escanea y detecta juegos automáticamente
func (a *App) ScanGames() (*ScanResult, error) {
	log.Println("Iniciando escaneo desde frontend...")
	return a.backupManager.ScanForGames()
}

// GetGameList devuelve la lista de juegos detectados
func (a *App) GetGameList() []*GameInfo {
	return a.backupManager.GetGameList()
}

// CreateBackup crea un backup de un juego específico
func (a *App) CreateBackup(gameID string) error {
	log.Printf("Creando backup para juego: %s", gameID)
	return a.backupManager.CreateBackup(gameID)
}

// AddCustomGame permite agregar un juego personalizado
func (a *App) AddCustomGame(name, savePath string, patterns []string) error {
	log.Printf("Agregando juego personalizado: %s", name)
	return a.backupManager.AddCustomGame(name, savePath, patterns)
}

// SearchGamesOnPCGW busca juegos en PCGamingWiki
func (a *App) SearchGamesOnPCGW(gameName string) ([]GameSearchResult, error) {
	log.Printf("Buscando juegos en PCGamingWiki: %s", gameName)
	return a.backupManager.SearchGamesOnPCGW(gameName)
}

// AddGameFromPCGW agrega un juego desde PCGamingWiki
func (a *App) AddGameFromPCGW(selection UserGameSelection) error {
	log.Printf("Agregando juego desde PCGamingWiki: %s", selection.Name)
	return a.backupManager.AddGameFromPCGW(selection)
}

// GetDefaultBackupPath devuelve la ruta por defecto para backups
func (a *App) GetDefaultBackupPath() string {
	return a.backupManager.GetDefaultBackupPath()
}

// SetBackupPath permite cambiar la ruta de backup
func (a *App) SetBackupPath(newPath string) error {
	log.Printf("Cambiando ruta de backup a: %s", newPath)
	return a.backupManager.SetBackupPath(newPath)
}

// ValidateGamePaths valida las rutas de un juego
func (a *App) ValidateGamePaths(gameID string) (map[string][]string, error) {
	validPaths, invalidPaths := a.backupManager.ValidateGamePaths(gameID)
	return map[string][]string{
		"valid":   validPaths,
		"invalid": invalidPaths,
	}, nil
}

// GetConfig devuelve la configuración actual
func (a *App) GetConfig() BackupConfig {
	return a.backupManager.Config
}

// UpdateConfig actualiza la configuración
func (a *App) UpdateConfig(config BackupConfig) error {
	a.backupManager.Config = config
	return a.backupManager.SaveConfig("config.json")
}

// GetGameInfo devuelve información detallada de un juego
func (a *App) GetGameInfo(gameID string) (*GameInfo, error) {
	game, exists := a.backupManager.DetectedGames[gameID]
	if !exists {
		return nil, fmt.Errorf("juego con ID %s no encontrado", gameID)
	}

	// Actualizar información antes de devolverla
	if err := a.backupManager.updateGameInfo(game); err != nil {
		log.Printf("Error actualizando info del juego %s: %v", gameID, err)
	}

	return game, nil
}

// RemoveGame elimina un juego de la lista detectada
func (a *App) RemoveGame(gameID string) error {
	if _, exists := a.backupManager.DetectedGames[gameID]; !exists {
		return fmt.Errorf("juego con ID %s no encontrado", gameID)
	}

	delete(a.backupManager.DetectedGames, gameID)
	return a.backupManager.SaveDatabase()
}

// ValidatePath verifica si una ruta existe
func (a *App) ValidatePath(path string) bool {
	expandedPath := ExpandPath(path)
	return a.backupManager.gameExists(&GameInfo{SavePaths: []string{expandedPath}})
}

// GetBackupHistory devuelve el historial de backups de un juego
func (a *App) GetBackupHistory(gameID string) ([]BackupInfo, error) {
	// Esta función sería una extensión para mostrar historial
	return []BackupInfo{}, nil
}

// CreateBackupForSelectedGames crea backups para una lista de juegos seleccionados
func (a *App) CreateBackupForSelectedGames(gameNames []string, backupPath string) (*BatchBackupResult, error) {
	log.Printf("Creando backups para %d juegos en: %s", len(gameNames), backupPath)
	
	result := &BatchBackupResult{
		TotalGames:    len(gameNames),
		SuccessCount:  0,
		ErrorCount:    0,
		Errors:        []string{},
		BackupPath:    backupPath,
	}

	// Configurar ruta de backup temporal si se especifica
	originalBackupDir := a.backupManager.Config.BackupDir
	if backupPath != "" {
		a.backupManager.Config.BackupDir = backupPath
	}

	// Primero obtener información detallada de los juegos y agregarlos al sistema
	detailedGames, err := a.GetAvailableGamesForBackup(gameNames)
	if err != nil {
		result.ErrorCount = len(gameNames)
		result.Errors = append(result.Errors, fmt.Sprintf("Error obteniendo información de juegos: %v", err))
		return result, err
	}

	for _, detailedGame := range detailedGames {
		if !detailedGame.Available {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", detailedGame.Name, detailedGame.Reason))
			continue
		}

		// Generar ID para el juego
		gameID := a.backupManager.generateGameID(detailedGame.Name)

		// Agregar el juego al sistema si no existe
		if _, exists := a.backupManager.DetectedGames[gameID]; !exists {
			gameInfo := &GameInfo{
				ID:          gameID,
				Name:        detailedGame.Name,
				Platform:    "pcgw",
				SavePaths:   detailedGame.SavePaths,
				Patterns:    SaveFilePatterns,
				CustomPaths: []string{},
				Metadata:    make(map[string]string),
			}

			if detailedGame.PageID != "" {
				gameInfo.Metadata["pcgw_page_id"] = detailedGame.PageID
			}
			if detailedGame.SteamAppID != "" {
				gameInfo.Metadata["steam_app_id"] = detailedGame.SteamAppID
			}
			if detailedGame.ReleaseDate != "" {
				gameInfo.Metadata["release_date"] = detailedGame.ReleaseDate
			}
			if detailedGame.CoverURL != "" {
				gameInfo.Metadata["cover_url"] = detailedGame.CoverURL
			}

			a.backupManager.DetectedGames[gameID] = gameInfo
			log.Printf("Juego agregado al sistema: %s", detailedGame.Name)
		}

		// Crear backup
		if err := a.backupManager.CreateBackup(gameID); err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", detailedGame.Name, err))
			log.Printf("Error creando backup para %s: %v", detailedGame.Name, err)
		} else {
			result.SuccessCount++
			log.Printf("Backup creado exitosamente para: %s", detailedGame.Name)
		}
	}

	// Guardar la base de datos con los nuevos juegos
	if err := a.backupManager.SaveDatabase(); err != nil {
		log.Printf("Error guardando base de datos: %v", err)
	}

	// Restaurar configuración original
	a.backupManager.Config.BackupDir = originalBackupDir

	return result, nil
}

// GetAvailableGamesForBackup obtiene juegos disponibles con información detallada desde PCGW
func (a *App) GetAvailableGamesForBackup(gameNames []string) ([]*DetailedGameInfo, error) {
	var detailedGames []*DetailedGameInfo

	for _, gameName := range gameNames {
		// Buscar en PCGamingWiki
		searchResults, err := a.backupManager.SearchGamesOnPCGW(gameName)
		if err != nil {
			log.Printf("Error buscando %s en PCGW: %v", gameName, err)
			continue
		}

		if len(searchResults) > 0 {
			// Tomar el primer resultado (más relevante)
			gameResult := searchResults[0]
			
			detailedGame := &DetailedGameInfo{
				Name:        gameResult.Name,
				PageID:      gameResult.PageID,
				SteamAppID:  gameResult.SteamAppID,
				ReleaseDate: gameResult.ReleaseDate,
				CoverURL:    gameResult.CoverURL,
				SavePaths:   gameResult.SavePaths,
				Available:   len(gameResult.SavePaths) > 0,
				Reason:      "",
			}

			if !detailedGame.Available {
				detailedGame.Reason = "No se encontraron rutas de guardado en PCGamingWiki"
			}

			detailedGames = append(detailedGames, detailedGame)
		} else {
			// Juego no encontrado en PCGW
			detailedGames = append(detailedGames, &DetailedGameInfo{
				Name:      gameName,
				Available: false,
				Reason:    "Juego no encontrado en PCGamingWiki",
			})
		}
	}

	return detailedGames, nil
}

// BackupInfo representa información de un backup específico
type BackupInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Created    time.Time `json:"created"`
	Compressed bool      `json:"compressed"`
}

// BatchBackupResult resultado de crear backups en lote
type BatchBackupResult struct {
	TotalGames   int      `json:"total_games"`
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors"`
	BackupPath   string   `json:"backup_path"`
}

// DetailedGameInfo información detallada de un juego para backup
type DetailedGameInfo struct {
	Name        string   `json:"name"`
	PageID      string   `json:"page_id"`
	SteamAppID  string   `json:"steam_app_id"`
	ReleaseDate string   `json:"release_date"`
	CoverURL    string   `json:"cover_url"`
	SavePaths   []string `json:"save_paths"`
	Available   bool     `json:"available"`
	Reason      string   `json:"reason"`
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "Game Save Backup Manager",
		Width:             1200,
		Height:            800,
		MinWidth:          800,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.OnStartup,
		OnDomReady:    app.OnDomReady,
		OnBeforeClose: app.OnBeforeClose,
		OnShutdown:    app.OnShutdown,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
