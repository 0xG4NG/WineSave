package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Estructuras principales
type GameInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	SavePaths   []string          `json:"save_paths"`
	Patterns    []string          `json:"patterns"`
	Platform    string            `json:"platform"`
	LastBackup  time.Time         `json:"last_backup"`
	TotalSize   int64             `json:"total_size"`
	FileCount   int               `json:"file_count"`
	CustomPaths []string          `json:"custom_paths"`
	Metadata    map[string]string `json:"metadata"`
}

type BackupConfig struct {
	BackupDir          string        `json:"backup_dir"`
	MaxBackups         int           `json:"max_backups"`
	CompressionEnabled bool          `json:"compression_enabled"`
	ScanInterval       time.Duration `json:"scan_interval"`
	ExcludePatterns    []string      `json:"exclude_patterns"`
	AutoBackup         bool          `json:"auto_backup"`
}

// BackupManager estructura principal con cliente PCGamingWiki
type BackupManager struct {
	Config        BackupConfig         `json:"config"`
	DetectedGames map[string]*GameInfo `json:"detected_games"`
	DatabasePath  string               `json:"database_path"`
	PCGWClient    *PCGWClient          `json:"-"` // No serializar el cliente
}

// UserGameSelection representa la selección de un usuario
type UserGameSelection struct {
	Name         string            `json:"name"`
	SelectedGame *GameSearchResult `json:"selected_game"`
	CustomPath   string            `json:"custom_path"`
	BackupPath   string            `json:"backup_path"`
}

type ScanResult struct {
	TotalGames int           `json:"total_games"`
	NewGames   []*GameInfo   `json:"new_games"`
	Updated    []*GameInfo   `json:"updated"`
	Errors     []string      `json:"errors"`
	ScanTime   time.Duration `json:"scan_time"`
}

// Definición de ubicaciones comunes de guardado para diferentes juegos
var CommonSavePaths = map[string][]string{
	"steam": {
		"%USERPROFILE%/Documents/My Games",
		"%APPDATA%",
		"%LOCALAPPDATA%",
		"%USERPROFILE%/Saved Games",
		"C:/Program Files (x86)/Steam/userdata",
		"C:/Program Files/Steam/userdata",
	},
	"epic": {
		"%LOCALAPPDATA%/EpicGamesLauncher/Saved",
		"%USERPROFILE%/Documents/My Games",
	},
	"uplay": {
		"%USERPROFILE%/Documents/My Games",
		"%APPDATA%/Ubisoft",
	},
	"origin": {
		"%USERPROFILE%/Documents/Electronic Arts",
		"%LOCALAPPDATA%/Electronic Arts",
	},
	"gog": {
		"%USERPROFILE%/Documents/My Games",
		"%APPDATA%/GOG.com",
	},
	"xbox": {
		"%LOCALAPPDATA%/Packages",
		"%USERPROFILE%/Documents/My Games",
	},
}

// Patrones de archivos de guardado comunes
var SaveFilePatterns = []string{
	"*.sav", "*.save", "*.dat", "*.bin", "*.cfg",
	"save*", "*.slot", "profile*", "*.bak",
	"*.json", "*.xml", "*.ini", "*.txt", "*.sl2",
}

