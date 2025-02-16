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
var videoFormatMap = make(map[string]string)
var audioFormatMap = make(map[string]string)

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
	AudioOnly    []Format
	VideoFormats []Format
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("GoTube Downloader")
	window.Resize(fyne.NewSize(1200, 600))
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Paste video URL")
	downloadPath, err := getDownloadPath()
	if err != nil {
		log.Fatal(err)
	}
	downloadPathLabel := widget.NewLabel(downloadPath)
	var videoInfo VideoInfo
	log.Println(updateYtDlp())
	selectFolderButton := widget.NewButton("Select Folder", func() {
		folder, err := dialog.Directory().Title("Select Folder").Browse()
		if err != nil {
			log.Println("Error:", err)
			return
		}
		downloadPath = folder
		downloadPathLabel.SetText(downloadPath)
	})
	videoTitle := widget.NewLabel("Title:")
	videoDescription := widget.NewLabel("Description:")
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
	mediaTypeRadio := widget.NewRadioGroup([]string{"Video + Audio", "Video", "Audio"}, nil)
	mediaTypeRadio.SetSelected("Video + Audio")
	videoQualitySelect := widget.NewSelect([]string{}, nil)
	audioQualitySelect := widget.NewSelect([]string{}, nil)
	fileFormatSelect := widget.NewSelect([]string{}, nil)
	setFileFormatOptions := func(mediaType string) {
		if mediaType == "Video" {
			fileFormatSelect.Options = []string{"mp4", "mkv", "webm"}
		} else if mediaType == "Audio" {
			fileFormatSelect.Options = []string{"mp3", "wav", "m4a", "aac"}
		} else if mediaType == "Video+Audio" {
			fileFormatSelect.Options = []string{"mp4", "mkv", "webm"}
		}
		fileFormatSelect.Refresh()
	}
	mediaTypeRadio.OnChanged = func(selected string) {
		if selected == "Video" {
			videoQualitySelect.Show()
			audioQualitySelect.Hide()
		} else if selected == "Audio" {
			videoQualitySelect.Hide()
			audioQualitySelect.Show()
		} else {
			videoQualitySelect.Show()
			audioQualitySelect.Show()
		}
		setFileFormatOptions(selected)
	}
	getVideoInfoButton := widget.NewButton("Get Video Info", func() {
		inputURL := urlEntry.Text
		if inputURL == "" {
			dialog2.ShowInformation("Error", "Invalid URL", window)
			return
		}
		go func() {
			progress.SetValue(0)
			statusText.SetText("Getting video info...")
			err := getVideoInfo(inputURL, &videoInfo)
			if err != nil {
				dialog2.ShowError(err, window)
				return
			}
			videoTitle.SetText("Title: " + videoInfo.Title)
			videoDescription.SetText("Description: " + videoInfo.Description)
			updateVideoQualityList(&videoInfo, videoQualitySelect)
			updateAudioQualityList(&videoInfo, audioQualitySelect)
			statusText.SetText("Info fetched")
			progress.SetValue(1.0)
		}()
	})
	downloadButton := widget.NewButton("Download", func() {
		inputURL := urlEntry.Text
		if inputURL == "" {
			dialog2.ShowInformation("Error", "Invalid URL", window)
			return
		}
		progress.SetValue(0)
		mediaType := mediaTypeRadio.Selected
		go func() {
			if mediaType == "Video" {
				if videoQualitySelect.Selected == "" {
					dialog2.ShowInformation("Error", "Select video quality", window)
					return
				}
				formatID := videoFormatMap[videoQualitySelect.Selected]
				err := downloadMedia(inputURL, formatID, downloadPath, progressChan, statusText, "", false, "")
				if err != nil {
					dialog2.ShowError(err, window)
				} else {
					dialog2.ShowInformation("Success", "Video downloaded", window)
				}
			} else if mediaType == "Audio" {
				if audioQualitySelect.Selected == "" {
					dialog2.ShowInformation("Error", "Select audio quality", window)
					return
				}
				formatID := audioFormatMap[audioQualitySelect.Selected]
				err := downloadMedia(inputURL, formatID, downloadPath, progressChan, statusText, "", true, fileFormatSelect.Selected)
				if err != nil {
					dialog2.ShowError(err, window)
				} else {
					dialog2.ShowInformation("Success", "Audio downloaded", window)
				}
			} else if mediaType == "Video+Audio" {
				if videoQualitySelect.Selected == "" || audioQualitySelect.Selected == "" {
					dialog2.ShowInformation("Error", "Select both video and audio quality", window)
					return
				}
				videoFormatID := videoFormatMap[videoQualitySelect.Selected]
				audioFormatID := audioFormatMap[audioQualitySelect.Selected]
				combinedFormat := videoFormatID + "+" + audioFormatID
				err := downloadMedia(inputURL, combinedFormat, downloadPath, progressChan, statusText, fileFormatSelect.Selected, false, "")
				if err != nil {
					dialog2.ShowError(err, window)
				} else {
					dialog2.ShowInformation("Success", "Video + Audio merged and downloaded", window)
				}
			}
		}()
	})
	leftSide := container.NewVBox(
		widget.NewLabel("Paste video URL"),
		urlEntry,
		selectFolderButton,
		downloadPathLabel,
		widget.NewSeparator(),
		widget.NewLabel("Select Media Type:"),
		mediaTypeRadio,
		widget.NewLabel("Video Quality:"),
		videoQualitySelect,
		widget.NewLabel("Audio Quality:"),
		audioQualitySelect,
		widget.NewLabel("Select file format:"),
		fileFormatSelect,
		widget.NewSeparator(),
		downloadButton,
		getVideoInfoButton,
		progress,
		progressText,
		statusText,
	)
	rightSide := container.NewVBox(
		thumbnailImage,
		videoTitle,
		description,
	)
	window.SetContent(container.NewHSplit(leftSide, rightSide))
	window.ShowAndRun()
}

