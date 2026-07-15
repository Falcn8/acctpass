package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
)

const (
	defaultPasswordLength = 24
	minPasswordLength     = 1
	warnPasswordLength    = 12
	lowerAlphabet         = "abcdefghijklmnopqrstuvwxyz"
	upperAlphabet         = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitAlphabet         = "0123456789"
	symbolAlphabet        = "!@#$%^&*()-_=+[]{}?"
)

const (
	defaultAlphabet   = lowerAlphabet + upperAlphabet + digitAlphabet + symbolAlphabet
	noSymbolsAlphabet = lowerAlphabet + upperAlphabet + digitAlphabet
)

type PasswordOptions struct {
	Platform string
	Email    string
	Counter  int
	Length   int
	Symbols  bool
}

func GeneratePassword(seed []byte, opts PasswordOptions) (string, error) {
	if len(seed) != seedSize {
		return "", fmt.Errorf("seed must be %d bytes", seedSize)
	}
	if opts.Counter < 1 {
		return "", fmt.Errorf("counter must be at least 1")
	}
	if opts.Length < minPasswordLength {
		return "", fmt.Errorf("length must be at least %d", minPasswordLength)
	}
	platform := normalizeIdentity(opts.Platform)
	email := normalizeIdentity(opts.Email)
	if platform == "" {
		return "", fmt.Errorf("platform cannot be empty")
	}
	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	alphabet := defaultAlphabet
	if !opts.Symbols {
		alphabet = noSymbolsAlphabet
	}
	baseContext := derivationContext(platform, email, opts.Counter, opts.Length, opts.Symbols)
	enforceCharacterClasses := opts.Length >= requiredCharacterClassCount(opts.Symbols)
	for attempt := 0; attempt < 10_000; attempt++ {
		context := baseContext
		if attempt > 0 {
			context = fmt.Sprintf("%s|attempt=%d", baseContext, attempt)
		}
		password, err := passwordFromContext(seed, context, alphabet, opts.Length)
		if err != nil {
			return "", err
		}
		if !enforceCharacterClasses || satisfiesPasswordRules(password, opts.Symbols) {
			return password, nil
		}
	}
	return "", fmt.Errorf("could not generate a password satisfying character-class rules")
}

func normalizeIdentity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func derivationContext(platform, email string, counter, length int, symbols bool) string {
	return fmt.Sprintf("acctpass:v1|platform=%s|email=%s|counter=%d|length=%d|symbols=%t", platform, email, counter, length, symbols)
}

func passwordFromContext(seed []byte, context, alphabet string, length int) (string, error) {
	var out strings.Builder
	out.Grow(length)
	block := 0
	for out.Len() < length {
		input := context
		if block > 0 {
			input = fmt.Sprintf("%s|block=%d", context, block)
		}
		digest := hmacSHA256(seed, []byte(input))
		part := rejectionSampleBytes(digest, alphabet, length-out.Len())
		out.WriteString(part)
		block++
	}
	return out.String(), nil
}

func hmacSHA256(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func rejectionSampleBytes(randomBytes []byte, alphabet string, needed int) string {
	if needed <= 0 || len(alphabet) == 0 {
		return ""
	}
	limit := byte(256 - (256 % len(alphabet)))
	var out strings.Builder
	out.Grow(needed)
	for _, b := range randomBytes {
		if out.Len() == needed {
			break
		}
		if b >= limit {
			continue
		}
		out.WriteByte(alphabet[int(b)%len(alphabet)])
	}
	return out.String()
}

func satisfiesPasswordRules(password string, requireSymbol bool) bool {
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := !requireSymbol
	for _, r := range password {
		switch {
		case strings.ContainsRune(lowerAlphabet, r):
			hasLower = true
		case strings.ContainsRune(upperAlphabet, r):
			hasUpper = true
		case strings.ContainsRune(digitAlphabet, r):
			hasDigit = true
		case strings.ContainsRune(symbolAlphabet, r):
			hasSymbol = true
		}
	}
	return hasLower && hasUpper && hasDigit && hasSymbol
}

func requiredCharacterClassCount(requireSymbol bool) int {
	if requireSymbol {
		return 4
	}
	return 3
}
