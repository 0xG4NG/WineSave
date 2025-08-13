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

//go:embed frontend/dist

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

// BackupInfo representa información de un backup específico
type BackupInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Created    time.Time `json:"created"`
	Compressed bool      `json:"compressed"`
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
