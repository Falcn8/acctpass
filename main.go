package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const appName = "acctpass"

type passwordReader func(prompt string) ([]byte, error)
type clipboardWriter func(text string) error
type confirmationReader func(prompt string) (bool, error)

type cliConfig struct {
	args               []string
	stdout             io.Writer
	stderr             io.Writer
	passwordReader     passwordReader
	confirmationReader confirmationReader
	clipboard          clipboardWriter
	vaultPathFunc      func() (string, error)
}

func main() {
	cfg := cliConfig{
		args:               os.Args[1:],
		stdout:             os.Stdout,
		stderr:             os.Stderr,
		passwordReader:     readPassword,
		confirmationReader: readConfirmation,
		clipboard:          copyToClipboard,
		vaultPathFunc:      VaultPath,
	}
	if err := runCLI(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	return password, nil
}

func readConfirmation(prompt string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(line), "yes"), nil
}

func runCLI(cfg cliConfig) error {
	if cfg.stdout == nil {
		cfg.stdout = io.Discard
	}
	if cfg.stderr == nil {
		cfg.stderr = io.Discard
	}
	if cfg.passwordReader == nil {
		cfg.passwordReader = readPassword
	}
	if cfg.confirmationReader == nil {
		cfg.confirmationReader = readConfirmation
	}
	if cfg.clipboard == nil {
		cfg.clipboard = copyToClipboard
	}
	if cfg.vaultPathFunc == nil {
		cfg.vaultPathFunc = VaultPath
	}
	if len(cfg.args) == 0 {
		printUsage(cfg.stdout)
		return nil
	}

	switch cfg.args[0] {
	case "init":
		return runInit(cfg, cfg.args[1:])
	case "gen":
		return runGen(cfg, cfg.args[1:])
	case "info":
		return runInfo(cfg, cfg.args[1:])
	case "help", "-h", "--help":
		printUsage(cfg.stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q; run \"acctpass help\"", cfg.args[0])
	}
}

func runInit(cfg cliConfig, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(cfg.stderr)
	force := fs.Bool("force", false, "overwrite existing vault")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("init does not accept positional arguments")
	}

	path, err := cfg.vaultPathFunc()
	if err != nil {
		return err
	}
	if !*force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("vault already exists at %s; use --force to overwrite", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check existing vault: %w", err)
		}
	}

	pw1, err := cfg.passwordReader("Master password: ")
	if err != nil {
		return err
	}
	pw2, err := cfg.passwordReader("Confirm master password: ")
	if err != nil {
		return err
	}
	if string(pw1) != string(pw2) {
		return fmt.Errorf("master passwords do not match")
	}
	if len(pw1) == 0 {
		return fmt.Errorf("master password cannot be empty")
	}
	if warnings := masterPasswordWarnings(pw1); len(warnings) > 0 {
		fmt.Fprintln(cfg.stderr, "Warning: this master password may be weak.")
		for _, warning := range warnings {
			fmt.Fprintf(cfg.stderr, "- %s\n", warning)
		}
		confirmed, err := cfg.confirmationReader("Use this master password anyway? Type \"yes\" to continue: ")
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("master password rejected")
		}
	}

	vault, err := NewVault(pw1)
	if err != nil {
		return err
	}
	if err := SaveVault(path, vault); err != nil {
		return err
	}
	fmt.Fprintf(cfg.stdout, "Vault created at %s\n", path)
	return nil
}

func masterPasswordWarnings(password []byte) []string {
	var warnings []string
	if len(password) < 16 {
		warnings = append(warnings, "it is shorter than 16 characters")
	}
	lower := strings.ToLower(string(password))
	commonParts := []string{"password", "master", "acctpass", "admin", "letmein", "qwerty"}
	for _, part := range commonParts {
		if strings.Contains(lower, part) {
			warnings = append(warnings, "it contains a common password word")
			break
		}
	}
	return warnings
}

