//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"
)

func getDownloadPath() (string, error) {
	// Dla systemów innych niż Windows, domyślnie ustawiamy katalog Downloads w katalogu domowym.
	downloadPath := fmt.Sprintf("%s/Downloads", os.Getenv("HOME"))
	return downloadPath, nil
}
