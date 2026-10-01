package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohunterapi/application"
	"github.com/victoraldir/myvideohunterapi/config"
)

// Serves the account endpoints: the signed in user's library of folders and
// saved videos, and the chat block list.
func main() {

	config.Init()

	handler := application.NewUserDataHandler(config.Config)

	lambda.Start(handler.Handle)
}
