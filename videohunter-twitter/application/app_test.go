package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohuntertwitter/usecase"
)

func TestLoadBots(t *testing.T) {
	t.Setenv("TWITTER_BOT_USERNAMES", "BaixadorDeVideo,descargarvid,@AnotherBot")
	t.Setenv("TWITTER_BOT_AUTH_TOKEN_BAIXADORDEVIDEO", "tok1")
	t.Setenv("TWITTER_BOT_AUTH_TOKEN_DESCARGARVID", "tok2")
	// @AnotherBot has no token configured and must be skipped.

	bots := loadBots()

	assert.Equal(t, []usecase.BotConfig{
		{Username: "BaixadorDeVideo", Language: "pt", AuthToken: "tok1"},
		{Username: "descargarvid", Language: "es", AuthToken: "tok2"},
	}, bots)
}

func TestLoadBots_LanguageOverride(t *testing.T) {
	t.Setenv("TWITTER_BOT_USERNAMES", "BaixadorDeVideo")
	t.Setenv("TWITTER_BOT_AUTH_TOKEN_BAIXADORDEVIDEO", "tok1")
	t.Setenv("TWITTER_BOT_LANGUAGE_BAIXADORDEVIDEO", "en")

	bots := loadBots()

	assert.Equal(t, "en", bots[0].Language)
}

func TestLoadBots_NoBots(t *testing.T) {
	t.Setenv("TWITTER_BOT_USERNAMES", "")

	assert.Empty(t, loadBots())
}

func TestDefaultLanguage(t *testing.T) {
	assert.Equal(t, "pt", defaultLanguage("BaixadorDeVideo"))
	assert.Equal(t, "es", defaultLanguage("descargarvid"))
	assert.Equal(t, "en", defaultLanguage("unknown"))
}
