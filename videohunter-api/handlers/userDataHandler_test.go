package handlers

import (
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoraldir/myvideohunterapi/auth"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

type stubVerifier struct {
	claims *auth.Claims
	err    error
	tokens []string
}

func (s *stubVerifier) Verify(token string) (*auth.Claims, error) {
	s.tokens = append(s.tokens, token)
	return s.claims, s.err
}

// stubUserData records what the handler asked for and answers with whatever
// the test configured, so the tests assert behaviour rather than DynamoDB.
type stubUserData struct {
	folders        []domain.Folder
	listErr        error
	createdFolder  *domain.Folder
	createErr      error
	renames        []string
	renameErr      error
	deletes        []string
	deleteErr      error
	saves          []string
	saveErr        error
	videoDeletes   []string
	videoDeleteErr error
	blocks         []string
	blocksErr      error
	blocked        []string
	blockErr       error
	unblocks       []string
	unblockErr     error
}

func (s *stubUserData) ListFolders(string) ([]domain.Folder, error) { return s.folders, s.listErr }

func (s *stubUserData) CreateFolder(string, string) (*domain.Folder, error) {
	return s.createdFolder, s.createErr
}

func (s *stubUserData) RenameFolder(_ string, folderId, name string) error {
	s.renames = append(s.renames, folderId+"/"+name)
	return s.renameErr
}

func (s *stubUserData) DeleteFolder(_ string, folderId string) error {
	s.deletes = append(s.deletes, folderId)
	return s.deleteErr
}

func (s *stubUserData) SaveVideoToFolder(_ string, folderId, videoId string) error {
	s.saves = append(s.saves, folderId+"/"+videoId)
	return s.saveErr
}

func (s *stubUserData) DeleteVideoFromFolder(_ string, folderId, videoId string) error {
	s.videoDeletes = append(s.videoDeletes, folderId+"/"+videoId)
	return s.videoDeleteErr
}

func (s *stubUserData) ListBlockedUsers(string) ([]string, error) { return s.blocks, s.blocksErr }

func (s *stubUserData) BlockUser(_ string, blockedUserId string) error {
	s.blocked = append(s.blocked, blockedUserId)
	return s.blockErr
}

func (s *stubUserData) UnblockUser(_ string, blockedUserId string) error {
	s.unblocks = append(s.unblocks, blockedUserId)
	return s.unblockErr
}

type stubVideos struct {
	video  *domain.Video
	videos map[string]*domain.Video
	err    error
	asked  []string
}

func (s *stubVideos) SaveVideo(video *domain.Video) (*domain.Video, error) { return video, nil }

func (s *stubVideos) GetVideo(videoId string) (*domain.Video, error) {
	s.asked = append(s.asked, videoId)

	if s.err != nil {
		return nil, s.err
	}

	if video, ok := s.videos[videoId]; ok {
		return video, nil
	}

	return s.video, nil
}

func signedIn() *stubVerifier {
	return &stubVerifier{claims: &auth.Claims{Sub: "user-1", Email: "victor@example.com", Name: "Victor"}}
}

func request(resource, method string, headers map[string]string, body string, pathParameters map[string]string) events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		Resource:       resource,
		HTTPMethod:     method,
		Headers:        headers,
		Body:           body,
		PathParameters: pathParameters,
	}
}

func authHeader() map[string]string {
	return map[string]string{"Authorization": "Bearer a-token"}
}

func TestUserDataHandler_RejectsRequestsWithoutAValidToken(t *testing.T) {

	library := &stubUserData{}
	verifier := &stubVerifier{err: errors.New("expired")}

	handler := NewUserDataHandler(verifier, library, &stubVideos{})

	for name, headers := range map[string]map[string]string{
		"no header":    {},
		"other header": {"Authorization": "Bearer nonsense"},
	} {
		t.Run(name, func(t *testing.T) {
			response, err := handler.Handle(request("/me/folders", "GET", headers, "", nil))

			require.NoError(t, err)
			assert.Equal(t, 401, response.StatusCode)
			assert.Contains(t, response.Body, "Please log in")
		})
	}

	// Nothing was reached, so nothing in the user's library moved.
	assert.Empty(t, library.renames)
	assert.Empty(t, library.deletes)
}

