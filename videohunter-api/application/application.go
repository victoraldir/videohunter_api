package application

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/ffmpeg"
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
