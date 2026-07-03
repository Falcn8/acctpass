package main

import (
	"os"
	"path/filepath"
)

func VaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return VaultPathFromConfigDir(configDir), nil
}

func VaultPathFromConfigDir(configDir string) string {
	return filepath.Join(configDir, appName, "vault.json")
}
