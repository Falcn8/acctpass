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

func TestMasterPasswordWarningsAreAlwaysBypassable(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		warning  string
	}{
		{name: "short", password: "short", warning: "shorter than 16"},
		{name: "common word", password: "a-very-long-password-value", warning: "common password word"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vault.json")
			stderr := &bytes.Buffer{}
			confirmationCalled := false
			cfg := testCLI("init")
			cfg.stderr = stderr
			cfg.vaultPathFunc = func() (string, error) { return path, nil }
			cfg.passwordReader = func(prompt string) ([]byte, error) { return []byte(tc.password), nil }
			cfg.confirmationReader = func(prompt string) (bool, error) {
				confirmationCalled = true
				return true, nil
			}

			if err := runCLI(cfg); err != nil {
				t.Fatal(err)
			}
			if !confirmationCalled {
				t.Fatal("warning did not offer confirmation")
			}
			if !strings.Contains(stderr.String(), tc.warning) {
				t.Fatalf("stderr = %q, want warning containing %q", stderr.String(), tc.warning)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("vault was not created after accepting warning: %v", err)
			}
		})
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
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com", "--length", "0")
	err := runCLI(cfg)
	if err == nil || !strings.Contains(err.Error(), "length must be at least") {
		t.Fatalf("error = %v, want invalid length error", err)
	}
}

func TestCLIShortLengthWarnsAndGenerates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com", "--no-symbols", "--length", "8", "--print")
	cfg.stdout = stdout
	cfg.stderr = stderr
	cfg.vaultPathFunc = func() (string, error) { return path, nil }

	if err := runCLI(cfg); err != nil {
		t.Fatal(err)
	}
	password := strings.TrimSpace(stdout.String())
	if len(password) != 8 {
		t.Fatalf("password length = %d, want 8", len(password))
	}
	warning := stderr.String()
	if !strings.Contains(warning, "may be weak") || !strings.Contains(warning, "shorter than 12") || !strings.Contains(warning, "symbols are disabled") {
		t.Fatalf("stderr = %q, want weak generated-password warnings", warning)
	}
}

func TestGeneratedPasswordWarningCases(t *testing.T) {
	testCases := []struct {
		name        string
		length      int
		symbols     bool
		wantWarning string
	}{
		{name: "short", length: 8, symbols: true, wantWarning: "shorter than 12"},
		{name: "no symbols", length: defaultPasswordLength, symbols: false, wantWarning: "symbols are disabled"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			warnings := strings.Join(generatedPasswordWarnings(tc.length, tc.symbols), "\n")
			if !strings.Contains(warnings, tc.wantWarning) {
				t.Fatalf("warnings = %q, want %q", warnings, tc.wantWarning)
			}
		})
	}
}

func TestCLICustomSymbolFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	vault, err := newVaultWithSeed([]byte("master"), testSeed(), fastKDFParams(), time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveVault(path, vault); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name      string
		flag      string
		value     string
		forbidden string
	}{
		{name: "allowed", flag: "--allowed-symbols", value: "!@", forbidden: "#$%^&*()-_=+[]{}?"},
		{name: "excluded", flag: "--exclude-symbols", value: "[]{}", forbidden: "[]{}"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			cfg := testCLI("gen", "--platform", "github", "--email", "alice@example.com", tc.flag, tc.value, "--print")
			cfg.stdout = stdout
			cfg.vaultPathFunc = func() (string, error) { return path, nil }
			if err := runCLI(cfg); err != nil {
				t.Fatal(err)
			}
			password := strings.TrimSpace(stdout.String())
			if strings.ContainsAny(password, tc.forbidden) {
				t.Fatalf("password %q contains a prohibited symbol from %q", password, tc.forbidden)
			}
		})
	}
}

func TestCLIRejectsConflictingOrInvalidSymbolFlags(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "allowed and excluded", args: []string{"--allowed-symbols", "!", "--exclude-symbols", "@"}, wantErr: "cannot be combined"},
		{name: "no symbols and allowed", args: []string{"--no-symbols", "--allowed-symbols", "!"}, wantErr: "cannot be combined"},
		{name: "empty allowed set", args: []string{"--allowed-symbols", ""}, wantErr: "at least one"},
		{name: "unsupported allowed symbol", args: []string{"--allowed-symbols", "!|"}, wantErr: "unsupported symbol"},
		{name: "exclude all symbols", args: []string{"--exclude-symbols", symbolAlphabet}, wantErr: "cannot exclude every"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"gen", "--platform", "github", "--email", "alice@example.com"}, tc.args...)
			err := runCLI(testCLI(args...))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error containing %q", err, tc.wantErr)
			}
		})
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
