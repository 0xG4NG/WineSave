package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PCGamingWiki API structures
type PCGWSearchResult struct {
	Query struct {
		Cargoquery []struct {
			Title struct {
				Page     string `json:"Page"`
				PageID   string `json:"PageID"`
				AppID    string `json:"Steam AppID"`
				Released string `json:"Released"`
				Cover    string `json:"Cover URL"`
			} `json:"title"`
		} `json:"cargoquery"`
	} `json:"query"`
}

type PCGWGameData struct {
	Parse struct {
		Wikitext struct {
			Content string `json:"*"`
		} `json:"wikitext"`
	} `json:"parse"`
}

type GameSearchResult struct {
	Name        string   `json:"name"`
	PageID      string   `json:"page_id"`
	SteamAppID  string   `json:"steam_app_id"`
	ReleaseDate string   `json:"release_date"`
	CoverURL    string   `json:"cover_url"`
	SavePaths   []string `json:"save_paths"`
}

// PCGamingWiki API client
type PCGWClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPCGWClient creates a new PCGamingWiki API client
func NewPCGWClient() *PCGWClient {
	return &PCGWClient{
		baseURL:    "https://www.pcgamingwiki.com/w/api.php",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SearchGames busca juegos en PCGamingWiki por nombre y obtiene automáticamente las rutas de guardado
func (c *PCGWClient) SearchGames(gameName string) ([]GameSearchResult, error) {
	// Escape the game name for URL
	escapedName := url.QueryEscape(gameName)

	// Construir URL de búsqueda
	searchURL := fmt.Sprintf("%s?action=cargoquery&tables=Infobox_game&fields=Infobox_game._pageName=Page,Infobox_game._pageID=PageID,Infobox_game.Steam_AppID,Infobox_game.Released,Infobox_game.Cover_URL&where=Infobox_game._pageName LIKE \"%%%s%%\"&limit=10&format=json",
		c.baseURL, escapedName)

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result PCGWSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var games []GameSearchResult
	for _, item := range result.Query.Cargoquery {
		game := GameSearchResult{
			Name:        item.Title.Page,
			PageID:      item.Title.PageID,
			SteamAppID:  item.Title.AppID,
			ReleaseDate: item.Title.Released,
			CoverURL:    item.Title.Cover,
		}

		// Obtener automáticamente las rutas de guardado para cada juego
		if savePaths, err := c.GetGameSaveData(game.PageID); err == nil && len(savePaths) > 0 {
			game.SavePaths = savePaths
		}

		games = append(games, game)
	}

	return games, nil
}

// GetGameSaveData obtiene los datos de guardado de un juego específico
func (c *PCGWClient) GetGameSaveData(pageID string) ([]string, error) {
	// Get the wikitext content
	wikitextURL := fmt.Sprintf("%s?action=parse&format=json&pageid=%s&prop=wikitext", c.baseURL, pageID)

	resp, err := c.httpClient.Get(wikitextURL)
	if err != nil {
		return nil, fmt.Errorf("error getting wikitext: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading wikitext response: %v", err)
	}

	var result PCGWGameData
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing wikitext JSON: %v", err)
	}

	// Parse the wikitext to extract save data locations
	return c.parseSaveDataFromWikitext(result.Parse.Wikitext.Content), nil
}

// parseSaveDataFromWikitext extrae las rutas de guardado del wikitext
func (c *PCGWClient) parseSaveDataFromWikitext(wikitext string) []string {
	var savePaths []string

	// Look for Game data/saves sections
	lines := strings.Split(wikitext, "\n")
	inSaveSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect save data section
		if strings.Contains(line, "{{Game data/saves") {
			inSaveSection = true
			continue
		}

		// End of section
		if inSaveSection && strings.HasPrefix(line, "}}") {
			inSaveSection = false
			continue
		}

		// Extract paths from save section
		if inSaveSection && strings.Contains(line, "{{P|") {
			paths := c.extractPathsFromLine(line)
			savePaths = append(savePaths, paths...)
		}
	}

	// Also look for common patterns outside sections
	commonPatterns := []string{
		"{{P|userprofile}}\\Documents\\My Games\\",
		"{{P|appdata}}\\",
		"{{P|localappdata}}\\",
		"{{P|userprofile}}\\Saved Games\\",
	}

	for _, pattern := range commonPatterns {
		if strings.Contains(wikitext, pattern) {
			// Extract the full path
			if path := c.extractFullPath(wikitext, pattern); path != "" {
				savePaths = append(savePaths, path)
			}
		}
	}

	return c.cleanAndDeduplicatePaths(savePaths)
}

