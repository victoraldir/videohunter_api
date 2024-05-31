package ffmpeg

// func TestDownloaderHlsRepository(t *testing.T) {

// 	// Arrange
// 	url := "https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8?a=1718310735%2COGE5ZDY5NmE0MzY3NmQyM2UzMTNkNTJkZmMxMmRhNzg4MmM2MzQzNTczYzY0YTYzOGFjMzQwNWQ4ZTViN2I0Zg%3D%3D&amp;v=1&amp;f=sd"
// 	downloaderHlsRepository := NewDownloaderHlsRepository()

// 	// Act
// 	video, err := downloaderHlsRepository.DownloadHls(url)

// 	// Assert
// 	assert.Nil(t, err)
// 	assert.NotNil(t, video)
// }

// func TestDownloaderHlsRepository_MixAudioAndVideo(t *testing.T) {

// 	// Arrange
// 	videoUrl := "https://v.redd.it/b4cikpfnw80d1/DASH_480.mp4?source=fallback"
// 	audioUrl := "https://v.redd.it/b4cikpfnw80d1/DASH_AUDIO_128.mp4"
// 	downloaderHlsRepository := NewDownloaderHlsRepository()

// 	// Act
// 	video, err := downloaderHlsRepository.MixAudioAndVideo(videoUrl, audioUrl)

// 	// Assert
// 	assert.Nil(t, err)
// 	assert.NotNil(t, video)
// 	assert.FileExists(t, video.Path)
// }
