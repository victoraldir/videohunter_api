package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohunterapi/application"
	"github.com/victoraldir/myvideohunterapi/config"
)

// Serves the public browser configuration: the Cognito hosted UI endpoints and
// the chat websocket endpoint.
func main() {

	config.Init()

	handler := application.NewConfigHandler(config.Config)

	lambda.Start(handler.Handle)
}