// extractPathsFromLine extrae rutas de una línea específica
func (c *PCGWClient) extractPathsFromLine(line string) []string {
	var paths []string

	// Convert template variables to actual paths
	conversions := map[string]string{
		"{{P|userprofile}}":  "%USERPROFILE%",
		"{{P|appdata}}":      "%APPDATA%",
		"{{P|localappdata}}": "%LOCALAPPDATA%",
		"{{P|game}}":         "%GAME_DIR%",
		"{{P|documents}}":    "%USERPROFILE%\\Documents",
	}

	// Apply conversions
	convertedLine := line
	for template, replacement := range conversions {
		convertedLine = strings.ReplaceAll(convertedLine, template, replacement)
	}

	// Extract paths between pipes
	parts := strings.Split(convertedLine, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "%") && !strings.Contains(part, "{{") {
			paths = append(paths, part)
		}
	}

	return paths
}

// extractFullPath extrae la ruta completa basada en un patrón
func (c *PCGWClient) extractFullPath(wikitext, pattern string) string {
	// This is a simplified extraction - in a real implementation,
	// you'd want more sophisticated parsing
	index := strings.Index(wikitext, pattern)
	if index == -1 {
		return ""
	}

	// Extract a reasonable substring around the pattern
	start := index
	end := index + len(pattern) + 100
	if end > len(wikitext) {
		end = len(wikitext)
	}

	substring := wikitext[start:end]
	lines := strings.Split(substring, "\n")
	if len(lines) > 0 {
		return c.cleanPath(lines[0])
	}

	return ""
}

// cleanPath limpia y normaliza una ruta
func (c *PCGWClient) cleanPath(path string) string {
	// Remove wiki markup
	path = strings.ReplaceAll(path, "{{P|userprofile}}", "%USERPROFILE%")
	path = strings.ReplaceAll(path, "{{P|appdata}}", "%APPDATA%")
	path = strings.ReplaceAll(path, "{{P|localappdata}}", "%LOCALAPPDATA%")
	path = strings.ReplaceAll(path, "{{P|documents}}", "%USERPROFILE%\\Documents")
	path = strings.ReplaceAll(path, "{{P|game}}", "%GAME_DIR%")

	// Remove remaining wiki markup
	path = strings.ReplaceAll(path, "{{", "")
	path = strings.ReplaceAll(path, "}}", "")
	path = strings.ReplaceAll(path, "|", "")

	// Clean up
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"'")

	return path
}

// cleanAndDeduplicatePaths limpia y elimina duplicados
func (c *PCGWClient) cleanAndDeduplicatePaths(paths []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, path := range paths {
		cleaned := c.cleanPath(path)
		if cleaned != "" && !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, cleaned)
		}
	}

	return result
}

// SearchGameBySteamID busca un juego por Steam App ID
func (c *PCGWClient) SearchGameBySteamID(steamAppID string) (*GameSearchResult, error) {
	searchURL := fmt.Sprintf("%s?action=cargoquery&tables=Infobox_game&fields=Infobox_game._pageName=Page,Infobox_game._pageID=PageID,Infobox_game.Steam_AppID,Infobox_game.Released,Infobox_game.Cover_URL&where=Infobox_game.Steam_AppID HOLDS \"%s\"&format=json",
		c.baseURL, steamAppID)

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result PCGWSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	if len(result.Query.Cargoquery) == 0 {
		return nil, fmt.Errorf("game not found")
	}

	item := result.Query.Cargoquery[0]
	game := &GameSearchResult{
		Name:        item.Title.Page,
		PageID:      item.Title.PageID,
		SteamAppID:  item.Title.AppID,
		ReleaseDate: item.Title.Released,
		CoverURL:    item.Title.Cover,
	}

	// Get save data
	savePaths, err := c.GetGameSaveData(game.PageID)
	if err == nil {
		game.SavePaths = savePaths
	}

	return game, nil
}
