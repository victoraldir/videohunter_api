package ffmpeg

import (
	"bytes"
	"fmt"
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

	// Use ffmpeg to download the video
	// Command: ffmpeg -i {url} -c copy -bsf:a aac_adtstoasc /tmp/output.mp4
	timestamp := time.Now()
	videoPath := fmt.Sprintf("/tmp/%s.mp4", timestamp.Format("20060102150405"))

	command := exec.Command("ffmpeg", "-i", url, "-c", "copy", "-bsf:a", "aac_adtstoasc", videoPath)

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
