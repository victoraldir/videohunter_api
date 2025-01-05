package reddit

type Post struct {
	Kind string   `json:"kind"`
	Data PostData `json:"data"`
}

type PostData struct {
	After     any     `json:"after"`
	Dist      int     `json:"dist"`
	Modhash   string  `json:"modhash"`
	GeoFilter string  `json:"geo_filter"`
	Children  []Child `json:"children"`
	Before    any     `json:"before"`
}

type Child struct {
	Kind string    `json:"kind"`
	Data ChildData `json:"data"`
}

type ChildData struct {
	ApprovedAtUtc              any              `json:"approved_at_utc"`
	Subreddit                  string           `json:"subreddit"`
	Selftext                   string           `json:"selftext"`
	UserReports                []any            `json:"user_reports"`
	Saved                      bool             `json:"saved"`
	ModReasonTitle             any              `json:"mod_reason_title"`
	Gilded                     int              `json:"gilded"`
	Clicked                    bool             `json:"clicked"`
	Title                      string           `json:"title"`
	LinkFlairRichtext          []any            `json:"link_flair_richtext"`
	SubredditNamePrefixed      string           `json:"subreddit_name_prefixed"`
	Hidden                     bool             `json:"hidden"`
	Pwls                       any              `json:"pwls"`
	LinkFlairCSSClass          any              `json:"link_flair_css_class"`
	Downs                      int              `json:"downs"`
	ThumbnailHeight            int              `json:"thumbnail_height"`
	TopAwardedType             any              `json:"top_awarded_type"`
	HideScore                  bool             `json:"hide_score"`
	Name                       string           `json:"name"`
	Quarantine                 bool             `json:"quarantine"`
	LinkFlairTextColor         string           `json:"link_flair_text_color"`
	UpvoteRatio                float64          `json:"upvote_ratio"`
	AuthorFlairBackgroundColor any              `json:"author_flair_background_color"`
	SubredditType              string           `json:"subreddit_type"`
	Ups                        int              `json:"ups"`
	TotalAwardsReceived        int              `json:"total_awards_received"`
	MediaEmbed                 MediaEmbed       `json:"media_embed"`
	ThumbnailWidth             int              `json:"thumbnail_width"`
	AuthorFlairTemplateID      any              `json:"author_flair_template_id"`
	IsOriginalContent          bool             `json:"is_original_content"`
	AuthorFullname             string           `json:"author_fullname"`
	SecureMedia                SecureMedia      `json:"secure_media"`
	IsRedditMediaDomain        bool             `json:"is_reddit_media_domain"`
	IsMeta                     bool             `json:"is_meta"`
	Category                   any              `json:"category"`
	SecureMediaEmbed           SecureMediaEmbed `json:"secure_media_embed"`
	LinkFlairText              any              `json:"link_flair_text"`
	CanModPost                 bool             `json:"can_mod_post"`
	Score                      int              `json:"score"`
	ApprovedBy                 any              `json:"approved_by"`
	IsCreatedFromAdsUI         bool             `json:"is_created_from_ads_ui"`
	AuthorPremium              bool             `json:"author_premium"`
	Thumbnail                  string           `json:"thumbnail"`
	Edited                     any              `json:"edited"`
	AuthorFlairCSSClass        any              `json:"author_flair_css_class"`
	AuthorFlairRichtext        []any            `json:"author_flair_richtext"`
	Gildings                   Gildings         `json:"gildings"`
	PostHint                   string           `json:"post_hint"`
	ContentCategories          any              `json:"content_categories"`
	IsSelf                     bool             `json:"is_self"`
	ModNote                    any              `json:"mod_note"`
	Created                    float64          `json:"created"`
	LinkFlairType              string           `json:"link_flair_type"`
	Wls                        any              `json:"wls"`
	RemovedByCategory          any              `json:"removed_by_category"`
	BannedBy                   any              `json:"banned_by"`
	AuthorFlairType            string           `json:"author_flair_type"`
	Domain                     string           `json:"domain"`
	AllowLiveComments          bool             `json:"allow_live_comments"`
	SelftextHTML               any              `json:"selftext_html"`
	Likes                      any              `json:"likes"`
	SuggestedSort              string           `json:"suggested_sort"`
	BannedAtUtc                any              `json:"banned_at_utc"`
	URLOverriddenByDest        string           `json:"url_overridden_by_dest"`
	ViewCount                  any              `json:"view_count"`
	Archived                   bool             `json:"archived"`
	NoFollow                   bool             `json:"no_follow"`
	IsCrosspostable            bool             `json:"is_crosspostable"`
	Pinned                     bool             `json:"pinned"`
	Over18                     bool             `json:"over_18"`
	Preview                    Preview          `json:"preview"`
	AllAwardings               []any            `json:"all_awardings"`
	Awarders                   []any            `json:"awarders"`
	MediaOnly                  bool             `json:"media_only"`
	CanGild                    bool             `json:"can_gild"`
	Spoiler                    bool             `json:"spoiler"`
	Locked                     bool             `json:"locked"`
	AuthorFlairText            any              `json:"author_flair_text"`
	TreatmentTags              []any            `json:"treatment_tags"`
	Visited                    bool             `json:"visited"`
	RemovedBy                  any              `json:"removed_by"`
	NumReports                 any              `json:"num_reports"`
	Distinguished              any              `json:"distinguished"`
	SubredditID                string           `json:"subreddit_id"`
	AuthorIsBlocked            bool             `json:"author_is_blocked"`
	ModReasonBy                any              `json:"mod_reason_by"`
	RemovalReason              any              `json:"removal_reason"`
	LinkFlairBackgroundColor   string           `json:"link_flair_background_color"`
	ID                         string           `json:"id"`
	IsRobotIndexable           bool             `json:"is_robot_indexable"`
	NumDuplicates              int              `json:"num_duplicates"`
	ReportReasons              any              `json:"report_reasons"`
	Author                     string           `json:"author"`
	DiscussionType             any              `json:"discussion_type"`
	NumComments                int              `json:"num_comments"`
	SendReplies                bool             `json:"send_replies"`
	Media                      Media            `json:"media"`
	ContestMode                bool             `json:"contest_mode"`
	AuthorPatreonFlair         bool             `json:"author_patreon_flair"`
	AuthorFlairTextColor       any              `json:"author_flair_text_color"`
	Permalink                  string           `json:"permalink"`
	Stickied                   bool             `json:"stickied"`
	URL                        string           `json:"url"`
	SubredditSubscribers       int              `json:"subreddit_subscribers"`
	CreatedUtc                 float64          `json:"created_utc"`
	NumCrossposts              int              `json:"num_crossposts"`
	ModReports                 []any            `json:"mod_reports"`
	IsVideo                    bool             `json:"is_video"`
	CrosspostParentList        []ChildData      `json:"crosspost_parent_list"`
}

