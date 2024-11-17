package main

import (
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/victoraldir/myvideohunterbsky/application"
)

func main() {

	bskyUserName := os.Getenv("BSKY_USERNAME")
	bskyPassword := os.Getenv("BSKY_PASSWORD")

	slog.Info("credentials", slog.Any("bskyUserName", bskyUserName), slog.Any("bskyPassword", bskyPassword))

	handler := application.NewFetchPostHandler()

	lambda.Start(handler.Handle)
}
