package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	telegramBotToken = os.Getenv("BOT_TOKEN")
	headers          = map[string]string{"Content-Type": "application/json"}
)

//go:generate mockgen -destination=./mockHttpClient.go -package=main github.com/victoraldir/myvideohuntertelegram HttpClient
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type handler struct {
	Client HttpClient
}

func NewHandler(client HttpClient) *handler {
	return &handler{
		Client: client,
	}
}

func (h *handler) lambdaHandler(event Event) (map[string]interface{}, error) {
	log.Println("Event:", event)
	telegramBotToken = os.Getenv("BOT_TOKEN")

	url := "https://myvideohunter.com/prod/url"

	// Remove /n from the body
	bodySanitized := strings.ReplaceAll(event.Body, "\n", "")
	var telegramMsg TelegramMessage
	json.Unmarshal([]byte(bodySanitized), &telegramMsg)

	if telegramMsg.Message.Text == "" {
		log.Println("Key 'message' not found in the body.")
		return map[string]interface{}{
			"statusCode": 400,
			"body":       "Key 'message' not found in the body",
		}, nil
	}

	telegramChatID := telegramMsg.Message.From.ID

	if telegramMsg.Message.Text == "/help" {
		helpMessage := "Send me a Twitter, Bsky or Reddit video URL and I'll send you the video to download 🎥"
		sendMessage(helpMessage, telegramChatID)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "ok",
		}, nil
	}

	if telegramMsg.Message.Text == "/about" {
		aboutMessage := "This bot is part of the VideoHunter project. \n\nYou can access the website at https://myvideohunter.com"
		sendMessage(aboutMessage, telegramChatID)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "ok",
		}, nil
	}

	if !isValidURL(telegramMsg.Message.Text) {
		log.Println("Invalid URL")
		sendMessage("Invalid URL", telegramChatID)
		return map[string]interface{}{
			"statusCode": 400,
			"body":       "Invalid URL",
		}, nil
	}

	urlTwitter := telegramMsg.Message.Text

	data := map[string]string{"video_url": urlTwitter}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshalling data:", err)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "Error marshalling data",
		}, nil
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(dataJSON))
	if err != nil {
		log.Println("Error creating request:", err)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "Error creating request",
		}, nil
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		sendMessage("HTTP downaloding the video", telegramChatID)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "HTTP Error occurred",
		}, nil
	}
	defer resp.Body.Close()

	var videoResponse VideoResponse
	err = json.NewDecoder(resp.Body).Decode(&videoResponse)
	if err != nil {
		log.Println("Error decoding response:", err)
		return map[string]interface{}{
			"statusCode": 200,
			"body":       "Error decoding response",
		}, nil
	}

	videoID := videoResponse.ID
	log.Println("Video ID:", videoID)

	fullMessage := fmt.Sprintf("Here's your video: \n\n %s/%s", url, videoID)
	sendMessage(fullMessage, telegramChatID)

	return map[string]interface{}{
		"statusCode": 200,
		"body":       "ok",
	}, nil
}

func sendMessage(message string, chatID int64) {
	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	params := map[string]string{
		"chat_id": fmt.Sprintf("%v", chatID),
		"text":    message,
	}

	bodyMessage, err := json.Marshal(params)
	if err != nil {
		log.Println("Error marshalling params:", err)
		return
	}

	req, err := http.NewRequest("POST", telegramURL, bytes.NewBuffer(bodyMessage))
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Response:", resp)
}

func isValidURL(url string) bool {
	// Check if the URL is a valid X (former twitter), Bsky or Reddit video URL
	regex := regexp.MustCompile(`(https:\/\/twitter\.com\/.*\/status\/\d+)|https:\/\/x\.com\/.*\/status\/\d+|(https:\/\/bsky\.app\/.*\/post\/\d+)|(https:\/\/www\.reddit\.com\/r\/.*\/comments\/.*\/.*\/)`)

	return regex.MatchString(url)
}
