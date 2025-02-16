//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func updateYtDlp() error {
	currentVersion := ""
	if data, err := os.ReadFile("info.json"); err == nil {
		var info struct {
			YtDlpVersion string `json:"yt-dlp_version"`
		}
		if err := json.Unmarshal(data, &info); err == nil {
			currentVersion = info.YtDlpVersion
		}
	}
	resp, err := http.Get("https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}
	if release.TagName == currentVersion {
		fmt.Println("Yt-dlp is up to date.")
		return nil
	}
	// Wyszukujemy asset dla Linuxa – przykładowo szukamy pliku bez rozszerzenia lub z nazwą "linux"
	var assetURL string
	for _, asset := range release.Assets {
		// Możesz tutaj ustalić własne kryteria
		if strings.Contains(strings.ToLower(asset.Name), "linux") {
			assetURL = asset.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no linux asset found in release")
	}
	out, err := os.Create("yt-dlp")
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err = http.Get(assetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	// Nadajemy plikowi prawa wykonywalności
	if err := os.Chmod("yt-dlp", 0755); err != nil {
		return err
	}
	infoData, err := json.MarshalIndent(struct {
		YtDlpVersion string `json:"yt-dlp_version"`
	}{release.TagName}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("info.json", infoData, 0644); err != nil {
		return err
	}
	fmt.Println("Yt-dlp updated to", release.TagName)
	return nil
}
