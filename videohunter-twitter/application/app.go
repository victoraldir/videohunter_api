package application

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohuntershared/services/getxapi"
	"github.com/victoraldir/myvideohuntershared/services/videohunterapi"
	"github.com/victoraldir/myvideohuntertwitter/repository/dynamodb"
	"github.com/victoraldir/myvideohuntertwitter/usecase"
)

const defaultSettingsTable = "settings"

type fetchMentionsHandler struct {
	fetchMentions usecase.FetchMentions
}

func NewFetchMentionsHandler() *fetchMentionsHandler {

	// Clients
	httpClient := &http.Client{}
	dynamodbClient := createDynamodbClient()

	// Services
	getxapiService := getxapi.NewGetXAPIService(httpClient, os.Getenv("GETXAPI_KEY"))
	videoHunterApi := videohunterapi.NewVideoHunterApi(httpClient)

	// Repository
	settingsTable := os.Getenv("SETTINGS_TABLE")
	if settingsTable == "" {
		settingsTable = defaultSettingsTable
	}
	dynamodbRepo := dynamodb.NewDynamoSettingsRepository(dynamodbClient, settingsTable)

	// Use Cases
	fetchMentions := usecase.NewFetchMentions(getxapiService, videoHunterApi, dynamodbRepo)

	return &fetchMentionsHandler{fetchMentions: fetchMentions}
}

func createDynamodbClient() *dynamodb_aws.DynamoDB {

	region := os.Getenv("REGION")
	if region == "" {
		region = "us-east-1"
	}

	slog.Info("Creating DynamoDB client", "Region", region)
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region)},
	}))

	return dynamodb_aws.New(sess)
}

// Handle is the Lambda function handler.
func (f *fetchMentionsHandler) Handle(ctx context.Context) (string, error) {

	request := usecase.FetchMentionsRequest{
		Bots: loadBots(),
	}

	if len(request.Bots) == 0 {
		slog.Warn("No twitter bots configured, nothing to do")
		return "No bots configured", nil
	}

	if err := f.fetchMentions.Execute(request); err != nil {
		slog.Error("Error fetching mentions", slog.Any("error", err))
		return "Error fetching mentions", err
	}

	return "Mentions fetched", nil
}

// loadBots builds the bot list from the environment. Bots are declared through
// TWITTER_BOT_USERNAMES (comma separated). Each bot reads its auth token from
// TWITTER_BOT_AUTH_TOKEN_<UPPERCASE_USERNAME> and its optional language from
// TWITTER_BOT_LANGUAGE_<UPPERCASE_USERNAME>.
func loadBots() []usecase.BotConfig {
	bots := make([]usecase.BotConfig, 0)

	for _, rawUsername := range strings.Split(os.Getenv("TWITTER_BOT_USERNAMES"), ",") {
		username := strings.TrimSpace(strings.TrimPrefix(rawUsername, "@"))
		if username == "" {
			continue
		}

		envSuffix := strings.ToUpper(username)

		authToken := os.Getenv("TWITTER_BOT_AUTH_TOKEN_" + envSuffix)
		if authToken == "" {
			slog.Warn("Auth token not found for bot, skipping", slog.String("bot", username))
			continue
		}

		language := os.Getenv("TWITTER_BOT_LANGUAGE_" + envSuffix)
		if language == "" {
			language = defaultLanguage(username)
		}

		bots = append(bots, usecase.BotConfig{
			Username:  username,
			Language:  language,
			AuthToken: authToken,
		})
	}

	return bots
}

func defaultLanguage(username string) string {
	switch strings.ToLower(username) {
	case "baixadordevideo":
		return "pt"
	case "descargarvid":
		return "es"
	default:
		return "en"
	}
}
