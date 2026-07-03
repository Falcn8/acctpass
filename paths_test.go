package main

import (
	"path/filepath"
	"testing"
)

func TestVaultPathFromConfigDir(t *testing.T) {
	base := filepath.Join("tmp", "config")
	got := VaultPathFromConfigDir(base)
	want := filepath.Join(base, appName, "vault.json")
	if got != want {
		t.Fatalf("VaultPathFromConfigDir = %q, want %q", got, want)
	}
}
