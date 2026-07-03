package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	vaultVersion = 1
	kdfName      = "argon2id"
	cipherName   = "xchacha20poly1305"
)

const (
	maxArgon2MemoryKiB = 1024 * 1024
	maxArgon2Time      = 10
	maxArgon2Threads   = 16
)

type Vault struct {
	Version   int        `json:"version"`
	CreatedAt string     `json:"created_at"`
	KDF       KDFConfig  `json:"kdf"`
	Cipher    CipherData `json:"cipher"`
}

type KDFConfig struct {
	Name      string `json:"name"`
	MemoryKiB uint32 `json:"memory_kib"`
	Time      uint32 `json:"time"`
	Threads   uint8  `json:"threads"`
	SaltB64   string `json:"salt_b64"`
}

type CipherData struct {
	Name          string `json:"name"`
	NonceB64      string `json:"nonce_b64"`
	CiphertextB64 string `json:"ciphertext_b64"`
}

func NewVault(masterPassword []byte) (*Vault, error) {
	seed, err := randomBytes(seedSize)
	if err != nil {
		return nil, err
	}
	params := Argon2idParams{
		MemoryKiB: defaultMemory,
		Time:      defaultTime,
		Threads:   defaultThreads,
	}
	return newVaultWithSeed(masterPassword, seed, params, time.Now().UTC())
}

func newVaultWithSeed(masterPassword, seed []byte, params Argon2idParams, createdAt time.Time) (*Vault, error) {
	if len(seed) != seedSize {
		return nil, fmt.Errorf("seed must be %d bytes", seedSize)
	}
	salt, err := randomBytes(saltSize)
	if err != nil {
		return nil, err
	}
	key := deriveEncryptionKey(masterPassword, salt, params)
	nonce, ciphertext, err := encryptSeed(key, seed)
	if err != nil {
		return nil, err
	}

	return &Vault{
		Version:   vaultVersion,
		CreatedAt: createdAt.UTC().Format(time.RFC3339),
		KDF: KDFConfig{
			Name:      kdfName,
			MemoryKiB: params.MemoryKiB,
			Time:      params.Time,
			Threads:   params.Threads,
			SaltB64:   base64.StdEncoding.EncodeToString(salt),
		},
		Cipher: CipherData{
			Name:          cipherName,
			NonceB64:      base64.StdEncoding.EncodeToString(nonce),
			CiphertextB64: base64.StdEncoding.EncodeToString(ciphertext),
		},
	}, nil
}

func (v *Vault) DecryptSeed(masterPassword []byte) ([]byte, error) {
	if err := v.ValidateMetadata(); err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(v.KDF.SaltB64)
	if err != nil {
		return nil, fmt.Errorf("decode vault salt: %w", err)
	}
	if len(salt) != saltSize {
		return nil, fmt.Errorf("vault salt has invalid length")
	}
	nonce, err := base64.StdEncoding.DecodeString(v.Cipher.NonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode vault nonce: %w", err)
	}
	if len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("vault nonce has invalid length")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(v.Cipher.CiphertextB64)
	if err != nil {
		return nil, fmt.Errorf("decode vault ciphertext: %w", err)
	}
	params := Argon2idParams{
		MemoryKiB: v.KDF.MemoryKiB,
		Time:      v.KDF.Time,
		Threads:   v.KDF.Threads,
	}
	key := deriveEncryptionKey(masterPassword, salt, params)
	return decryptSeed(key, nonce, ciphertext)
}

func (v *Vault) ValidateMetadata() error {
	if v.Version != vaultVersion {
		return fmt.Errorf("unsupported vault version %d", v.Version)
	}
	if v.KDF.Name != kdfName {
		return fmt.Errorf("unsupported KDF %q", v.KDF.Name)
	}
	if v.Cipher.Name != cipherName {
		return fmt.Errorf("unsupported cipher %q", v.Cipher.Name)
	}
	if v.KDF.MemoryKiB == 0 || v.KDF.Time == 0 || v.KDF.Threads == 0 {
		return fmt.Errorf("invalid Argon2id parameters in vault")
	}
	if v.KDF.MemoryKiB > maxArgon2MemoryKiB {
		return fmt.Errorf("Argon2id memory parameter too large in vault")
	}
	if v.KDF.Time > maxArgon2Time {
		return fmt.Errorf("Argon2id time parameter too large in vault")
	}
	if v.KDF.Threads > maxArgon2Threads {
		return fmt.Errorf("Argon2id threads parameter too large in vault")
	}
	return nil
}

func LoadVault(path string) (*Vault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var vault Vault
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, fmt.Errorf("parse vault JSON: %w", err)
	}
	if err := vault.ValidateMetadata(); err != nil {
		return nil, err
	}
	return &vault, nil
}

func SaveVault(path string, vault *Vault) error {
	data, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return fmt.Errorf("encode vault JSON: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	dirPerm := os.FileMode(0o700)
	filePerm := os.FileMode(0o600)
	if runtime.GOOS == "windows" {
		dirPerm = 0o755
		filePerm = 0o600
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if info, err := os.Lstat(dir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to use symlink config directory")
		}
	} else {
		return fmt.Errorf("check config directory: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, dirPerm); err != nil {
			return fmt.Errorf("set config directory permissions: %w", err)
		}
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to overwrite symlink vault path")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check vault path: %w", err)
	}
	if err := writeFileAtomic(path, data, filePerm); err != nil {
		return fmt.Errorf("write vault: %w", err)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vault-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, perm); err != nil {
			return err
		}
		syncDir(dir)
	}
	return nil
}

func syncDir(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	defer f.Close()
	_ = f.Sync()
}
