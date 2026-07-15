package main

import (
	"strings"
	"testing"
)

func testSeed() []byte {
	seed := make([]byte, seedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return seed
}

func defaultTestOptions() PasswordOptions {
	return PasswordOptions{
		Platform: "GitHub",
		Email:    "Alice@Example.com",
		Counter:  1,
		Length:   defaultPasswordLength,
		Symbols:  true,
	}
}

func TestGeneratePasswordDeterministic(t *testing.T) {
	seed := testSeed()
	opts := defaultTestOptions()
	first, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same seed and inputs produced different passwords: %q != %q", first, second)
	}
}

func TestGeneratePasswordDifferentPlatform(t *testing.T) {
	seed := testSeed()
	opts := defaultTestOptions()
	first, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Platform = "GitLab"
	second, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different platform produced same password")
	}
}

func TestGeneratePasswordDifferentEmail(t *testing.T) {
	seed := testSeed()
	opts := defaultTestOptions()
	first, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Email = "bob@example.com"
	second, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different email produced same password")
	}
}

func TestGeneratePasswordDifferentCounter(t *testing.T) {
	seed := testSeed()
	opts := defaultTestOptions()
	first, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Counter = 2
	second, err := GeneratePassword(seed, opts)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different counter produced same password")
	}
}

func TestGeneratePasswordLength(t *testing.T) {
	opts := defaultTestOptions()
	opts.Length = 32
	password, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(password) != opts.Length {
		t.Fatalf("password length = %d, want %d", len(password), opts.Length)
	}
}

func TestGeneratePasswordAllowsShortLength(t *testing.T) {
	opts := defaultTestOptions()
	opts.Length = 1
	password, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(password) != opts.Length {
		t.Fatalf("password length = %d, want %d", len(password), opts.Length)
	}
}

func TestGeneratePasswordRejectsNonPositiveLength(t *testing.T) {
	opts := defaultTestOptions()
	opts.Length = 0
	_, err := GeneratePassword(testSeed(), opts)
	if err == nil || !strings.Contains(err.Error(), "length must be at least") {
		t.Fatalf("error = %v, want invalid length error", err)
	}
}

func TestDefaultPasswordContainsRequiredClasses(t *testing.T) {
	password, err := GeneratePassword(testSeed(), defaultTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !satisfiesPasswordRules(password, symbolAlphabet) {
		t.Fatalf("password does not satisfy required classes: %q", password)
	}
}

func TestNoSymbolsPasswordContainsNoSymbols(t *testing.T) {
	opts := defaultTestOptions()
	opts.Symbols = false
	password, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(password, symbolAlphabet) {
		t.Fatalf("no-symbols password contains a symbol: %q", password)
	}
	if !satisfiesPasswordRules(password, "") {
		t.Fatalf("no-symbols password does not satisfy required classes: %q", password)
	}
}

func TestGeneratePasswordUsesOnlyAllowedSymbols(t *testing.T) {
	opts := defaultTestOptions()
	opts.AllowedSymbols = "@!@"
	password, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(password, "#$%^&*()-_=+[]{}?") {
		t.Fatalf("password contains a symbol outside the allowed set: %q", password)
	}
	if !strings.ContainsAny(password, "!@") {
		t.Fatalf("password does not contain a required allowed symbol: %q", password)
	}
}

func TestGeneratePasswordNormalizesAllowedSymbolOrder(t *testing.T) {
	opts := defaultTestOptions()
	opts.AllowedSymbols = "@!"
	first, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.AllowedSymbols = "!@"
	second, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("equivalent allowed-symbol sets produced different passwords: %q != %q", first, second)
	}
}

func TestGeneratePasswordFullAllowedSetPreservesDefaultPassword(t *testing.T) {
	opts := defaultTestOptions()
	defaultPassword, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.AllowedSymbols = symbolAlphabet
	explicitPassword, err := GeneratePassword(testSeed(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if defaultPassword != explicitPassword {
		t.Fatalf("full allowed-symbol set changed the default password: %q != %q", defaultPassword, explicitPassword)
	}
}

func TestGeneratePasswordRejectsUnsupportedAllowedSymbol(t *testing.T) {
	opts := defaultTestOptions()
	opts.AllowedSymbols = "!|"
	_, err := GeneratePassword(testSeed(), opts)
	if err == nil || !strings.Contains(err.Error(), "unsupported symbol") {
		t.Fatalf("error = %v, want unsupported symbol error", err)
	}
}

func TestAllowedSymbolsAfterExcluding(t *testing.T) {
	allowed, err := allowedSymbolsAfterExcluding("{}[]")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(allowed, "{}[]") {
		t.Fatalf("allowed symbols %q include an excluded symbol", allowed)
	}
	if allowed != "!@#$%^&*()-_=+?" {
		t.Fatalf("allowed symbols = %q, want %q", allowed, "!@#$%^&*()-_=+?")
	}
}

func TestRejectionSamplingRejectsOutOfRangeBytes(t *testing.T) {
	got := rejectionSampleBytes([]byte{250, 251, 252, 253, 254, 255, 0, 1}, "0123456789", 2)
	if got != "01" {
		t.Fatalf("rejection sampling result = %q, want %q", got, "01")
	}
}

func TestRejectionSamplingSupportsAlphabetThatDividesByteRange(t *testing.T) {
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@"
	got := rejectionSampleBytes([]byte{0, 63, 64, 255}, alphabet, 4)
	if got != "a@a@" {
		t.Fatalf("rejection sampling result = %q, want %q", got, "a@a@")
	}
}

func TestNormalizeIdentity(t *testing.T) {
	if got := normalizeIdentity("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("normalizeIdentity = %q", got)
	}
}
