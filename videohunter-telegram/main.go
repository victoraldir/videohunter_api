package main

import (
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	handler := NewHandler(httpClient)
	lambda.Start(handler.lambdaHandler)
}
