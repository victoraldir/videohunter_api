package application

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/apigateway"
	"github.com/victoraldir/myvideohunterapi/adapters/cognito"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/ffmpeg"
	"github.com/victoraldir/myvideohunterapi/auth"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
	"github.com/victoraldir/myvideohuntershared/services/reddit"
	"github.com/victoraldir/myvideohuntershared/services/twitter"

	config_api "github.com/victoraldir/myvideohunterapi/config"
	"github.com/victoraldir/myvideohunterapi/handlers"
	"github.com/victoraldir/myvideohunterapi/usecases"
)

type LambdaAPIGatewayApplication struct {
	CreateUrlHandler        *handlers.CreateUrlHandler
	CreateUrlBatchHandler   handlers.CreateUrlBatchHandler
	GetUrlHandler           *handlers.GetUrlHandler
	DownloadVideoHlsHandler *handlers.DownloadVideoHlsHandler
	MixAudioVideoHandler    *handlers.MixAudioVideoHandler
}

func NewAPIGatewayHandler(config config_api.Configuration) *LambdaAPIGatewayApplication {

	// Clients
	httpClient := &http.Client{}

	var client dynamodb.DynamodDBClient

	if config.Environment == config_api.Local {
		client = CreateLocalDynamodbClient(config)
	} else {
		client = createDynamodbClient(config)
	}

	// Repositories
	videoRepository := dynamodb.NewDynamodbVideoRepository(client, config.VideoTableName)
	settingsRepository := dynamodb.NewDynamoSettingsRepository(client, config.SettingsTableName)
	twitterRepository := twitter.NewTwitterDownloaderRepository(httpClient)
	redditDownloaderRepository := reddit.NewRedditDownloaderRepository(httpClient)
	downloadVideoHlsRepository := ffmpeg.NewDownloaderHlsRepository()
	bskyRepository := bsky.NewBskyService(httpClient, "", "")
	socialNetworkRepository := bsky.NewBskyService(httpClient, "", "")

	// Use Cases
	videoDownloaderUseCase := usecases.NewVideoDownloaderUseCase(
		videoRepository,
		twitterRepository,
		redditDownloaderRepository,
		bskyRepository,
		settingsRepository,
	)

	// redditDownloaderUseCase := usecases.NewRedditVideoDownloaderUseCase(videoRepository, redditDownloaderRepository)

	downloadVideoHlsUseCase := usecases.NewDownloadVideoHlsUseCase(
		videoRepository,
		downloadVideoHlsRepository,
	)

	mixAudioVideoUseCase := usecases.NewMixAudioVideoUseCase(downloadVideoHlsRepository)

	createUrlBatchUseCase := usecases.NewCreateUrlBatchUseCase(socialNetworkRepository, videoRepository)

	// Handlers
	createUrlHandler := &handlers.CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCase,
		// RedditDownloaderUseCase: redditDownloaderUseCase,
	}

	getUrlHandler := &handlers.GetUrlHandler{
		GerUrlUseCase: usecases.NewGetUrlUseCase(videoRepository),
	}

	downloadVideoHlsHandler := handlers.NewDownloadVideoHlsHandler(downloadVideoHlsUseCase)

	mixAudioVideoHandler := handlers.NewMixAudioVideoHandler(mixAudioVideoUseCase)

	createUrlBatchHandler := handlers.NewCreateUrlBatchHandler(createUrlBatchUseCase)

	return &LambdaAPIGatewayApplication{
		CreateUrlHandler:        createUrlHandler,
		GetUrlHandler:           getUrlHandler,
		DownloadVideoHlsHandler: downloadVideoHlsHandler,
		MixAudioVideoHandler:    mixAudioVideoHandler,
		CreateUrlBatchHandler:   createUrlBatchHandler,
	}

}

// NewUserDataHandler builds the handler for a signed in user's library, chat
// block list and account deletion. It shares the video repository so that a
// save can only reference a video the API has already resolved.
func NewUserDataHandler(config config_api.Configuration) *handlers.UserDataHandler {
	client := dynamoClient(config)

	return handlers.NewUserDataHandler(
		auth.NewVerifier(config.CognitoRegion, config.CognitoUserPoolID, config.CognitoClientID),
		dynamodb.NewUserDataRepository(client, config.UserDataTableName),
		dynamodb.NewDynamodbVideoRepository(client, config.VideoTableName),
		dynamodb.NewChatDataRepository(client, config.ChatDataTableName),
		cognito.NewUserDirectory(cognito.NewCognitoClient(config.CognitoRegion), config.CognitoUserPoolID),
	)
}

// NewChatHandler builds the handler behind the websocket that serves the chat
// room of a video page. It shares the user data repository so that the name a
// room shows comes from the same profile the library shows.
func NewChatHandler(config config_api.Configuration) *handlers.ChatHandler {
	client := dynamoClient(config)

	return handlers.NewChatHandler(
		auth.NewVerifier(config.CognitoRegion, config.CognitoUserPoolID, config.CognitoClientID),
		dynamodb.NewChatDataRepository(client, config.ChatDataTableName),
		dynamodb.NewUserDataRepository(client, config.UserDataTableName),
		apigateway.NewConnectionManager,
	)
}

// NewConfigHandler builds the handler that publishes the public browser
// configuration (Cognito endpoints and the chat websocket URL).
func NewConfigHandler(config config_api.Configuration) *handlers.ConfigHandler {
	return handlers.NewConfigHandler(config.CognitoDomain, config.CognitoClientID, config.ChatWsEndpoint)
}

// dynamoClient returns the DynamoDB client for the current environment.
func dynamoClient(config config_api.Configuration) *dynamodb_aws.DynamoDB {
	if config.Environment == config_api.Local {
		return CreateLocalDynamodbClient(config)
	}

	return createDynamodbClient(config)
}

func CreateLocalDynamodbClient(config config_api.Configuration) *dynamodb_aws.DynamoDB {

	slog.Debug("Connecting to local DynamoDB",
		"LocalDynamodbAddr", config.LocalDynamodbAddr,
		"AwsApiKey", config.AwsApiKey,
		"AwsSecretAccessKey", config.AwsSecretAccessKey,
		"Region", config.Region)

	// Set dummy credentials
	os.Setenv("AWS_ACCESS_KEY_ID", config.AwsApiKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", config.AwsSecretAccessKey)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:   aws.String(config.Region),
			Endpoint: aws.String(config.LocalDynamodbAddr)},
	}))

	return dynamodb_aws.New(sess)
}

func createDynamodbClient(config config_api.Configuration) *dynamodb_aws.DynamoDB {

	slog.Info("Creating DynamoDB client", "Region", config.Region)
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(config.Region)},
	}))

	return dynamodb_aws.New(sess)
}