func runGen(cfg cliConfig, args []string) error {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(cfg.stderr)
	platform := fs.String("platform", "", "platform or service name")
	email := fs.String("email", "", "account email address")
	counter := fs.Int("counter", 1, "password counter/version")
	length := fs.Int("length", defaultPasswordLength, "password length")
	noSymbols := fs.Bool("no-symbols", false, "exclude symbols")
	allowedSymbols := fs.String("allowed-symbols", "", "use only these supported symbols")
	excludeSymbols := fs.String("exclude-symbols", "", "exclude these supported symbols")
	printPassword := fs.Bool("print", false, "print password instead of copying to clipboard")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("gen does not accept positional arguments")
	}
	if *platform == "" {
		return fmt.Errorf("missing required flag --platform")
	}
	if *email == "" {
		return fmt.Errorf("missing required flag --email")
	}
	if *counter < 1 {
		return fmt.Errorf("counter must be at least 1")
	}
	if *length < minPasswordLength {
		return fmt.Errorf("length must be at least %d", minPasswordLength)
	}
	setFlags := make(map[string]bool)
	fs.Visit(func(flag *flag.Flag) {
		setFlags[flag.Name] = true
	})
	if *noSymbols && (setFlags["allowed-symbols"] || setFlags["exclude-symbols"]) {
		return fmt.Errorf("--no-symbols cannot be combined with --allowed-symbols or --exclude-symbols")
	}
	if setFlags["allowed-symbols"] && setFlags["exclude-symbols"] {
		return fmt.Errorf("--allowed-symbols and --exclude-symbols cannot be combined")
	}
	customSymbols := ""
	if setFlags["allowed-symbols"] {
		var err error
		customSymbols, err = normalizeSymbolSubset(*allowedSymbols)
		if err != nil {
			return fmt.Errorf("invalid --allowed-symbols: %w", err)
		}
		if customSymbols == "" {
			return fmt.Errorf("--allowed-symbols must contain at least one supported symbol")
		}
	}
	if setFlags["exclude-symbols"] {
		var err error
		customSymbols, err = allowedSymbolsAfterExcluding(*excludeSymbols)
		if err != nil {
			return fmt.Errorf("invalid --exclude-symbols: %w", err)
		}
		if customSymbols == "" {
			return fmt.Errorf("--exclude-symbols cannot exclude every supported symbol; use --no-symbols instead")
		}
	}
	if warnings := generatedPasswordWarnings(*length, !*noSymbols); len(warnings) > 0 {
		fmt.Fprintln(cfg.stderr, "Warning: this generated password may be weak.")
		for _, warning := range warnings {
			fmt.Fprintf(cfg.stderr, "- %s\n", warning)
		}
	}

	path, err := cfg.vaultPathFunc()
	if err != nil {
		return err
	}
	vault, err := LoadVault(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("vault not found at %s; run \"acctpass init\" first", path)
		}
		return err
	}
	masterPassword, err := cfg.passwordReader("Master password: ")
	if err != nil {
		return err
	}
	seed, err := vault.DecryptSeed(masterPassword)
	if err != nil {
		return fmt.Errorf("could not decrypt vault; wrong master password or corrupted vault")
	}

	password, err := GeneratePassword(seed, PasswordOptions{
		Platform:       *platform,
		Email:          *email,
		Counter:        *counter,
		Length:         *length,
		Symbols:        !*noSymbols,
		AllowedSymbols: customSymbols,
	})
	if err != nil {
		return err
	}
	if *printPassword {
		fmt.Fprintln(cfg.stdout, password)
		return nil
	}
	if err := cfg.clipboard(password); err != nil {
		return fmt.Errorf("copy password to clipboard: %w; rerun with --print if you need to display it", err)
	}
	fmt.Fprintln(cfg.stdout, "Password copied to clipboard.")
	return nil
}

func runInfo(cfg cliConfig, args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(cfg.stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("info does not accept positional arguments")
	}
	path, err := cfg.vaultPathFunc()
	if err != nil {
		return err
	}
	vault, err := LoadVault(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("vault not found at %s; run \"acctpass init\" first", path)
		}
		return err
	}

	fmt.Fprintf(cfg.stdout, "Vault path: %s\n", path)
	fmt.Fprintf(cfg.stdout, "Created at: %s\n", vault.CreatedAt)
	fmt.Fprintf(cfg.stdout, "Vault version: %d\n", vault.Version)
	fmt.Fprintf(cfg.stdout, "KDF: %s\n", vault.KDF.Name)
	fmt.Fprintf(cfg.stdout, "Argon2id memory: %d KiB\n", vault.KDF.MemoryKiB)
	fmt.Fprintf(cfg.stdout, "Argon2id time: %d\n", vault.KDF.Time)
	fmt.Fprintf(cfg.stdout, "Argon2id threads: %d\n", vault.KDF.Threads)
	fmt.Fprintf(cfg.stdout, "Cipher: %s\n", vault.Cipher.Name)
	return nil
}

func generatedPasswordWarnings(length int, symbols bool) []string {
	var warnings []string
	if length < warnPasswordLength {
		warnings = append(warnings, fmt.Sprintf("it is shorter than %d characters; %d is recommended", warnPasswordLength, defaultPasswordLength))
	}
	if !symbols {
		warnings = append(warnings, "symbols are disabled")
	}
	return warnings
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `acctpass - deterministic offline account password generator

Usage:
  acctpass init [--force]
  acctpass gen --platform <name> --email <email> [--counter <n>] [--length <n>] [--no-symbols | --allowed-symbols <symbols> | --exclude-symbols <symbols>] [--print]
  acctpass info
  acctpass help

Examples:
  acctpass init
  acctpass gen --platform github --email alice@example.com
  acctpass gen --platform github --email alice@example.com --counter 2 --print
  acctpass gen --platform bank --email alice@example.com --length 32 --no-symbols
  acctpass gen --platform legacy --email alice@example.com --allowed-symbols '!@#'
  acctpass gen --platform forum --email alice@example.com --exclude-symbols '[]{}'

Defaults:
  counter = 1
  length = `+strconv.Itoa(defaultPasswordLength)+`
  generated passwords are copied to the clipboard unless --print is passed
`)
}
