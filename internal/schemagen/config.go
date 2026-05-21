package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	BundleFormatVersion int             `json:"bundleFormatVersion"`
	Releases            []ReleaseConfig `json:"releases"`
}

type ReleaseConfig struct {
	Tag             string `json:"tag"`
	Commit          string `json:"commit"`
	RawCRDDir       string `json:"rawCRDDir"`
	CrossplaneGoMod string `json:"crossplaneGoMod"`
}

func LoadConfigFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	base := filepath.Dir(path)
	for i := range cfg.Releases {
		cfg.Releases[i].RawCRDDir = resolveConfigPath(base, cfg.Releases[i].RawCRDDir)
		cfg.Releases[i].CrossplaneGoMod = resolveConfigPath(base, cfg.Releases[i].CrossplaneGoMod)
	}
	return cfg, nil
}

func resolveConfigPath(base, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	resolved := filepath.Join(base, path)
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return resolved
}
