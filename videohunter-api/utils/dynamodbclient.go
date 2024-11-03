package utils

import (
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/config"
)

func CreateLocalDynamodbClient(config config.Configuration) *dynamodb_aws.DynamoDB {

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
