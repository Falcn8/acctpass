package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testCLI(args ...string) cliConfig {
	return cliConfig{
		args:   args,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		passwordReader: func(prompt string) ([]byte, error) {
			return []byte("master"), nil
		},
		confirmationReader: func(prompt string) (bool, error) {
			return true, nil
		},
		clipboard: func(text string) error {
			return nil
		},
		vaultPathFunc: func() (string, error) {
			return filepath.Join(os.TempDir(), "acctpass-test-vault.json"), nil
		},
	}
}

func TestCLIInitWeakMasterPasswordCanBeRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	cfg := testCLI("init")
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	cfg.passwordReader = func(prompt string) ([]byte, error) { return []byte("master"), nil }
	cfg.confirmationReader = func(prompt string) (bool, error) { return false, nil }

	err := runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "master password rejected") {
		t.Fatalf("error = %v, want rejection", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("vault was created despite rejection; stat error = %v", statErr)
	}
}

func TestCLIInitWeakMasterPasswordCanBeAccepted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	stderr := &bytes.Buffer{}
	cfg := testCLI("init")
	cfg.stderr = stderr
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	cfg.passwordReader = func(prompt string) ([]byte, error) { return []byte("master"), nil }
	cfg.confirmationReader = func(prompt string) (bool, error) { return true, nil }

	if err := runCLI(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "may be weak") {
		t.Fatalf("stderr = %q, want weak-password warning", stderr.String())
	}
}

func TestCLIInitStrongMasterPasswordSkipsWeakConfirmation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	confirmationCalled := false
	cfg := testCLI("init")
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	cfg.passwordReader = func(prompt string) ([]byte, error) {
		return []byte("long unique passphrase"), nil
	}
	cfg.confirmationReader = func(prompt string) (bool, error) {
		confirmationCalled = true
		return false, nil
	}

	if err := runCLI(cfg); err != nil {
		t.Fatal(err)
	}
	if confirmationCalled {
		t.Fatal("confirmation was requested for a stronger master password")
	}
}

func TestCLIMissingVaultError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com")
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	err := runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "vault not found") {
		t.Fatalf("error = %v, want missing vault error", err)
	}
}

func TestCLIWrongPasswordError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	vault, err := newVaultWithSeed([]byte("right"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com")
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	cfg.passwordReader = func(prompt string) ([]byte, error) { return []byte("wrong"), nil }
	err = runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "wrong master password") {
		t.Fatalf("error = %v, want wrong password error", err)
	}
}

func TestCLIInvalidLengthError(t *testing.T) {
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com", "--length", "8")
	err := runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "length must be at least") {
		t.Fatalf("error = %v, want invalid length error", err)
	}
}

func TestCLIMissingRequiredFlags(t *testing.T) {
	cfg := testCLI("gen", "--email", "alice@example.com")
	err := runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "missing required flag --platform") {
		t.Fatalf("error = %v, want missing platform error", err)
	}

	cfg = testCLI("gen", "--platform", "github")
	err = runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "missing required flag --email") {
		t.Fatalf("error = %v, want missing email error", err)
	}
}

func TestCLIGenPrintDoesNotUseClipboard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	clipboardCalled := false
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com", "--print")
	cfg.stdout = stdout
	cfg.vaultPathFunc = func() (string, error) { return path, nil }
	cfg.clipboard = func(text string) error {
		clipboardCalled = true
		return nil
	}
	if err := runCLI(cfg); err != nil {
		t.Fatal(err)
	}
	if clipboardCalled {
		t.Fatal("clipboard was called with --print")
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatal("expected printed password")
	}
}
