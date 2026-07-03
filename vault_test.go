package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fastKDFParams() Argon2idParams {
	return Argon2idParams{
		MemoryKiB: 1024,
		Time:      1,
		Threads:   1,
	}
}

func TestVaultEncryptDecryptRoundTrip(t *testing.T) {
	master := []byte("correct horse battery staple")
	seed := testSeed()
	vault, err := newVaultWithSeed(master, seed, fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	got, err := vault.DecryptSeed(master)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(seed) {
		t.Fatal("decrypted seed did not match original seed")
	}
	if _, err := vault.DecryptSeed([]byte("wrong password")); err == nil {
		t.Fatal("wrong password decrypted vault")
	}
}

func TestSaveLoadVault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config", "acctpass", "vault.json")
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadVault(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Cipher.CiphertextB64 == "" || loaded.Cipher.NonceB64 == "" || loaded.KDF.SaltB64 == "" {
		t.Fatal("loaded vault missing encoded cryptographic fields")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestSaveVaultTightensExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config", "acctpass", "vault.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("vault permissions = %o, want 600", got)
	}
}

func TestSaveVaultRefusesSymlinkPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior requires extra privileges on some Windows setups")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	path := filepath.Join(dir, "vault.json")
	original := []byte("do not overwrite")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}

	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	err = SaveVault(path, vault)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SaveVault error = %v, want symlink refusal", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("symlink target was overwritten: %q", got)
	}
}

func TestValidateMetadataRejectsOversizedKDFParams(t *testing.T) {
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		edit func(*Vault)
	}{
		{
			name: "memory",
			edit: func(v *Vault) {
				v.KDF.MemoryKiB = maxArgon2MemoryKiB + 1
			},
		},
		{
			name: "time",
			edit: func(v *Vault) {
				v.KDF.Time = maxArgon2Time + 1
			},
		},
		{
			name: "threads",
			edit: func(v *Vault) {
				v.KDF.Threads = maxArgon2Threads + 1
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			edited := *vault
			tc.edit(&edited)
			if err := edited.ValidateMetadata(); err == nil {
				t.Fatal("ValidateMetadata accepted oversized KDF parameters")
			}
			if _, err := edited.DecryptSeed([]byte("master")); err == nil {
				t.Fatal("DecryptSeed accepted oversized KDF parameters")
			}
		})
	}
}

func TestDecryptSeedRejectsInvalidNonceLength(t *testing.T) {
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	vault.Cipher.NonceB64 = "AA=="

	_, err = vault.DecryptSeed([]byte("master"))
	if err == nil || !strings.Contains(err.Error(), "nonce") {
		t.Fatalf("DecryptSeed error = %v, want nonce length error", err)
	}
}

func TestLoadVaultMissingFile(t *testing.T) {
	_, err := LoadVault(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadVault error = %v, want os.ErrNotExist", err)
	}
}
