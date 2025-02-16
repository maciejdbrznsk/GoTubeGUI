//go:build windows
// +build windows

package main

import (
	"golang.org/x/sys/windows/registry"
)

func getDownloadPath() (string, error) {
	const key = `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Shell Folders`
	const value = "{374DE290-123F-4565-9164-39C4925E467B}"
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.READ)
	if err != nil {
		return "", err
	}
	defer k.Close()
	downloadPath, _, err := k.GetStringValue(value)
	if err != nil {
		return "", err
	}
	return downloadPath, nil
}
