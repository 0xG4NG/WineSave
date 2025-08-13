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

// App contiene el contexto y el administrador de backups
type App struct {
	ctx           context.Context
	backupManager *BackupManager
}

// NewApp crea una nueva instancia de App
func NewApp() *App {
	return &App{}
}

// OnStartup se ejecuta al iniciar la aplicación
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.initBackupManager()
	log.Println("[INFO] Aplicación iniciada correctamente")
}

// initBackupManager inicializa el gestor de backups con config.json o valores por defecto
func (a *App) initBackupManager() {
	bm, err := NewBackupManager("config.json")
	if err != nil {
		log.Printf("[WARN] Error inicializando backup manager: %v", err)
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
}

// OnDomReady se ejecuta cuando el frontend está listo
func (a *App) OnDomReady(ctx context.Context) {
	if err := a.backupManager.LoadDatabase(); err != nil {
		log.Printf("[WARN] Error cargando base de datos: %v", err)
	}
}

// OnBeforeClose se ejecuta antes de cerrar la aplicación
func (a *App) OnBeforeClose(ctx context.Context) (prevent bool) {
	if err := a.backupManager.SaveConfig("config.json"); err != nil {
		log.Printf("[ERROR] Error guardando configuración: %v", err)
	}
	if err := a.backupManager.SaveDatabase(); err != nil {
		log.Printf("[ERROR] Error guardando base de datos: %v", err)
	}
	return false
}

// OnShutdown se ejecuta cuando la aplicación se está cerrando
func (a *App) OnShutdown(ctx context.Context) {
	log.Println("[INFO] Aplicación cerrada")
}

// ------------------- Métodos expuestos al frontend -------------------

// ScanGames escanea y detecta juegos automáticamente
func (a *App) ScanGames() (*ScanResult, error) {
	log.Println("[INFO] Escaneo iniciado desde frontend...")
	return a.backupManager.ScanForGames()
}

// GetGameList devuelve la lista de juegos detectados
func (a *App) GetGameList() []*GameInfo {
	return a.backupManager.GetGameList()
}

// CreateBackup crea un backup de un juego específico
func (a *App) CreateBackup(gameID string) error {
	log.Printf("[INFO] Creando backup para juego: %s", gameID)
	return a.backupManager.CreateBackup(gameID)
}

// AddCustomGame agrega un juego personalizado
func (a *App) AddCustomGame(name, savePath string, patterns []string) error {
	log.Printf("[INFO] Agregando juego personalizado: %s", name)
	return a.backupManager.AddCustomGame(name, savePath, patterns)
}

// SearchGamesOnPCGW busca juegos en PCGamingWiki
func (a *App) SearchGamesOnPCGW(gameName string) ([]GameSearchResult, error) {
	log.Printf("[INFO] Buscando juegos en PCGamingWiki: %s", gameName)
	return a.backupManager.SearchGamesOnPCGW(gameName)
}

// AddGameFromPCGW agrega un juego desde PCGamingWiki
func (a *App) AddGameFromPCGW(selection UserGameSelection) error {
	log.Printf("[INFO] Agregando juego desde PCGamingWiki: %s", selection.Name)
	return a.backupManager.AddGameFromPCGW(selection)
}

// GetDefaultBackupPath devuelve la ruta por defecto para backups
func (a *App) GetDefaultBackupPath() string {
	return a.backupManager.GetDefaultBackupPath()
}

// SetBackupPath cambia la ruta de backup
func (a *App) SetBackupPath(newPath string) error {
	log.Printf("[INFO] Cambiando ruta de backup a: %s", newPath)
	return a.backupManager.SetBackupPath(newPath)
}

// ValidateGamePaths valida las rutas de guardado de un juego
func (a *App) ValidateGamePaths(gameID string) (map[string][]string, error) {
	valid, invalid := a.backupManager.ValidateGamePaths(gameID)
	return map[string][]string{"valid": valid, "invalid": invalid}, nil
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
	if err := a.backupManager.updateGameInfo(game); err != nil {
		log.Printf("[WARN] Error actualizando info del juego %s: %v", gameID, err)
	}
	return game, nil
}

// RemoveGame elimina un juego detectado
func (a *App) RemoveGame(gameID string) error {
	if _, exists := a.backupManager.DetectedGames[gameID]; !exists {
		return fmt.Errorf("juego con ID %s no encontrado", gameID)
	}
	delete(a.backupManager.DetectedGames, gameID)
	return a.backupManager.SaveDatabase()
}

// ValidatePath verifica si una ruta existe
func (a *App) ValidatePath(path string) bool {
	return a.backupManager.gameExists(&GameInfo{SavePaths: []string{ExpandPath(path)}})
}

// GetBackupHistory devuelve el historial de backups de un juego
func (a *App) GetBackupHistory(gameID string) ([]BackupInfo, error) {
	// Implementar si se requiere
	return []BackupInfo{}, nil
}

// ------------------- Tipos de datos -------------------

type BackupInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Created    time.Time `json:"created"`
	Compressed bool      `json:"compressed"`
}

type BatchBackupResult struct {
	TotalGames   int      `json:"total_games"`
	SuccessCount int      `json:"success_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors"`
	BackupPath   string   `json:"backup_path"`
}

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

// ------------------- main -------------------

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Game Save Backup Manager",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.OnStartup,
		OnDomReady:    app.OnDomReady,
		OnBeforeClose: app.OnBeforeClose,
		OnShutdown:    app.OnShutdown,
		Bind: []interface{}{
			app, // <- Esto es lo que expone tus métodos al frontend
		},
	})

	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
}
