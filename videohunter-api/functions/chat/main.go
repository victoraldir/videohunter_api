package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohunterapi/application"
	"github.com/victoraldir/myvideohunterapi/config"
)

// Serves the websocket routes of the chat room attached to a video page.
// $connect checks the login, every other route carries a room action.
func main() {

	config.Init()

	handler := application.NewChatHandler(config.Config)

	lambda.Start(handler.Handle)
}
