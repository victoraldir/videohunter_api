package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var httpClientMock *MockHttpClient

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpClientMock = NewMockHttpClient(ctrl)
}

func TestMainHelp(t *testing.T) {

	setup(t)

	// Load file from fixtures
	eventFile, err := os.OpenFile("fixture/event_help.json", os.O_RDONLY, 0644)
	if err != nil {
		t.Error("Error opening file:", err)
	}

	eventBytes, err := io.ReadAll(eventFile)
	if err != nil {
		t.Error("Error reading file:", err)
	}

	// Call main function
	var event Event
	json.Unmarshal(eventBytes, &event)
	httpClientMock.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(nil),
	}, nil)

	handler := NewHandler(httpClientMock)

	response, err := handler.lambdaHandler(event)

	// assert
	assert.Nil(t, err)
	assert.Equal(t, 200, response["statusCode"])
}

func TestMainAbout(t *testing.T) {

	setup(t)

	// Load file from fixtures
	eventFile, _ := os.OpenFile("fixture/event_about.json", os.O_RDONLY, 0644)
	eventBytes, _ := io.ReadAll(eventFile)

	// Call main function
	var event Event
	json.Unmarshal(eventBytes, &event)

	handler := NewHandler(httpClientMock)
	httpClientMock.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(nil),
	}, nil)
	response, err := handler.lambdaHandler(event)

	// assert
	assert.Nil(t, err)
	assert.Equal(t, 200, response["statusCode"])

}

func TestMainRedditVideo(t *testing.T) {

	setup(t)

	// Load file from fixtures
	eventFile, _ := os.OpenFile("fixture/event_reddit.json", os.O_RDONLY, 0644)

	eventBytes, _ := io.ReadAll(eventFile)

	// Call main function
	var event Event
	json.Unmarshal(eventBytes, &event)

	handler := NewHandler(httpClientMock)
	httpClientMock.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(nil),
	}, nil)
	response, err := handler.lambdaHandler(event)

	// assert
	assert.Nil(t, err)
	assert.Equal(t, 200, response["statusCode"])
}

func TestMainValidUrl(t *testing.T) {

	validUrl := []string{
		"https://www.reddit.com/r/botecodoreddit/comments/1hz1wic/agora_que_eu_fui_chutado_pela_%C3%BAltima_vou_logo/",
		"https://x.com/PicturesFoIder/status/1877806319119278422",
		"https://twitter.com/elonmusk/status/1877806319119278422",
		"https://bsky.app/profile/obrasilquedeucerto.com.br/post/3lfi7by5flc2h",
		"https://www.reddit.com/r/antitrampo/s/LzWKIr9zmt",
		"https://www.reddit.com/r/botecodoreddit/s/7dQSEx7Fk6",
	}

	for _, url := range validUrl {
		fmt.Println("Testing:", url)
		assert.True(t, isValidURL(url))
	}

	invalidUrls := []string{
		"https://www.invalid.com",
		"https://google.com",
	}

	for _, url := range invalidUrls {
		fmt.Println("Testing:", url)
		assert.False(t, isValidURL(url))
	}

}
