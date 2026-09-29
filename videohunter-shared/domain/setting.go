package domain

type KeySetting string

const (
	AuthToken             KeySetting = "auth_token"
	BskyLastExecutionTime KeySetting = "bsky_last_execution_time"
	BskyAccessToken       KeySetting = "bsky_access_token"
	BskyRefreshToken      KeySetting = "bsky_refresh_token"
	BskyRepliedPosts      KeySetting = "bsky_replied_posts"

	// Twitter bot settings. They are namespaced per bot account through
	// TwitterBotSettingKey so both bots can share the settings table.
	TwitterLastMentionID   KeySetting = "twitter_last_mention_id"
	TwitterRepliedMentions KeySetting = "twitter_replied_mentions"
)

// TwitterBotSettingKey builds a per-bot namespaced settings key, for example
// "twitter_last_mention_id:BaixadorDeVideo".
func TwitterBotSettingKey(prefix KeySetting, botUsername string) KeySetting {
	return KeySetting(string(prefix) + ":" + botUsername)
}

type Settings struct {
	KeySetting string `json:"key"`
	Value      string `json:"value"`
}
