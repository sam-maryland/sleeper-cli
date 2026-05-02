package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sam-maryland/sleeper-cli/pkg/sleeper"
)

// CachedStandings holds the finalized standings and league info for a completed season.
type CachedStandings struct {
	Standings []sleeper.Standing    `json:"standings"`
	League    sleeper.SleeperLeague `json:"league"`
}

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".sleeper", "cache"), nil
}

func cachePath(leagueID string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, leagueID+".json"), nil
}

// Load returns the cached standings for a league, and whether a cache entry existed.
func Load(leagueID string) (CachedStandings, bool, error) {
	path, err := cachePath(leagueID)
	if err != nil {
		return CachedStandings{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return CachedStandings{}, false, nil
	}
	if err != nil {
		return CachedStandings{}, false, fmt.Errorf("reading cache: %w", err)
	}
	var cs CachedStandings
	if err := json.Unmarshal(data, &cs); err != nil {
		return CachedStandings{}, false, fmt.Errorf("parsing cache: %w", err)
	}
	return cs, true, nil
}

// Save writes standings for a completed league to the local cache.
func Save(leagueID string, cs CachedStandings) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	path := filepath.Join(dir, leagueID+".json")
	data, err := json.Marshal(cs)
	if err != nil {
		return fmt.Errorf("serializing cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}
	return nil
}
