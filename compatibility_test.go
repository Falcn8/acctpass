package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func TestCompatibilityVectors(t *testing.T) {
	data, err := os.ReadFile("compatibility/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		SeedB64        string `json:"seed_b64"`
		MasterPassword string `json:"master_password"`
		Vault          Vault  `json:"vault"`
		Passwords      []struct {
			PasswordOptions
			Expected string `json:"expected"`
		} `json:"passwords"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	seed, err := base64.StdEncoding.DecodeString(raw.SeedB64)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := raw.Vault.DecryptSeed([]byte(raw.MasterPassword))
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(seed) {
		t.Fatal("fixture vault did not decrypt to the shared seed")
	}
	for _, vector := range raw.Passwords {
		got, err := GeneratePassword(seed, vector.PasswordOptions)
		if err != nil {
			t.Fatal(err)
		}
		if got != vector.Expected {
			t.Fatalf("password = %q, want %q", got, vector.Expected)
		}
	}
}
