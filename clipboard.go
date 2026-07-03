package main

import "github.com/atotto/clipboard"

func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}
