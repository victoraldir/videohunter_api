package main

import (
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohuntertwitter/application"
)

func main() {

	slog.Info("twitter bot starting",
		slog.String("bots", os.Getenv("TWITTER_BOT_USERNAMES")),
		slog.Bool("getxapiKeySet", os.Getenv("GETXAPI_KEY") != ""))

	handler := application.NewFetchMentionsHandler()

	lambda.Start(handler.Handle)
}
