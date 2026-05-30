package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DocsDir  string
	CacheDir string
	DBPath   string
	Theme    string
}

func Default() Config {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "help")
	return Config{
		DocsDir:  filepath.Join(home, "wonderinstruments", "help", "docs"),
		CacheDir: cacheDir,
		DBPath:   filepath.Join(cacheDir, "index.db"),
		Theme:    "default-dark",
	}
}

func (c Config) EnsureCacheDir() error {
	return os.MkdirAll(c.CacheDir, 0755)
}
