//go:build windows
// +build windows

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
	// Wczytujemy aktualną wersję z pliku info.json (jeśli istnieje)
	currentVersion := ""
	if data, err := os.ReadFile("info.json"); err == nil {
		var info struct {
			YtDlpVersion string `json:"yt-dlp_version"`
		}
		if err := json.Unmarshal(data, &info); err == nil {
			currentVersion = info.YtDlpVersion
		}
	}

	// Pobieramy informacje o najnowszym release z GitHuba
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

	// Wyszukujemy asset kończący się na .exe
	var assetURL string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".exe") {
			assetURL = asset.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no .exe asset found in release")
	}

	// Pobieramy asset i zapisujemy jako yt-dlp.exe
	out, err := os.Create("yt-dlp.exe")
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

	// Zapisujemy nową wersję do pliku info.json
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
