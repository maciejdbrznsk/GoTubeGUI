//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"
)

func getDownloadPath() (string, error) {
	downloadPath := fmt.Sprintf("%s/Downloads", os.Getenv("HOME"))
	return downloadPath, nil
}
