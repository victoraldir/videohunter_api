package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohunterapi/application"
	"github.com/victoraldir/myvideohunterapi/config"
)

func main() {

	// Load configuration
	config.Init()

	// Create Lambda Application
	lambdaApplication := application.NewAPIGatewayHandler(config.Config)

	// Start Lambda. At the moment, we only have one handler.
	lambda.Start(lambdaApplication.GetUrlHandler.Handle)
}
