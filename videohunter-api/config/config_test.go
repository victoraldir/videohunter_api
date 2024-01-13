package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_LoadEnv(t *testing.T) {
	t.Run("should load env variables", func(t *testing.T) {

		// Arrange
		videoTable := "video_table"
		logLevel := INFO

		os.Setenv("VIDEO_TABLE", videoTable)
		os.Setenv("LOG_LEVEL", string(logLevel))

		// Act
		Init()

		// Assert
		assert.NotNil(t, Config)
		assert.Equal(t, videoTable, Config.VideoTableName)
		assert.Equal(t, INFO, Config.LogLevel)
	})
}