// Juegos específicos con ubicaciones conocidas
var KnownGames = map[string]*GameInfo{
	"elden-ring": {
		ID:       "elden-ring",
		Name:     "Elden Ring",
		Platform: "steam",
		SavePaths: []string{
			"%APPDATA%/EldenRing",
		},
		Patterns: []string{"*.sl2"},
		Metadata: map[string]string{
			"publisher": "FromSoftware",
			"genre":     "Action RPG",
		},
	},
	"dark-souls-3": {
		ID:       "dark-souls-3",
		Name:     "Dark Souls III",
		Platform: "steam",
		SavePaths: []string{
			"%APPDATA%/DarkSoulsIII",
		},
		Patterns: []string{"*.sl2"},
		Metadata: map[string]string{
			"publisher": "FromSoftware",
			"genre":     "Action RPG",
		},
	},
	"cyberpunk-2077": {
		ID:       "cyberpunk-2077",
		Name:     "Cyberpunk 2077",
		Platform: "multiple",
		SavePaths: []string{
			"%USERPROFILE%/Saved Games/CD Projekt Red/Cyberpunk 2077",
		},
		Patterns: []string{"*.dat", "*.json"},
		Metadata: map[string]string{
			"publisher": "CD Projekt RED",
			"genre":     "Action RPG",
		},
	},
	"witcher-3": {
		ID:       "witcher-3",
		Name:     "The Witcher 3: Wild Hunt",
		Platform: "multiple",
		SavePaths: []string{
			"%USERPROFILE%/Documents/The Witcher 3",
		},
		Patterns: []string{"*.sav"},
		Metadata: map[string]string{
			"publisher": "CD Projekt RED",
			"genre":     "Action RPG",
		},
	},
	"skyrim-se": {
		ID:       "skyrim-se",
		Name:     "The Elder Scrolls V: Skyrim Special Edition",
		Platform: "steam",
		SavePaths: []string{
			"%USERPROFILE%/Documents/My Games/Skyrim Special Edition",
		},
		Patterns: []string{"*.ess", "*.skse"},
		Metadata: map[string]string{
			"publisher": "Bethesda",
			"genre":     "Action RPG",
		},
	},
	"fallout-4": {
		ID:       "fallout-4",
		Name:     "Fallout 4",
		Platform: "steam",
		SavePaths: []string{
			"%USERPROFILE%/Documents/My Games/Fallout4",
		},
		Patterns: []string{"*.fos", "*.f4se"},
		Metadata: map[string]string{
			"publisher": "Bethesda",
			"genre":     "Action RPG",
		},
	},
	"minecraft": {
		ID:       "minecraft",
		Name:     "Minecraft",
		Platform: "multiple",
		SavePaths: []string{
			"%APPDATA%/.minecraft/saves",
		},
		Patterns: []string{"level.dat", "*.mca", "*.dat"},
		Metadata: map[string]string{
			"publisher": "Mojang Studios",
			"genre":     "Sandbox",
		},
	},
}

// NewBackupManager crea una nueva instancia del manager de backups
func NewBackupManager(configPath string) (*BackupManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	defaultBackupDir := filepath.Join(homeDir, "WineSaveBackups")

	bm := &BackupManager{
		Config: BackupConfig{
			BackupDir:          defaultBackupDir,
			MaxBackups:         10,
			CompressionEnabled: true,
			ScanInterval:       time.Hour * 24,
			ExcludePatterns:    []string{"*.tmp", "*.log", "*.cache", "*.lock"},
			AutoBackup:         false,
		},
		DetectedGames: make(map[string]*GameInfo),
		DatabasePath:  "game_saves.json",
		PCGWClient:    NewPCGWClient(),
	}

	// Cargar configuración si existe
	if _, err := os.Stat(configPath); err == nil {
		if err := bm.LoadConfig(configPath); err != nil {
			log.Printf("Error cargando configuración: %v", err)
		}
	}

	// Cargar base de datos de juegos detectados
	if err := bm.LoadDatabase(); err != nil {
		log.Printf("Error cargando base de datos: %v", err)
	}

	return bm, nil
}

// ExpandPath expande variables de entorno en rutas de Windows/Linux/macOS
func ExpandPath(path string) string {
	// Variables de Windows
	expanded := strings.ReplaceAll(path, "%USERPROFILE%", os.Getenv("USERPROFILE"))
	expanded = strings.ReplaceAll(expanded, "%APPDATA%", os.Getenv("APPDATA"))
	expanded = strings.ReplaceAll(expanded, "%LOCALAPPDATA%", os.Getenv("LOCALAPPDATA"))
	expanded = strings.ReplaceAll(expanded, "%PROGRAMFILES%", os.Getenv("PROGRAMFILES"))
	expanded = strings.ReplaceAll(expanded, "%PROGRAMFILES(X86)%", os.Getenv("PROGRAMFILES(X86)"))

	// Variables de Unix (Linux/macOS)
	if home := os.Getenv("HOME"); home != "" {
		expanded = strings.ReplaceAll(expanded, "~", home)
		expanded = strings.ReplaceAll(expanded, "$HOME", home)
	}

	// Variables adicionales de Linux
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		expanded = strings.ReplaceAll(expanded, "$XDG_CONFIG_HOME", xdgConfig)
	}

	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		expanded = strings.ReplaceAll(expanded, "$XDG_DATA_HOME", xdgData)
	}

	return expanded
}

