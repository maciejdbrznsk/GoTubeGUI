package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	dialog2 "fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/sqweek/dialog"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var ytdlp = "./yt-dlp"
var resolutionFormatMap = make(map[string]string)

type Format struct {
	FormatID   string   `json:"format_id"`
	FormatNote string   `json:"format_note"`
	Ext        string   `json:"ext"`
	Acodec     string   `json:"acodec"`
	Vcodec     string   `json:"vcodec"`
	Url        string   `json:"url"`
	Width      *int     `json:"width"`
	Height     *int     `json:"height"`
	Fps        *float64 `json:"fps"`
	Abr        *float64 `json:"abr"`
	Vbr        *float64 `json:"vbr"`
	Resolution string   `json:"resolution"`
}

type VideoInfo struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Formats      []Format `json:"formats"`
	Storyboards  []Format
	AudioOnly    []Format
	VideoFormats []Format
}

func main() {
	GTD := app.New()
	window := GTD.NewWindow("GoTube Downloader")

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Paste video URL")
	urlEntry.SetText("Paste video URL")
	downloadPath, err := getDownloadPath()
	if err != nil {
		log.Fatal(err)
	}
	downloadPathLabel := widget.NewLabel(downloadPath)

	var videoInfo VideoInfo

	selectFolderButton := widget.NewButton("Select Folder", func() {
		folder, err := dialog.Directory().Title("Select Folder").Browse()
		if err != nil {
			log.Println("Error has occurred:", err)
			return
		}
		downloadPath = folder
		downloadPathLabel.SetText(downloadPath)
	})

	videoTitle := widget.NewLabel("Video Title: ")
	videoDescription := widget.NewLabel("Video Description: ")

	thumbnailImage := canvas.NewImageFromResource(nil)
	thumbnailImage.FillMode = canvas.ImageFillContain
	description := container.NewScroll(videoDescription)
	description.SetMinSize(fyne.NewSize(400, 550))

	progress := widget.NewProgressBar()
	progressText := widget.NewLabel("")
	statusText := widget.NewLabel("")
	progressChan := make(chan float64)

	go func() {
		for prog := range progressChan {
			progress.SetValue(prog)
			progressText.SetText(fmt.Sprintf("%.2f%%", prog*100))
		}
	}()

	formatSelect := widget.NewSelect([]string{"mp4", "mkv", "webm", "mp3", "wav"}, nil)
	formatSelect.SetSelected("mp4")

	resolutionSelect := widget.NewSelect([]string{}, nil)

	videoCheck := widget.NewCheck("Video", nil)
	audioCheck := widget.NewCheck("Audio", nil)
	storyboardCheck := widget.NewCheck("Storyboard", nil)

	updateResolutionList(&videoInfo, videoCheck.Checked, audioCheck.Checked, storyboardCheck.Checked, resolutionSelect)

	videoCheck = widget.NewCheck("Video", func(checked bool) {
		updateResolutionList(&videoInfo, videoCheck.Checked, audioCheck.Checked, storyboardCheck.Checked, resolutionSelect)
	})
	audioCheck = widget.NewCheck("Audio", func(checked bool) {
		updateResolutionList(&videoInfo, videoCheck.Checked, audioCheck.Checked, storyboardCheck.Checked, resolutionSelect)
	})
	storyboardCheck = widget.NewCheck("Storyboard", func(checked bool) {
		updateResolutionList(&videoInfo, videoCheck.Checked, audioCheck.Checked, storyboardCheck.Checked, resolutionSelect)
	})
	videoCheck.SetChecked(true)
	audioCheck.SetChecked(true)
	storyboardCheck.SetChecked(true)

	getVideoInfoButton := widget.NewButton("Get VideoInfo", func() {
		inputURL := urlEntry.Text
		if inputURL == "" {
			dialog2.ShowInformation("Error", "Invalid URL", window)
			return
		}

		go func() {
			log.Println("Getting VideoInfo...")
			progress.SetValue(0)
			statusText.SetText("Getting VideoInfo...")

			err := getVideoInfo(inputURL, &videoInfo)
			if err != nil {
				log.Println("Error has occurred:", err)
				dialog2.ShowError(err, window)
				return
			}

			log.Println("Getting VideoInfo: success")
			videoTitle.SetText("Title: " + videoInfo.Title)
			videoDescription.SetText("Description: " + videoInfo.Description)

			statusText.SetText("Info fetched")
			progress.SetValue(100)
		}()
	})

	downloadVideoButton := widget.NewButton("Download", func() {
		inputURL := urlEntry.Text
		selectedDisplay := resolutionSelect.Selected
		if inputURL == "" || selectedDisplay == "" {
			dialog2.ShowInformation("Error", "Invalid URL or Resolution", window)
			return
		}
		selectedFormatID, ok := resolutionFormatMap[selectedDisplay]
		if !ok {
			dialog2.ShowInformation("Error", "Selected format not found", window)
			return
		}
		progress.SetValue(0)

		go func() {
			err := downloadVideo(inputURL, selectedFormatID, downloadPath, progressChan, statusText)
			if err != nil {
				dialog2.ShowError(err, window)
			} else {
				dialog2.ShowInformation("Success", "Video Downloaded", window)
			}
		}()
	})

	separator := widget.NewSeparator()

	leftSide := container.NewVBox(
		widget.NewLabel("Paste video URL"),
		urlEntry,
		widget.NewSeparator(),
		selectFolderButton,
		downloadPathLabel,
		widget.NewSeparator(),
		formatSelect,
		widget.NewSeparator(),
		container.NewHBox(videoCheck, audioCheck, storyboardCheck),
		resolutionSelect,
		downloadVideoButton,
		getVideoInfoButton,
		widget.NewSeparator(),
		progress,
		progressText,
		statusText,
	)

	rightSide := container.NewVBox(
		thumbnailImage,
		videoTitle,
		separator,
		description,
	)

	window.SetContent(container.NewHSplit(leftSide, rightSide))
	window.Resize(fyne.NewSize(800, 600))
	window.SetFixedSize(true)
	window.ShowAndRun()
}

