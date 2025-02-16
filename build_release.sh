LDFLAGS="-s -w"

YT_DLP_DL="yt-dlp"
YT_DLP="yt-dlp"
GTG_NOUPX="GoTubeGUI-noupx"
GTG_UPX="GoTubeGUI-upx"
set GOOS=linux

if [ "$GOOS" = "windows" ]; then
  LDFLAGS="$LDFLAGS -H=windowsgui"
  YT_DLP_DL="$YT_DLP_DL.exe"
  YT_DLP="$YT_DLP.exe"
  GTG_NOUPX="$GTG_NOUPX.exe"
  GTG_UPX="$GTG_UPX.exe"
elif [ "$GOOS" = "linux" ]; then
  YT_DLP_DL="${YT_DLP_DL}_linux"
else
  echo "Set the GOOS environment variable!"
  exit
fi

mkdir -p release-out

# Download yt-dlp
# TODO: This should be handled by GoTubeGUI itself!!!
if [ ! -f "release-out/$YT_DLP" ]; then
  echo "$YT_DLP doesn't exist, downloading..."
  curl -L "https://github.com/yt-dlp/yt-dlp/releases/download/2024.11.04/$YT_DLP_DL" -o "release-out/$YT_DLP"
fi


echo "Building $GTG_NOUPX"
go build -ldflags "$LDFLAGS" -o "release-out/$GTG_NOUPX"


if [ -f "upx" ] || [ -f "upx.exe" ]; then
  echo "upx-ing..."
  ./upx -o"release-out/$GTG_UPX" -f "release-out/$GTG_NOUPX"
else
  echo "upx doesn't exist. Not compressing!"
  exit
fi
