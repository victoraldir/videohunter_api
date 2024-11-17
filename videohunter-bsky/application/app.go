package application

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterbsky/repository/dynamodb"
	"github.com/victoraldir/myvideohunterbsky/services/bsky"
	"github.com/victoraldir/myvideohunterbsky/usecase"
)

type fetchPostHandler struct {
	fetchPostUserCase usecase.FetchPost
}

func NewFetchPostHandler() *fetchPostHandler {

	// Clients
	httpClient := &http.Client{}
	dynamodbClient := createDynamodbClient()

	// Services
	bskyUserName := os.Getenv("BSKY_USERNAME")
	bskyPassword := os.Getenv("BSKY_PASSWORD")

	bskyService := bsky.NewBskyService(httpClient, bskyUserName, bskyPassword)

	// Respository
	dynamodbRepo := dynamodb.NewDynamoSettingsRepository(dynamodbClient, "settings")

	// Use Cases
	fetchPostUserCase := usecase.NewFetchPost(bskyService, dynamodbRepo)

	return &fetchPostHandler{fetchPostUserCase: fetchPostUserCase}
}

func createDynamodbClient() *dynamodb_aws.DynamoDB {

	slog.Info("Creating DynamoDB client", "Region", "us-east-1")
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String("us-east-1")},
	}))

	return dynamodb_aws.New(sess)
}

// Handler is the Lambda function handler
func (f *fetchPostHandler) Handle(ctx context.Context) (string, error) {

	request := usecase.FetchPostRequest{
		BotName: "@myvideohunter.com",
	}
	err := f.fetchPostUserCase.Execute(request)
	if err != nil {
		slog.Error("Error fetching post", "error", err)
		return "Error fetching post", err
	}

	return "Post fetched", nil
}
