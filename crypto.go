package main

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	seedSize       = 32
	saltSize       = 32
	keySize        = 32
	vaultAAD       = "acctpass:vault:v1"
	defaultMemory  = 64 * 1024
	defaultTime    = 3
	defaultThreads = 4
)

type Argon2idParams struct {
	MemoryKiB uint32 `json:"memory_kib"`
	Time      uint32 `json:"time"`
	Threads   uint8  `json:"threads"`
}

func randomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return buf, nil
}

func deriveEncryptionKey(password, salt []byte, params Argon2idParams) []byte {
	return argon2.IDKey(password, salt, params.Time, params.MemoryKiB, params.Threads, keySize)
}

func encryptSeed(key, seed []byte) (nonce []byte, ciphertext []byte, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize XChaCha20-Poly1305: %w", err)
	}
	nonce, err = randomBytes(chacha20poly1305.NonceSizeX)
	if err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, seed, []byte(vaultAAD)), nil
}

func decryptSeed(key, nonce, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize XChaCha20-Poly1305: %w", err)
	}
	seed, err := aead.Open(nil, nonce, ciphertext, []byte(vaultAAD))
	if err != nil {
		return nil, err
	}
	if len(seed) != seedSize {
		return nil, fmt.Errorf("decrypted seed has invalid length")
	}
	return seed, nil
}