// ScanForGames busca automáticamente juegos y sus archivos de guardado
func (bm *BackupManager) ScanForGames() (*ScanResult, error) {
	startTime := time.Now()
	result := &ScanResult{
		NewGames: []*GameInfo{},
		Updated:  []*GameInfo{},
		Errors:   []string{},
	}

	log.Println("Iniciando escaneo de juegos...")

	// Primero, agregar juegos conocidos a la base de datos
	for id, game := range KnownGames {
		if _, exists := bm.DetectedGames[id]; !exists {
			// Verificar si el juego realmente existe
			if bm.gameExists(game) {
				newGame := *game // Copiar estructura
				// Inicializar mapas si son nil
				if newGame.Metadata == nil {
					newGame.Metadata = make(map[string]string)
				}
				if newGame.CustomPaths == nil {
					newGame.CustomPaths = []string{}
				}
				bm.DetectedGames[id] = &newGame
				result.NewGames = append(result.NewGames, &newGame)
				log.Printf("Juego conocido detectado: %s", game.Name)
			}
		}
	}

	// Escanear ubicaciones comunes para detectar nuevos juegos
	for platform, paths := range CommonSavePaths {
		for _, basePath := range paths {
			expandedPath := ExpandPath(basePath)
			if err := bm.scanDirectory(expandedPath, platform, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error escaneando %s: %v", expandedPath, err))
			}
		}
	}

	// Actualizar información de juegos existentes
	for _, game := range bm.DetectedGames {
		if err := bm.updateGameInfo(game); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error actualizando %s: %v", game.Name, err))
		} else {
			result.Updated = append(result.Updated, game)
		}
	}

	result.TotalGames = len(bm.DetectedGames)
	result.ScanTime = time.Since(startTime)

	log.Printf("Escaneo completado: %d juegos detectados, %d nuevos, %d actualizados",
		result.TotalGames, len(result.NewGames), len(result.Updated))

	return result, bm.SaveDatabase()
}

// gameExists verifica si un juego realmente existe verificando sus rutas de guardado
func (bm *BackupManager) gameExists(game *GameInfo) bool {
	for _, path := range game.SavePaths {
		expandedPath := ExpandPath(path)
		if _, err := os.Stat(expandedPath); err == nil {
			return true
		}
	}
	return false
}

// scanDirectory escanea un directorio en busca de posibles archivos de guardado
func (bm *BackupManager) scanDirectory(path, platform string, result *ScanResult) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Directorio no existe, continuar
	}

	return filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continuar con otros directorios
		}

		if d.IsDir() {
			// Verificar si este directorio parece contener archivos de guardado
			if bm.looksLikeSaveDirectory(currentPath) {
				gameID := bm.generateGameID(currentPath)
				if _, exists := bm.DetectedGames[gameID]; !exists {
					// Crear nueva entrada de juego
					game := &GameInfo{
						ID:          gameID,
						Name:        bm.inferGameName(currentPath),
						Platform:    platform,
						SavePaths:   []string{currentPath},
						Patterns:    SaveFilePatterns,
						CustomPaths: []string{},
						Metadata:    make(map[string]string),
					}

					bm.DetectedGames[gameID] = game
					result.NewGames = append(result.NewGames, game)
					log.Printf("Nuevo juego detectado: %s en %s", game.Name, currentPath)
				}
			}
		}

		return nil
	})
}

// looksLikeSaveDirectory determina si un directorio parece contener archivos de guardado
func (bm *BackupManager) looksLikeSaveDirectory(path string) bool {
	// Buscar archivos que coincidan con patrones de guardado
	files, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	saveFileCount := 0
	totalFiles := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		totalFiles++
		fileName := strings.ToLower(file.Name())

		// Verificar patrones específicos
		for _, pattern := range SaveFilePatterns {
			if matched, _ := filepath.Match(strings.ToLower(pattern), fileName); matched {
				saveFileCount++
				break
			}
		}

		// Buscar palabras clave en nombres de archivo
		keywords := []string{"save", "profile", "config", "settings", "user", "player"}
		for _, keyword := range keywords {
			if strings.Contains(fileName, keyword) {
				saveFileCount++
				break
			}
		}
	}

	// Considerar como directorio de guardado si:
	// - Tiene al menos 1 archivo que parece de guardado
	// - O si es un directorio pequeño (menos de 20 archivos) con archivos de configuración
	return saveFileCount >= 1 || (totalFiles > 0 && totalFiles < 20 && saveFileCount > 0)
}

