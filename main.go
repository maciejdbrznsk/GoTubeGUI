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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"image"
	"image/jpeg"
	"log"
	"math"
	"net/http"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
)

var ytDlp = "./yt-dlp"
var ffmpegPath = "./ffmpeg"
var downloadPath = getDefaultDownloadPath()

type videoInfo struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Thumbnail   string   `json:"thumbnail"`
	Formats     []format `json:"formats"`
}

type format struct {
	FormatID   string `json:"format_id"`
	Resolution string `json:"resolution"`
	FormatNote string `json:"format_note"`
	Ext        string `json:"ext"`
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

	thumbnailImage := canvas.NewImageFromResource(nil)
	thumbnailImage.FillMode = canvas.ImageFillContain
	scrollableDescription := container.NewScroll(videoDescription)
	scrollableDescription.SetMinSize(fyne.NewSize(400, 550))

	Progress := widget.NewProgressBar()
	ProgressText := widget.NewLabel("")
	StatusText := widget.NewLabel("")

	progressChan := make(chan float64)

	go func() {
		for progress := range progressChan {
			Progress.SetValue(progress)
			ProgressText.SetText(fmt.Sprintf("%.2f%%", progress*100))
		}
	}()

	// Dodaj listę wyboru formatu i rozdzielczości
	formatSelect := widget.NewSelect([]string{"mp4", "mkv", "mp3", "wav"}, nil)
	formatSelect.SetSelected("mp4")

	resolutionSelect := widget.NewSelect([]string{}, nil)

	fetchVideoInfoButton := widget.NewButton("Fetch Video Info", func() {
		url := urlEntry.Text

		if url == "" {
			dialog.ShowInformation("Error", "Incorrect URL", myWindow)
			return
		}

		go func() {
			Progress.SetValue(0)
			StatusText.SetText("Fetching info...")

			videoInfo, err := getVideoInfo(url)
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}

			videoTitle.SetText("Title: " + videoInfo.Title)
			videoDescription.SetText("Description: " + videoInfo.Description)

			// Pobierz miniaturę i ustaw ją w interfejsie
			thumbnail, err := fetchThumbnail(videoInfo.Thumbnail)
			if err == nil {
				thumbnailImage.Resource = thumbnail
				thumbnailImage.Refresh()
			}

			StatusText.SetText("Info fetched")
			Progress.SetValue(100)
		}()
	})

	downloadButton := widget.NewButton("Download", func() {
		url := urlEntry.Text
		selectedFormat := formatSelect.Selected
		selectedResolution := resolutionSelect.Selected

		if url == "" {
			dialog.ShowInformation("Error", "Incorrect URL", myWindow)
			return
		} else if downloadPath == "" {
			dialog.ShowInformation("Error", "Incorrect Download Path", myWindow)
			return
		}

		Progress.SetValue(0)

		go func() {
			err := downloadVideo(url, downloadPath, selectedFormat, selectedResolution, progressChan)
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
		widget.NewSeparator(),
		selectFolderButton,
		downloadPathLabel,
		formatSelect,
		resolutionSelect,
		downloadButton,
		fetchVideoInfoButton,
		widget.NewSeparator(),
		Progress,
		StatusText,
	)

	rightSide := container.NewVBox(
		thumbnailImage, // Miniatura nad tytułem
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

func downloadVideo(url, downloadPath, format, resolution string, progressChan chan float64) error {
	var args []string
	if resolution == "Audio Only" {
		args = append(args, "-f", "bestaudio")
	} else {
		args = append(args, "-f", "bestvideo[height="+resolution+"]+bestaudio")
	}
	args = append(args, "--ffmpeg-location", ffmpegPath, "-o", filepath.Join(downloadPath, "%(title)s.%(ext)s"), url)

	cmd := exec.Command(ytDlp, args...)
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
			roundedProgress := math.Round(progressPercent)
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

	var info videoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return videoInfo{}, fmt.Errorf("Error while parsing information: %v", err)
	}

	return info, nil
}

func fetchThumbnail(url string) (*fyne.StaticResource, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return nil, err
	}

	return fyne.NewStaticResource("thumbnail.jpg", buf.Bytes()), nil
}
