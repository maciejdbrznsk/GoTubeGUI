package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"math"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
)

var ytDlp = "/home/maciej/GolandProjects/GoTubeGUI/yt-dlp"
var downloadPath = getDefaultDownloadPath()

type videoInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("GoTube Downloader")

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Paste YouTube video link")

	downloadPathLabel := widget.NewLabel(downloadPath)

	selectFolderButton := widget.NewButton("Select Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri != nil {
				downloadPath = uri.Path()
				downloadPathLabel.SetText(downloadPath)
			}
		}, myWindow)
	})

	videoTitle := widget.NewLabel("Video Title: ")
	videoDescription := widget.NewLabel("Video Description: ")

	scrollableDescription := container.NewScroll(videoDescription)
	scrollableDescription.SetMinSize(fyne.NewSize(400, 550))

	Progress := widget.NewProgressBar()
	ProgressText := widget.NewLabel("0%")

	progressChan := make(chan float64)

	go func() {
		for progress := range progressChan {
			Progress.SetValue(progress)
			ProgressText.SetText(fmt.Sprintf("%.2f%%", progress*100))
		}
	}()

	fetchVideoInfoButton := widget.NewButton("Fetch Video Info", func() {
		url := urlEntry.Text

		if url == "" {
			dialog.ShowInformation("Error", "Incorrect URL", myWindow)
			return
		}

		go func() {
			Progress.SetValue(0)
			ProgressText.SetText("Fetching info...")

			videoInfo, err := getVideoInfo(url)
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}

			videoTitle.SetText("Title: " + videoInfo.Title)
			videoDescription.SetText("Description: " + videoInfo.Description)
			ProgressText.SetText("Info fetched")
			Progress.SetValue(100)
		}()
	})

	downloadButton := widget.NewButton("Download", func() {
		url := urlEntry.Text
		re := regexp.MustCompile(`^https?://(www\.)?(youtube\.com|youtu\.be)/.+$`)

		if url == "" {
			dialog.ShowInformation("Error", "Incorrect URL", myWindow)
			return
		} else if !re.MatchString(url) {
			dialog.ShowError(fmt.Errorf("Invalid YouTube URL"), myWindow)
			return
		} else if downloadPath == "" {
			dialog.ShowInformation("Error", "Incorrect Download Path", myWindow)
			return
		}

		Progress.SetValue(0)

		go func() {
			err := downloadVideo(url, downloadPath, progressChan)
			if err != nil {
				dialog.ShowError(err, myWindow)
			} else {
				dialog.ShowInformation("Success", "Video Downloaded", myWindow)
			}
		}()
	})

	separator := widget.NewSeparator()

	leftSide := container.NewVBox(
		widget.NewLabel("Paste YouTube link"),
		urlEntry,
		selectFolderButton,
		downloadPathLabel,
		downloadButton,
		fetchVideoInfoButton,
		widget.NewSeparator(),
		Progress,
		ProgressText,
	)

	rightSide := container.NewVBox(
		videoTitle,
		separator,
		scrollableDescription,
	)

	myWindow.SetContent(
		container.NewHSplit(leftSide, rightSide),
	)

	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.SetFixedSize(true)
	myWindow.ShowAndRun()
}

func downloadVideo(url, downloadPath string, progressChan chan float64) error {
	cmd := exec.Command(ytDlp, "-P", downloadPath, url, "--newline")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Error starting download: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Error starting download: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	re := regexp.MustCompile(`(\d{1,3}\.\d+)%`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 0 {
			progressPercent, _ := strconv.ParseFloat(matches[1], 64)

			roundedProgress := round(progressPercent)
			progressChan <- roundedProgress / 100
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Error during download: %v", err)
	}

	return nil
}

func getDefaultDownloadPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(usr.HomeDir, "Downloads")
}

func getVideoInfo(url string) (videoInfo, error) {
	cmd := exec.Command(ytDlp, "-j", "--skip-download", url)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return videoInfo{}, fmt.Errorf("Error while downloading video information: %v, %s", err, string(output))
	}

	fmt.Println("Raw yt-dlp output:", string(output))

	var info videoInfo

	if err := json.Unmarshal(output, &info); err != nil {
		return videoInfo{}, fmt.Errorf("Error while parsing information: %v", err)
	}

	return info, nil
}

func round(val float64) float64 {
	return math.Round(val)
}