// generateGameID genera un ID único para un juego basado en su ruta
func (bm *BackupManager) generateGameID(path string) string {
	// Extraer el nombre del directorio del juego
	parts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	if len(parts) > 0 {
		gameName := parts[len(parts)-1]
		// Limpiar el nombre para usarlo como ID
		re := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
		id := re.ReplaceAllString(strings.ToLower(gameName), "-")
		return strings.Trim(id, "-")
	}
	return fmt.Sprintf("unknown-game-%d", time.Now().Unix())
}

// inferGameName infiere el nombre del juego desde su ruta
func (bm *BackupManager) inferGameName(path string) string {
	parts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Limpiar y mejorar el nombre
		name = strings.ReplaceAll(name, "_", " ")
		name = strings.ReplaceAll(name, "-", " ")
		// Capitalizar primera letra de cada palabra
		words := strings.Fields(name)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}
	return "Juego Desconocido"
}

// updateGameInfo actualiza la información de un juego (tamaño, número de archivos, etc.)
func (bm *BackupManager) updateGameInfo(game *GameInfo) error {
	var totalSize int64
	var fileCount int

	for _, savePath := range game.SavePaths {
		expandedPath := ExpandPath(savePath)

		err := filepath.WalkDir(expandedPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() {
				if bm.matchesPatterns(d.Name(), game.Patterns) && !bm.isExcluded(d.Name()) {
					if info, err := d.Info(); err == nil {
						totalSize += info.Size()
						fileCount++
					}
				}
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	game.TotalSize = totalSize
	game.FileCount = fileCount

	return nil
}

// matchesPatterns verifica si un archivo coincide con los patrones del juego
func (bm *BackupManager) matchesPatterns(filename string, patterns []string) bool {
	filename = strings.ToLower(filename)
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), filename); matched {
			return true
		}
	}
	return false
}

// isExcluded verifica si un archivo debe ser excluido del backup
func (bm *BackupManager) isExcluded(filename string) bool {
	filename = strings.ToLower(filename)
	for _, pattern := range bm.Config.ExcludePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), filename); matched {
			return true
		}
	}
	return false
}

// CreateBackup crea un backup de un juego específico
func (bm *BackupManager) CreateBackup(gameID string) error {
	game, exists := bm.DetectedGames[gameID]
	if !exists {
		return fmt.Errorf("juego con ID %s no encontrado", gameID)
	}

	log.Printf("Creando backup para: %s", game.Name)

	// Crear directorio de backup si no existe
	backupDir := filepath.Join(bm.Config.BackupDir, game.ID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de backup: %v", err)
	}

	// Generar nombre de archivo de backup con timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	var backupPath string

	if bm.Config.CompressionEnabled {
		backupPath = filepath.Join(backupDir, fmt.Sprintf("%s_%s.zip", game.ID, timestamp))
		if err := bm.createZipBackup(game, backupPath); err != nil {
			return err
		}
	} else {
		backupPath = filepath.Join(backupDir, fmt.Sprintf("%s_%s", game.ID, timestamp))
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return err
		}
		if err := bm.createFolderBackup(game, backupPath); err != nil {
			return err
		}
	}

	game.LastBackup = time.Now()
	log.Printf("Backup creado exitosamente: %s", backupPath)

	// Limpiar backups antiguos
	if err := bm.cleanOldBackups(game.ID); err != nil {
		log.Printf("Error limpiando backups antiguos: %v", err)
	}

	return bm.SaveDatabase()
}

