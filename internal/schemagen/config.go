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
		rawCRDDir, err := resolveConfigPath(base, cfg.Releases[i].RawCRDDir)
		if err != nil {
			return Config{}, fmt.Errorf("release %s rawCRDDir: %w", cfg.Releases[i].Tag, err)
		}
		crossplaneGoMod, err := resolveConfigPath(base, cfg.Releases[i].CrossplaneGoMod)
		if err != nil {
			return Config{}, fmt.Errorf("release %s crossplaneGoMod: %w", cfg.Releases[i].Tag, err)
		}
		cfg.Releases[i].RawCRDDir = rawCRDDir
		cfg.Releases[i].CrossplaneGoMod = crossplaneGoMod
	}
	return cfg, nil
}

func resolveConfigPath(base, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	resolved := path
	if !filepath.IsAbs(path) {
		resolved = filepath.Join(base, path)
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("%s: %w", resolved, err)
	}
	return resolved, nil
}
