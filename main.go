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
	"sort"
	"strconv"
)

var ytDlp = "./yt-dlp"
var ffmpegPath = "./ffmpeg"
var downloadPath = getDefaultDownloadPath()

type videoInfo struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Thumbnail   string        `json:"thumbnail"`
	Formats     []ytDlpFormat `json:"formats"`
}

type ytDlpFormat struct {
	FormatID   string  `json:"format_id"`
	Resolution string  `json:"resolution"`
	Height     int     `json:"height"`
	Width      int     `json:"width"`
	Ext        string  `json:"ext"`
	TBR        float32 `json:"tbr"`
	ASR        int     `json:"asr"`
	FPS        float64 `json:"fps"`
	VCodec     string  `json:"vcodec"`
	ACodec     string  `json:"acodec"`
	Filesize   int64   `json:"filesize"`
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
			log.Println("Starting to fetch video info...")
			Progress.SetValue(0)
			StatusText.SetText("Fetching info...")

			videoInfo, err := getVideoInfo(url)
			if err != nil {
				log.Println("Error fetching video info:", err)
				dialog.ShowError(err, myWindow)
				return
			}

			log.Println("Video info fetched successfully:", videoInfo)

			videoTitle.SetText("Title: " + videoInfo.Title)
			videoDescription.SetText("Description: " + videoInfo.Description)

			formats, err := getAvailableFormats(url)
			if err != nil {
				log.Println("Error fetching formats:", err)
				dialog.ShowError(err, myWindow)
				return
			}

			resolutions := getUniqueResolutions(formats)
			log.Println("Resolutions found:", resolutions)
			resolutionSelect.Options = resolutions
			resolutionSelect.Refresh()

			thumbnail, err := fetchAndConvertThumbnail(videoInfo.Thumbnail)
			if err == nil {
				thumbnailImage.Resource = thumbnail
				thumbnailImage.Refresh()
			} else {
				log.Println("Error loading thumbnail:", err)
			}

			StatusText.SetText("Info fetched")
			Progress.SetValue(100)
		}()
	})

	downloadButton := widget.NewButton("Download", func() {
		url := urlEntry.Text
		selectedResolution := resolutionSelect.Selected

		if url == "" || selectedResolution == "" {
			dialog.ShowInformation("Error", "Incorrect URL or resolution", myWindow)
			return
		}

		Progress.SetValue(0)

		go func() {
			err := downloadVideo(url, downloadPath, selectedResolution, progressChan)
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
		thumbnailImage,
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

func downloadVideo(url, downloadPath, resolution string, progressChan chan float64) error {
	args := []string{
		"-f", fmt.Sprintf("bestvideo[height=%s]+bestaudio/best", resolution),
		"-o", filepath.Join(downloadPath, "%(title)s.%(ext)s"),
		"--no-check-certificate",
		url,
	}

	cmd := exec.Command(ytDlp, args...)
	log.Println("Running download command:", cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error starting download: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Println("Error starting download:", err)
		return fmt.Errorf("error starting download: %v", err)
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
		log.Println("Download error:", err)
		return fmt.Errorf("error during download: %v", err)
	}

	log.Println("Download completed successfully")
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
	cmd := exec.Command(ytDlp, "-j", "--skip-download", "--no-warnings", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return videoInfo{}, fmt.Errorf("error while running yt-dlp: %v, output: %s", err, string(output))
	}

	startIdx := bytes.IndexByte(output, '{')
	endIdx := bytes.LastIndexByte(output, '}')
	if startIdx == -1 || endIdx == -1 {
		return videoInfo{}, fmt.Errorf("unable to locate JSON in output")
	}

	jsonData := output[startIdx : endIdx+1]

	var info videoInfo
	if err := json.Unmarshal(jsonData, &info); err != nil {
		return videoInfo{}, fmt.Errorf("error while parsing information: %v\nRaw output:\n%s", err, string(jsonData))
	}

	return info, nil
}

func fetchAndConvertThumbnail(url string) (*fyne.StaticResource, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching thumbnail: %v", err)
	}
	defer response.Body.Close()

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return nil, fmt.Errorf("error encoding image to jpeg: %v", err)
	}

	return fyne.NewStaticResource("thumbnail.jpg", buf.Bytes()), nil
}

func getAvailableFormats(url string) ([]ytDlpFormat, error) {
	cmd := exec.Command("yt-dlp", "-j", "--skip-download", "--format-sort", "res,br", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error while downloading video information: %v, %s", err, string(output))
	}

	startIdx := bytes.IndexByte(output, '{')
	endIdx := bytes.LastIndexByte(output, '}')
	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("unable to locate JSON in output")
	}

	jsonData := output[startIdx : endIdx+1]

	var info videoInfo
	if err := json.Unmarshal(jsonData, &info); err != nil {
		return nil, fmt.Errorf("Error while parsing information: %v", err)
	}
	return info.Formats, nil
}

func getUniqueResolutions(formats []ytDlpFormat) []string {
	resolutions := map[string]struct{}{}
	for _, format := range formats {
		if format.Resolution != "" {
			resolutions[format.Resolution] = struct{}{}
		}
	}

	sortedResolutions := make([]string, 0, len(resolutions))
	for res := range resolutions {
		sortedResolutions = append(sortedResolutions, res)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(sortedResolutions)))
	return sortedResolutions
}