func updateResolutionList(info *VideoInfo, includeVideo, includeAudio, includeStoryboard bool, sel *widget.Select) {
	resolutionFormatMap = make(map[string]string)
	var options []string

	if includeVideo {
		for _, format := range info.VideoFormats {
			if format.Width != nil && format.Height != nil {
				fpsStr := "N/A"
				if format.Fps != nil {
					fpsStr = fmt.Sprintf("%.2f", *format.Fps)
				}
				vbrStr := "N/A"
				if format.Vbr != nil {
					vbrStr = fmt.Sprintf("%.2f", *format.Vbr)
				}
				display := fmt.Sprintf("Video: %s | FPS: %s | VBR: %s | Ext: %s", format.Resolution, fpsStr, vbrStr, format.Ext)
				options = append(options, display)
				resolutionFormatMap[display] = format.FormatID
			}
		}
	}

	if includeAudio {
		for _, format := range info.AudioOnly {
			abrStr := "N/A"
			if format.Abr != nil {
				abrStr = fmt.Sprintf("%.2f", *format.Abr)
			}
			display := fmt.Sprintf("Audio: ABR: %s | Ext: %s", abrStr, format.Ext)
			options = append(options, display)
			resolutionFormatMap[display] = format.FormatID
		}
	}

	if includeStoryboard {
		for _, format := range info.Storyboards {
			display := fmt.Sprintf("Storyboard: %s | Ext: %s", format.Resolution, format.Ext)
			options = append(options, display)
			resolutionFormatMap[display] = format.FormatID
		}
	}

	sel.Options = options
	sel.Refresh()
}

func getVideoInfo(url string, info *VideoInfo) error {
	cmd := exec.Command(ytdlp, "-j", "--quiet", "--no-warnings", url)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	decoder := json.NewDecoder(stdout)
	if err := decoder.Decode(info); err != nil && err != io.EOF {
		return err
	}
	
	for _, format := range info.Formats {
		if format.Resolution == "audio only" {
			info.AudioOnly = append(info.AudioOnly, format)
		} else if format.Width == nil || format.Height == nil {
			info.Storyboards = append(info.Storyboards, format)
		} else {
			info.VideoFormats = append(info.VideoFormats, format)
		}
	}

	fmt.Printf("Video ID: %s\nTitle: %s\n", info.ID, info.Title)
	return cmd.Wait()
}

func splitCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func downloadVideo(url, formatID, downloadPath string, progressChan chan float64, statusText *widget.Label) error {
	formatOption := formatID
	outputPath := filepath.Join(downloadPath, "%(title)s.%(ext)s")
	log.Printf("Download format: %s", formatOption)
	log.Printf("Output Path: %s", outputPath)
	args := []string{
		"-o", outputPath,
		"-f", formatOption,
		"--no-check-certificate",
		"--merge-output-format", "mp4",
		url,
	}
	cmd := exec.Command(ytdlp, args...)
	log.Printf("Running command: %v", cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error has occurred: %s", err)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Error has occurred: %s", err)
		return fmt.Errorf("error has occurred: %v", err)
	}
	re := regexp.MustCompile(`(?i)\[download\]\s+(\d+(?:\.\d+)?)%\s+of\s+([\d.]+\s*[KMGT]iB)(?:\s+in\s+([\d:]+))?\s+at\s+([\d.]+\s*[KMGT]iB/s)(?:\s+ETA\s+([\d:]+))?`)
	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitCRLF)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			progressValue := matches[1]
			totalSize := "Unknown"
			speed := "Unknown"
			eta := "Unknown"
			if len(matches) >= 3 && strings.TrimSpace(matches[2]) != "" {
				totalSize = matches[2]
			}
			if len(matches) >= 5 && strings.TrimSpace(matches[4]) != "" {
				speed = matches[4]
			}
			if len(matches) >= 6 && strings.TrimSpace(matches[5]) != "" {
				eta = matches[5]
			}
			statusText.SetText(fmt.Sprintf("Progress: %s%% | Total: %s | Speed: %s | ETA: %s", progressValue, totalSize, speed, eta))
			prog, err := strconv.ParseFloat(progressValue, 64)
			if err == nil {
				progressChan <- prog / 100.0
			}
		} else {
			log.Println("Skipping line:", line)
		}
	}
	if err := cmd.Wait(); err != nil {
		log.Println("Error has occurred: ", err)
		return fmt.Errorf("error has occurred: %v", err)
	}
	log.Println("Finished downloading video.")
	return nil
}