func updateVideoQualityList(info *VideoInfo, sel *widget.Select) {
	var options []string
	videoFormatMap = make(map[string]string)
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
			display := fmt.Sprintf("Video: %s | FPS: %s | VBR: %s", format.Resolution, fpsStr, vbrStr)
			options = append(options, display)
			videoFormatMap[display] = format.FormatID
		}
	}
	sel.Options = options
	sel.Refresh()
}

func updateAudioQualityList(info *VideoInfo, sel *widget.Select) {
	var options []string
	audioFormatMap = make(map[string]string)
	for _, format := range info.AudioOnly {
		abrStr := "N/A"
		if format.Abr != nil {
			abrStr = fmt.Sprintf("%.2f", *format.Abr)
		}
		display := fmt.Sprintf("Audio: ABR: %s", abrStr)
		options = append(options, display)
		audioFormatMap[display] = format.FormatID
	}
	sel.Options = options
	sel.Refresh()
}

func getVideoInfo(url string, info *VideoInfo) error {
	cmd := exec.Command(ytdlp, "-j", "--quiet", "--no-warnings", url)
	stdout, err := cmd.StdoutPipe()
	cmd.SysProcAttr = getOSSysProcAttr()
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
		} else if format.Width != nil && format.Height != nil {
			info.VideoFormats = append(info.VideoFormats, format)
		}
	}
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

func downloadMedia(url, formatID, downloadPath string, progressChan chan float64, statusText *widget.Label, mergeExt string, isAudioOnly bool, audioOutputFormat string) error {
	outputPath := filepath.Join(downloadPath, "%(title)s.%(ext)s")
	args := []string{
		"-o", outputPath,
		"-f", formatID,
		"--no-check-certificate",
	}
	if isAudioOnly {
		args = append(args, "--extract-audio")
		if audioOutputFormat != "" {
			args = append(args, "--audio-format", audioOutputFormat)
		}
	}
	if mergeExt != "" {
		args = append(args, "--merge-output-format", mergeExt)
	}
	args = append(args, url)
	cmd := exec.Command(ytdlp, args...)
	stdout, err := cmd.StdoutPipe()
	cmd.SysProcAttr = getOSSysProcAttr()
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Error: %v", err)
	}
	re := regexp.MustCompile(`(?i)\[download]\s+(\d+(?:\.\d+)?)%\s+of\s+(?:~\s+)?([\d.]+\s*[KMGT]iB)(?:\s+in\s+([\d:]+))?\s+at\s+([\d.]+\s*[KMGT]iB/s)(?:\s+ETA\s+([\d:]+))?`)
	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitCRLF)
	for scanner.Scan() {
		line := scanner.Text()
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
			if prog, err := strconv.ParseFloat(progressValue, 64); err == nil {
				progressChan <- prog / 100.0
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Error: %v", err)
	}
	return nil
}
