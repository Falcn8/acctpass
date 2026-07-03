//go:build !windows
// +build !windows

package main

import "os"

func replaceFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
