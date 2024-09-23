LDFLAGS="-s -w"

if [ "$GOOS" = "windows" ]; then
  YT_DLP_DL="yt-dlp.exe"
  YT_DLP="yt-dlp.exe"
  GTG_NOUPX="GoTubeGUI-noupx.exe"
  GTG_UPX="GoTubeGUI-upx.exe"
elif [ "$GOOS" = "linux" ]; then
  YT_DLP_DL="yt-dlp_linux"
  YT_DLP="yt-dlp"
  GTG_NOUPX="GoTubeGUI-noupx"
  GTG_UPX="GoTubeGUI-upx"
else
  echo "Set the GOOS environment variable!"
  exit
fi

mkdir -p release-out

# Download yt-dlp
# TODO: This should be handled by GoTubeGUI itself!!!
if [ ! -f "release-out/$YT_DLP" ]; then
  echo "$YT_DLP doesn't exist, downloading..."
  curl -L "https://github.com/yt-dlp/yt-dlp/releases/download/2024.08.06/$YT_DLP_DL" -o "release-out/$YT_DLP"
fi


echo "Building $GTG_NOUPX"
go build -ldflags "$LDFLAGS" -o "release-out/$GTG_NOUPX"


if [ -f "upx" ] || [ -f "upx.exe" ]; then
  echo "upx-ing..."
  ./upx -o"release-out/$GTG_UPX" -f "release-out/$GTG_NOUPX"
else
  echo "upx doesn't exist. Not compressing!"
fi
