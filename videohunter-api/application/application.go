package application

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/twitter"

	config_api "github.com/victoraldir/myvideohunterapi/config"
	"github.com/victoraldir/myvideohunterapi/handlers"
	"github.com/victoraldir/myvideohunterapi/usecases"
)

type LambdaAPIGatewayApplication struct {
	CreateUrlHandler *handlers.CreateUrlHandler
	GetUrlHandler    *handlers.GetUrlHandler
}

func NewAPIGatewayHandler(config config_api.Configuration) *LambdaAPIGatewayApplication {

	// Clients
	httpClient := &http.Client{}

	var client dynamodb.DynamodDBClient

	if config.Environment == config_api.Local {
		client = createLocalDynamodbClient(config)
	} else {
		client = createDynamodbClient(config)
	}

	// Repositories
	videoRepository := dynamodb.NewDynamodbVideoRepository(client, config.VideoTableName)
	settingsRepository := dynamodb.NewDynamoSettingsRepository(client, config.SettingsTableName)
	downloadeRepository := twitter.NewTwitterDownloaderRepository(httpClient)

	// Use Cases
	videoDownloaderUseCase := usecases.NewVideoDownloaderUseCase(
		videoRepository,
		downloadeRepository,
		settingsRepository,
	)

	// Handlers
	createUrlHandler := &handlers.CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCase,
	}

	getUrlHndler := &handlers.GetUrlHandler{
		GerUrlUseCase: usecases.NewGetUrlUseCase(videoRepository),
	}

	return &LambdaAPIGatewayApplication{
		CreateUrlHandler: createUrlHandler,
		GetUrlHandler:    getUrlHndler,
	}

}

func createLocalDynamodbClient(config config_api.Configuration) *dynamodb_aws.DynamoDB {

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
