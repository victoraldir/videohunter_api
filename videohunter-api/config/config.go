package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v6"
)

type LogLevel string

/*
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
*/

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type Environment string

const (
	Local Environment = "local"
	Live  Environment = "live"
)

type Configuration struct {
	VideoTableName     string      `env:"VIDEO_TABLE"`
	SettingsTableName  string      `env:"SETTINGS_TABLE"`
	LogLevel           LogLevel    `env:"LOG_LEVEL"`
	AwsApiKey          string      `env:"AWS_ACCESS_KEY_ID"`
	AwsSecretAccessKey string      `env:"AWS_SECRET_ACCESS_KEY"`
	LocalDynamodbAddr  string      `env:"LOCAL_DYNAMODB_ADDR"`
	Region             string      `env:"REGION"`
	Environment        Environment `env:"ENVIRONMENT"`
}

var Config Configuration

func Init() {
	loadEnv()
	InitLogger()
}

func InitLogger() {

	var logLevel slog.Level

	switch Config.LogLevel {
	case DEBUG:
		logLevel = slog.LevelDebug
	case INFO:
		logLevel = slog.LevelInfo
	case WARN:
		logLevel = slog.LevelWarn
	case ERROR:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	if Config.Environment != Local {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: true,
		}))

		slog.SetDefault(logger)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))

	slog.SetDefault(logger)
}

func loadEnv() {
	Config = Configuration{}

	err := env.Parse(&Config)

	if err != nil {
		fatalError := fmt.Errorf("error loading environment variables: %v", err)
		panic(fatalError)
	}

	if Config.LogLevel == "" {
		Config.LogLevel = INFO
	}

	if Config.Region == "" {
		Config.Region = "us-east-1"
	}

	if Config.Environment == "" {
		Config.Environment = "prod"
	}

	if Config.Environment == Local {
		Config.AwsApiKey = "dummykey"
		Config.AwsSecretAccessKey = "dummysecret"
		Config.LocalDynamodbAddr = "http://dynamodb:8000"
	}

	slog.Info("Config: ", "config", Config)
}
