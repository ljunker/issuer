// Package config loads the optional user configuration from
// $XDG_CONFIG_HOME/issuer/config.toml (default ~/.config/issuer/config.toml).
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/ljunker/issuer/internal/provider"
)

type Config struct {
	// Pinned project IDs (e.g. "owner/repo") are listed first.
	Pinned []string `toml:"pinned"`
	// DefaultState is the issue state filter on start: open, closed or all.
	DefaultState provider.State `toml:"default_state"`
}

func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "issuer", "config.toml"), nil
}

// Load reads the config file; a missing file yields the defaults.
func Load() (Config, error) {
	cfg := Config{DefaultState: provider.StateOpen}
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, err
	}
	switch cfg.DefaultState {
	case provider.StateOpen, provider.StateClosed, provider.StateAll:
	default:
		cfg.DefaultState = provider.StateOpen
	}
	return cfg, nil
}