// createZipBackup crea un backup comprimido en ZIP
func (bm *BackupManager) createZipBackup(game *GameInfo, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, savePath := range game.SavePaths {
		expandedPath := ExpandPath(savePath)

		err := filepath.WalkDir(expandedPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() && bm.matchesPatterns(d.Name(), game.Patterns) && !bm.isExcluded(d.Name()) {
				relPath, _ := filepath.Rel(expandedPath, path)

				zipEntry, err := zipWriter.Create(relPath)
				if err != nil {
					return err
				}

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(zipEntry, file)
				return err
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// createFolderBackup crea un backup en carpeta sin comprimir
func (bm *BackupManager) createFolderBackup(game *GameInfo, backupPath string) error {
	for _, savePath := range game.SavePaths {
		expandedPath := ExpandPath(savePath)

		err := filepath.WalkDir(expandedPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() && bm.matchesPatterns(d.Name(), game.Patterns) && !bm.isExcluded(d.Name()) {
				relPath, _ := filepath.Rel(expandedPath, path)
				destPath := filepath.Join(backupPath, relPath)

				// Crear directorio destino si no existe
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}

				// Copiar archivo
				return copyFile(path, destPath)
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// copyFile copia un archivo de origen a destino
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// cleanOldBackups elimina backups antiguos manteniendo solo los más recientes
func (bm *BackupManager) cleanOldBackups(gameID string) error {
	backupDir := filepath.Join(bm.Config.BackupDir, gameID)

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	// Filtrar solo archivos de backup y ordenar por fecha
	var backupFiles []fs.DirEntry
	for _, file := range files {
		if strings.Contains(file.Name(), gameID) {
			backupFiles = append(backupFiles, file)
		}
	}

	if len(backupFiles) <= bm.Config.MaxBackups {
		return nil
	}

	// Ordenar por fecha de modificación (más reciente primero)
	sort.Slice(backupFiles, func(i, j int) bool {
		infoI, _ := backupFiles[i].Info()
		infoJ, _ := backupFiles[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Eliminar backups antiguos
	for i := bm.Config.MaxBackups; i < len(backupFiles); i++ {
		filePath := filepath.Join(backupDir, backupFiles[i].Name())
		if err := os.RemoveAll(filePath); err != nil {
			log.Printf("Error eliminando backup antiguo %s: %v", filePath, err)
		} else {
			log.Printf("Backup antiguo eliminado: %s", filePath)
		}
	}

	return nil
}

// LoadConfig carga la configuración desde un archivo JSON
func (bm *BackupManager) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &bm.Config)
}

// SaveConfig guarda la configuración actual en un archivo JSON
func (bm *BackupManager) SaveConfig(path string) error {
	data, err := json.MarshalIndent(bm.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadDatabase carga la base de datos de juegos detectados
func (bm *BackupManager) LoadDatabase() error {
	if _, err := os.Stat(bm.DatabasePath); os.IsNotExist(err) {
		return nil // No hay base de datos, empezar limpio
	}

	data, err := os.ReadFile(bm.DatabasePath)
	if err != nil {
		return err
	}

	var dbData struct {
		DetectedGames map[string]*GameInfo `json:"detected_games"`
	}

	if err := json.Unmarshal(data, &dbData); err != nil {
		return err
	}

	bm.DetectedGames = dbData.DetectedGames
	if bm.DetectedGames == nil {
		bm.DetectedGames = make(map[string]*GameInfo)
	}

	// Inicializar mapas nil para evitar errores
	for _, game := range bm.DetectedGames {
		if game.Metadata == nil {
			game.Metadata = make(map[string]string)
		}
		if game.CustomPaths == nil {
			game.CustomPaths = []string{}
		}
	}

	return nil
}

// SaveDatabase guarda la base de datos de juegos detectados
func (bm *BackupManager) SaveDatabase() error {
	dbData := struct {
		DetectedGames map[string]*GameInfo `json:"detected_games"`
		LastUpdate    time.Time            `json:"last_update"`
	}{
		DetectedGames: bm.DetectedGames,
		LastUpdate:    time.Now(),
	}

	data, err := json.MarshalIndent(dbData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bm.DatabasePath, data, 0644)
}

// GetGameList devuelve la lista de juegos detectados
func (bm *BackupManager) GetGameList() []*GameInfo {
	games := make([]*GameInfo, 0, len(bm.DetectedGames))
	for _, game := range bm.DetectedGames {
		games = append(games, game)
	}

	// Ordenar por nombre
	sort.Slice(games, func(i, j int) bool {
		return games[i].Name < games[j].Name
	})

	return games
}

// AddCustomGame permite agregar manualmente un juego personalizado
func (bm *BackupManager) AddCustomGame(name, savePath string, patterns []string) error {
	gameID := bm.generateGameID(savePath)

	// Verificar que la ruta existe
	expandedPath := ExpandPath(savePath)
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("la ruta de guardado no existe: %s", expandedPath)
	}

	game := &GameInfo{
		ID:          gameID,
		Name:        name,
		Platform:    "custom",
		SavePaths:   []string{savePath},
		Patterns:    patterns,
		CustomPaths: []string{savePath},
		Metadata:    make(map[string]string),
	}

	bm.DetectedGames[gameID] = game

	if err := bm.updateGameInfo(game); err != nil {
		return err
	}

	log.Printf("Juego personalizado agregado: %s", name)
	return bm.SaveDatabase()
}

// SearchGamesOnPCGW busca juegos en PCGamingWiki
func (bm *BackupManager) SearchGamesOnPCGW(gameName string) ([]GameSearchResult, error) {
	return bm.PCGWClient.SearchGames(gameName)
}

// AddGameFromPCGW agrega un juego desde PCGamingWiki con configuración del usuario
func (bm *BackupManager) AddGameFromPCGW(selection UserGameSelection) error {
	gameID := bm.generateGameID(selection.Name)

	// Crear GameInfo desde la selección
	game := &GameInfo{
		ID:          gameID,
		Name:        selection.Name,
		Platform:    "pcgw", // PCGamingWiki source
		SavePaths:   []string{},
		Patterns:    SaveFilePatterns,
		CustomPaths: []string{},
		Metadata:    make(map[string]string),
	}

	// Si el usuario seleccionó un juego específico de PCGW
	if selection.SelectedGame != nil {
		game.Metadata["pcgw_page_id"] = selection.SelectedGame.PageID
		game.Metadata["steam_app_id"] = selection.SelectedGame.SteamAppID
		game.Metadata["release_date"] = selection.SelectedGame.ReleaseDate
		game.Metadata["cover_url"] = selection.SelectedGame.CoverURL

		// Usar las rutas de guardado de PCGW
		for _, path := range selection.SelectedGame.SavePaths {
			expandedPath := ExpandPath(path)
			game.SavePaths = append(game.SavePaths, expandedPath)
		}
	}

	// Si el usuario especificó una ruta personalizada
	if selection.CustomPath != "" {
		expandedPath := ExpandPath(selection.CustomPath)
		game.SavePaths = append(game.SavePaths, expandedPath)
		game.CustomPaths = append(game.CustomPaths, expandedPath)
	}

	// Validar que al menos una ruta existe
	pathExists := false
	for _, path := range game.SavePaths {
		if _, err := os.Stat(path); err == nil {
			pathExists = true
			break
		}
	}

	if !pathExists {
		return fmt.Errorf("ninguna de las rutas de guardado especificadas existe")
	}

	// Agregar al manager
	bm.DetectedGames[gameID] = game

	// Actualizar información del juego
	if err := bm.updateGameInfo(game); err != nil {
		log.Printf("Error actualizando info del juego %s: %v", gameID, err)
	}

	log.Printf("Juego agregado desde PCGamingWiki: %s", selection.Name)
	return bm.SaveDatabase()
}

// GetDefaultBackupPath devuelve la ruta por defecto para backups del usuario
func (bm *BackupManager) GetDefaultBackupPath() string {
	return bm.Config.BackupDir
}

// SetBackupPath permite al usuario cambiar la ruta de backup
func (bm *BackupManager) SetBackupPath(newPath string) error {
	expandedPath := ExpandPath(newPath)

	// Crear el directorio si no existe
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("error creando directorio de backup: %v", err)
	}

	// Verificar que se puede escribir
	testFile := filepath.Join(expandedPath, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("no se puede escribir en el directorio especificado: %v", err)
	}
	os.Remove(testFile)

	bm.Config.BackupDir = expandedPath
	return nil
}

// ValidateGamePaths valida que las rutas de un juego existen
func (bm *BackupManager) ValidateGamePaths(gameID string) ([]string, []string) {
	game, exists := bm.DetectedGames[gameID]
	if !exists {
		return []string{}, []string{}
	}

	var validPaths, invalidPaths []string

	for _, path := range game.SavePaths {
		expandedPath := ExpandPath(path)
		if _, err := os.Stat(expandedPath); err == nil {
			validPaths = append(validPaths, expandedPath)
		} else {
			invalidPaths = append(invalidPaths, expandedPath)
		}
	}

	return validPaths, invalidPaths
}
