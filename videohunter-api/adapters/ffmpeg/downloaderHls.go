package ffmpeg

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/victoraldir/myvideohunterapi/domain"
)

type downloaderHlsRepository struct {
}

func NewDownloaderHlsRepository() *downloaderHlsRepository {
	return &downloaderHlsRepository{}
}

func (r *downloaderHlsRepository) DownloadHls(url string) (videoDownload *domain.Video, err error) {

	// Probe if url https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8 is reachable
	// Command: ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 {url}
	log.Println("Probing video from: ", url)
	command := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", url)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	// Execute the command
	err = command.Run()
	if err != nil {
		return nil, err
	}

	// Use ffmpeg to download the video
	// Command: ffmpeg -i {url} -c copy -bsf:a aac_adtstoasc /tmp/output.mp4
	timestamp := time.Now()
	videoPath := fmt.Sprintf("/tmp/%s.mp4", timestamp.Format("20060102150405"))

	log.Println("Downloading video from: ", url)
	log.Println("Saving video to: ", videoPath)
	command = exec.Command("ffmpeg", "-i", url, "-c", "copy", "-bsf:a", "aac_adtstoasc", videoPath)

	var outb, errb bytes.Buffer

	command.Stdout = &outb
	command.Stderr = &errb

	// Execute the command
	err = command.Run()

	log.Println("out:", outb.String(), "err:", errb.String())

	if err != nil {
		return nil, err
	}

	videoDownload = &domain.Video{
		Path: videoPath,
	}

	return videoDownload, nil
}

func (r *downloaderHlsRepository) MixAudioAndVideo(videoUrl, audioUrl string) (videoDownload *domain.Video, err error) {

	// Use ffmpeg to mix audio and video
	// Command: ffmpeg -i {videoUrl} -i {audioUrl} -c:v copy -c:a aac -strict experimental /tmp/output.mp4
	timestamp := time.Now()
	videoPath := fmt.Sprintf("/tmp/%s.mp4", timestamp.Format("20060102150405"))

	slog.Info("Mixing audio and video")
	//ffmpeg -i video.mp4 -i audio.wav -c:v copy -c:a aac output.mp4
	command := exec.Command("ffmpeg", "-i", videoUrl, "-i", audioUrl, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", videoPath)

	var outb, errb bytes.Buffer

	command.Stdout = &outb
	command.Stderr = &errb

	// Execute the command
	err = command.Run()

	if err != nil {
		return nil, err
	}

	fmt.Println("out:", outb.String(), "err:", errb.String())

	videoDownload = &domain.Video{
		Path: videoPath,
	}

	return videoDownload, nil
}