func TestUserDataHandler_ReturnsTheSignedInUser(t *testing.T) {

	handler := NewUserDataHandler(signedIn(), &stubUserData{}, &stubVideos{})

	response, err := handler.Handle(request("/me", "GET", authHeader(), "", nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Contains(t, response.Body, `"user_id":"user-1"`)
	// The identifier is safe to send back; the email is not echoed.
	assert.Contains(t, response.Body, `"name":"Victor"`)
	assert.NotContains(t, response.Body, "victor@example.com")
}

func TestUserDataHandler_ListsFolders(t *testing.T) {

	library := &stubUserData{folders: []domain.Folder{{Id: "f1", Name: "Music", Videos: []domain.SavedVideo{}}}}
	handler := NewUserDataHandler(signedIn(), library, &stubVideos{})

	response, err := handler.Handle(request("/me/folders", "GET", authHeader(), "", nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Contains(t, response.Body, `"folders":[{"id":"f1","name":"Music"`)
}

func TestUserDataHandler_DescribesSavedVideosSoTheLibraryCanDrawCards(t *testing.T) {

	library := &stubUserData{folders: []domain.Folder{{
		Id:   "f1",
		Name: "Music",
		Videos: []domain.SavedVideo{
			{VideoId: "video-a"},
			// This one is gone from the video table: it stays listed, with
			// only its id, so it can still be removed from the folder.
			{VideoId: "video-gone"},
		},
	}}}

	videos := &stubVideos{videos: map[string]*domain.Video{
		"video-a": {IdDB: "video-a", ThumbnailUrl: "https://cdn.example.com/a.jpg", Text: "A post"},
	}}

	handler := NewUserDataHandler(signedIn(), library, videos)

	response, err := handler.Handle(request("/me/folders", "GET", authHeader(), "", nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Contains(t, response.Body, `"video_id":"video-a","saved_at":"","thumbnail_url":"https://cdn.example.com/a.jpg","description":"A post"`)
	assert.Contains(t, response.Body, `"video_id":"video-gone"`)
	assert.Equal(t, []string{"video-a", "video-gone"}, videos.asked)
}

func TestUserDataHandler_CreatesAFolder(t *testing.T) {

	library := &stubUserData{createdFolder: &domain.Folder{Id: "f9", Name: "Music", Videos: []domain.SavedVideo{}}}
	handler := NewUserDataHandler(signedIn(), library, &stubVideos{})

	response, err := handler.Handle(request("/me/folders", "POST", authHeader(), `{"name":"  Music  "}`, nil))

	require.NoError(t, err)
	assert.Equal(t, 201, response.StatusCode)
	assert.Contains(t, response.Body, `"folder":{"id":"f9"`)
}

func TestUserDataHandler_RejectsEmptyAndOverlongFolderNames(t *testing.T) {

	handler := NewUserDataHandler(signedIn(), &stubUserData{}, &stubVideos{})

	for name, body := range map[string]string{
		"empty":     `{"name":"   "}`,
		"control":   `{"name":"\u0007\u0000"}`,
		"too long":  `{"name":"` + string(make([]rune, 61)) + `"}`,
		"no body":   ``,
		"bad json":  `{`,
		"not a map": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			response, err := handler.Handle(request("/me/folders", "POST", authHeader(), body, nil))

			require.NoError(t, err)
			assert.Equal(t, 400, response.StatusCode)
		})
	}
}

func TestUserDataHandler_RenamesAndDeletesFolders(t *testing.T) {

	library := &stubUserData{}
	handler := NewUserDataHandler(signedIn(), library, &stubVideos{})

	renamed, err := handler.Handle(request("/me/folders/{folderId}", "PATCH", authHeader(),
		`{"name":"Live sets"}`, map[string]string{"folderId": "f1"}))
	require.NoError(t, err)
	assert.Equal(t, 204, renamed.StatusCode)
	assert.Equal(t, []string{"f1/Live sets"}, library.renames)

	deleted, err := handler.Handle(request("/me/folders/{folderId}", "DELETE", authHeader(), "",
		map[string]string{"folderId": "f2"}))
	require.NoError(t, err)
	assert.Equal(t, 204, deleted.StatusCode)
	assert.Equal(t, []string{"f2"}, library.deletes)
}

func TestUserDataHandler_MapsAMissingFolderTo404(t *testing.T) {

	library := &stubUserData{deleteErr: repositories.ErrFolderNotFound}
	handler := NewUserDataHandler(signedIn(), library, &stubVideos{})

	response, err := handler.Handle(request("/me/folders/{folderId}", "DELETE", authHeader(), "",
		map[string]string{"folderId": "gone"}))

	require.NoError(t, err)
	assert.Equal(t, 404, response.StatusCode)
}

func TestUserDataHandler_SavesOnlyVideosThatExist(t *testing.T) {

	videos := &stubVideos{}
	library := &stubUserData{}
	handler := NewUserDataHandler(signedIn(), library, videos)

	// A link that never resolved cannot be saved.
	missing, err := handler.Handle(request("/me/folders/{folderId}/videos", "POST", authHeader(),
		`{"video_id":"unknown"}`, map[string]string{"folderId": "f1"}))
	require.NoError(t, err)
	assert.Equal(t, 404, missing.StatusCode)
	assert.Empty(t, library.saves)

	videos.video = &domain.Video{IdDB: "abc123"}

	saved, err := handler.Handle(request("/me/folders/{folderId}/videos", "POST", authHeader(),
		`{"video_id":"abc123"}`, map[string]string{"folderId": "f1"}))
	require.NoError(t, err)
	assert.Equal(t, 204, saved.StatusCode)
	assert.Equal(t, []string{"f1/abc123"}, library.saves)

	removed, err := handler.Handle(request("/me/folders/{folderId}/videos/{videoId}", "DELETE", authHeader(), "",
		map[string]string{"folderId": "f1", "videoId": "abc123"}))
	require.NoError(t, err)
	assert.Equal(t, 204, removed.StatusCode)
	assert.Equal(t, []string{"f1/abc123"}, library.videoDeletes)
}

func TestUserDataHandler_BlocksAndUnblocksUsers(t *testing.T) {

	library := &stubUserData{blocks: []string{"user-2"}}
	handler := NewUserDataHandler(signedIn(), library, &stubVideos{})

	listed, err := handler.Handle(request("/me/blocks", "GET", authHeader(), "", nil))
	require.NoError(t, err)
	assert.Contains(t, listed.Body, `"blocks":["user-2"]`)

	blocked, err := handler.Handle(request("/me/blocks", "POST", authHeader(), `{"user_id":"user-2"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 204, blocked.StatusCode)
	assert.Equal(t, []string{"user-2"}, library.blocked)

	// Blocking yourself would hide your own messages, so it is refused.
	itself, err := handler.Handle(request("/me/blocks", "POST", authHeader(), `{"user_id":"user-1"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 400, itself.StatusCode)

	unblocked, err := handler.Handle(request("/me/blocks/{blockedUserId}", "DELETE", authHeader(), "",
		map[string]string{"blockedUserId": "user-2"}))
	require.NoError(t, err)
	assert.Equal(t, 204, unblocked.StatusCode)
	assert.Equal(t, []string{"user-2"}, library.unblocks)
}

func TestUserDataHandler_UnknownRouteIs404(t *testing.T) {

	handler := NewUserDataHandler(signedIn(), &stubUserData{}, &stubVideos{})

	response, err := handler.Handle(request("/me/something-else", "GET", authHeader(), "", nil))

	require.NoError(t, err)
	assert.Equal(t, 404, response.StatusCode)
}

func TestValidFolderName(t *testing.T) {
	_, ok := validFolderName("   ")
	assert.False(t, ok)

	name, ok := validFolderName("  Road trip  ")
	assert.True(t, ok)
	assert.Equal(t, "Road trip", name)

	_, ok = validFolderName(string(make([]rune, folderNameMaxLength+1)))
	assert.False(t, ok)

	name, ok = validFolderName("a\u0000b")
	assert.True(t, ok)
	assert.Equal(t, "ab", name)
}
