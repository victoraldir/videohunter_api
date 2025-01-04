package domain

type KeySetting string

const (
	AuthToken             KeySetting = "auth_token"
	BskyLastExecutionTime KeySetting = "bsky_last_execution_time"
	BskyAccessToken       KeySetting = "bsky_access_token"
	BskyRefreshToken      KeySetting = "bsky_refresh_token"
)

type Settings struct {
	KeySetting string `json:"key"`
	Value      string `json:"value"`
}