type MediaEmbed struct {
	Content   string `json:"content"`
	Width     int    `json:"width"`
	Scrolling bool   `json:"scrolling"`
	Height    int    `json:"height"`
}

type SecureMedia struct {
	Oembed      Oembed      `json:"oembed"`
	Type        string      `json:"type"`
	RedditVideo RedditVideo `json:"reddit_video"`
}

type RedditVideo struct {
	BitrateKbps       int    `json:"bitrate_kbps"`
	FallbackURL       string `json:"fallback_url"`
	HasAudio          bool   `json:"has_audio"`
	Height            int    `json:"height"`
	Width             int    `json:"width"`
	ScrubberMediaURL  string `json:"scrubber_media_url"`
	DashURL           string `json:"dash_url"`
	Duration          int    `json:"duration"`
	HlsURL            string `json:"hls_url"`
	IsGif             bool   `json:"is_gif"`
	TranscodingStatus string `json:"transcoding_status"`
}

type Oembed struct {
	ProviderURL     string `json:"provider_url"`
	Title           string `json:"title"`
	Type            string `json:"type"`
	HTML            string `json:"html"`
	ThumbnailWidth  int    `json:"thumbnail_width"`
	Height          int    `json:"height"`
	Width           int    `json:"width"`
	Version         string `json:"version"`
	ProviderName    string `json:"provider_name"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailHeight int    `json:"thumbnail_height"`
}

type SecureMediaEmbed struct {
	Content        string `json:"content"`
	Width          int    `json:"width"`
	Scrolling      bool   `json:"scrolling"`
	MediaDomainURL string `json:"media_domain_url"`
	Height         int    `json:"height"`
}

type Gildings struct {
}

type Preview struct {
	Images             []Image     `json:"images"`
	RedditVideoPreview RedditVideo `json:"reddit_video_preview"`
	Enabled            bool        `json:"enabled"`
}

type Image struct {
	Source      Source       `json:"source"`
	Resolutions []Resolution `json:"resolutions"`
	Variants    Variants     `json:"variants"`
	ID          string       `json:"id"`
}

type Source struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Resolution struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Variants struct {
	Obfuscated Obfuscated `json:"obfuscated"`
	Nsfw       Nsfw       `json:"nsfw"`
}

type Obfuscated struct {
	Source      Source       `json:"source"`
	Resolutions []Resolution `json:"resolutions"`
}

type Nsfw struct {
	Source      Source       `json:"source"`
	Resolutions []Resolution `json:"resolutions"`
}

type Media struct {
	Oembed Oembed `json:"oembed"`
	Type   string `json:"type"`
}
