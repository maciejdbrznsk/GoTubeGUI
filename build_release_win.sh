# Exported Go flags
export GOOS=windows
export GOARCH=amd64

# Local variables
LDFLAGS="-s -w"

mkdir -p release-out

# Download yt-dlp.exe
# TODO: This should be handled by GoTubeGUI itself!!!
if [ ! -f "release-out/yt-dlp.exe" ]; then
  echo "yt-dlp.exe doesn't exist, downloading..."
  curl -L "https://github.com/yt-dlp/yt-dlp/releases/download/2024.08.06/yt-dlp.exe" -o "release-out/yt-dlp.exe"
fi


echo "Building GoTubeGUI-noupx.exe"
go build -ldflags "$LDFLAGS" -o "release-out/GoTubeGUI-noupx.exe"


if [ -f "upx" ] || [ -f "upx.exe" ]; then
  echo "upx-ing..."
  ./upx -o"release-out/GoTubeGUI-upx.exe" -f "release-out/GoTubeGUI-noupx.exe"
else
  echo "upx doesn't exist. Not compressing!"
fi
