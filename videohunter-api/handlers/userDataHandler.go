package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/auth"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

// folderNameMaxLength matches what the library screen shows without wrapping.
const folderNameMaxLength = 60

// maxDescribedVideos bounds the per-video lookups behind listing a library.
// The library is a browsing aid, not a mirror of the video table, and an
// unbounded loop would make a single request arbitrarily slow.
const maxDescribedVideos = 60

// UserDataHandler serves everything a signed in user owns: the folders of
// saved videos, and the chat block list. All of it is reached over the
// existing REST API, behind a manually verified token.
//
// Every route here is additive. Nothing in this handler can affect a download
// or a video page, which are served by their own functions.
type UserDataHandler struct {
	Verifier TokenVerifier
	Library  repositories.UserDataRepository
	Videos   repositories.VideoRepository
}

func NewUserDataHandler(verifier TokenVerifier, library repositories.UserDataRepository, videos repositories.VideoRepository) *UserDataHandler {
	return &UserDataHandler{
		Verifier: verifier,
		Library:  library,
		Videos:   videos,
	}
}

func (h *UserDataHandler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	claims, err := h.Verifier.Verify(auth.BearerToken(headerValue(request.Headers, "Authorization")))
	if err != nil {
		slog.Info("Rejecting an unauthenticated account request", "resource", request.Resource, "error", err)
		return unauthorizedResponse(), nil
	}

	switch request.Resource {
	case "/me":
		return h.me(claims), nil

	case "/me/folders":
		if request.HTTPMethod == http.MethodPost {
			return h.createFolder(claims, request), nil
		}
		return h.listFolders(claims), nil

	case "/me/folders/{folderId}":
		if request.HTTPMethod == http.MethodPatch {
			return h.renameFolder(claims, request), nil
		}
		return h.deleteFolder(claims, request), nil

	case "/me/folders/{folderId}/videos":
		return h.saveVideo(claims, request), nil

	case "/me/folders/{folderId}/videos/{videoId}":
		return h.deleteVideo(claims, request), nil

	case "/me/blocks":
		if request.HTTPMethod == http.MethodPost {
			return h.blockUser(claims, request), nil
		}
		return h.listBlocks(claims), nil

	case "/me/blocks/{blockedUserId}":
		return h.unblockUser(claims, request), nil
	}

	return jsonResponse(http.StatusNotFound, "Unknown account route."), nil
}

type meResponse struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
}

func (h *UserDataHandler) me(claims *auth.Claims) events.APIGatewayProxyResponse {
	return jsonValue(http.StatusOK, meResponse{UserId: claims.Sub, Name: claims.DisplayName()})
}

func (h *UserDataHandler) listFolders(claims *auth.Claims) events.APIGatewayProxyResponse {

	folders, err := h.Library.ListFolders(claims.Sub)
	if err != nil {
		slog.Error("Could not list folders", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not load your library. Please try again.")
	}

	h.describeVideos(folders)

	return jsonValue(http.StatusOK, map[string]interface{}{"folders": folders})
}

// describeVideos fills in the thumbnail and the post text of the saved videos
// so the library page can render them as cards. A video that has since been
// removed from the video table is left as a bare id and stays deletable.
func (h *UserDataHandler) describeVideos(folders []domain.Folder) {

	described := 0

	for folderIndex := range folders {
		for videoIndex := range folders[folderIndex].Videos {
			if described >= maxDescribedVideos {
				return
			}
			described++

			videoId := folders[folderIndex].Videos[videoIndex].VideoId

			video, err := h.Videos.GetVideo(videoId)
			if err != nil {
				slog.Warn("Could not describe a saved video", "videoId", videoId, "error", err)
				continue
			}

			if video == nil {
				continue
			}

			folders[folderIndex].Videos[videoIndex].ThumbnailUrl = video.ThumbnailUrl
			folders[folderIndex].Videos[videoIndex].Description = video.Text
		}
	}
}

func (h *UserDataHandler) createFolder(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeBody(request.Body, &payload); err != nil {
		return jsonResponse(http.StatusBadRequest, "Give the folder a name.")
	}

	name, ok := validFolderName(payload.Name)
	if !ok {
		return jsonResponse(http.StatusBadRequest, "Folder names are 1 to 60 characters long.")
	}

	folder, err := h.Library.CreateFolder(claims.Sub, name)
	if err != nil {
		slog.Error("Could not create folder", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not create that folder. Please try again.")
	}

	return jsonValue(http.StatusCreated, map[string]interface{}{"folder": folder})
}

func (h *UserDataHandler) renameFolder(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	folderId := request.PathParameters["folderId"]

	name, ok := validFolderName(requestFolderName(request))
	if !ok {
		return jsonResponse(http.StatusBadRequest, "Folder names are 1 to 60 characters long.")
	}

	if err := h.Library.RenameFolder(claims.Sub, folderId, name); err != nil {
		return h.folderError("rename", err)
	}

	return noContentResponse()
}

func (h *UserDataHandler) deleteFolder(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	if err := h.Library.DeleteFolder(claims.Sub, request.PathParameters["folderId"]); err != nil {
		return h.folderError("delete", err)
	}

	return noContentResponse()
}

func (h *UserDataHandler) saveVideo(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	var payload struct {
		VideoId string `json:"video_id"`
	}

	if err := decodeBody(request.Body, &payload); err != nil || strings.TrimSpace(payload.VideoId) == "" {
		return jsonResponse(http.StatusBadRequest, "Which video should be saved?")
	}

	videoId := strings.TrimSpace(payload.VideoId)

	// Only ids the API has already resolved can be saved, so a library cannot
	// be filled with links that never worked.
	video, err := h.Videos.GetVideo(videoId)
	if err != nil {
		slog.Error("Could not check the video", "videoId", videoId, "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not save that video. Please try again.")
	}

	if video == nil {
		return jsonResponse(http.StatusNotFound, "That video no longer exists.")
	}

	if err := h.Library.SaveVideoToFolder(claims.Sub, request.PathParameters["folderId"], videoId); err != nil {
		return h.folderError("save video", err)
	}

	return noContentResponse()
}

func (h *UserDataHandler) deleteVideo(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	err := h.Library.DeleteVideoFromFolder(
		claims.Sub,
		request.PathParameters["folderId"],
		request.PathParameters["videoId"],
	)
	if err != nil {
		slog.Error("Could not remove a saved video", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not update that folder. Please try again.")
	}

	return noContentResponse()
}

func (h *UserDataHandler) listBlocks(claims *auth.Claims) events.APIGatewayProxyResponse {

	blocked, err := h.Library.ListBlockedUsers(claims.Sub)
	if err != nil {
		slog.Error("Could not list blocked users", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not load your blocked users. Please try again.")
	}

	return jsonValue(http.StatusOK, map[string]interface{}{"blocks": blocked})
}

func (h *UserDataHandler) blockUser(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	var payload struct {
		UserId string `json:"user_id"`
	}

	if err := decodeBody(request.Body, &payload); err != nil || strings.TrimSpace(payload.UserId) == "" {
		return jsonResponse(http.StatusBadRequest, "Which user should be blocked?")
	}

	blockedUserId := strings.TrimSpace(payload.UserId)

	if blockedUserId == claims.Sub {
		return jsonResponse(http.StatusBadRequest, "You cannot block yourself.")
	}

	if err := h.Library.BlockUser(claims.Sub, blockedUserId); err != nil {
		slog.Error("Could not block a user", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not block that user. Please try again.")
	}

	return noContentResponse()
}

func (h *UserDataHandler) unblockUser(claims *auth.Claims, request events.APIGatewayProxyRequest) events.APIGatewayProxyResponse {

	if err := h.Library.UnblockUser(claims.Sub, request.PathParameters["blockedUserId"]); err != nil {
		slog.Error("Could not unblock a user", "error", err)
		return jsonResponse(http.StatusInternalServerError, "We could not unblock that user. Please try again.")
	}

	return noContentResponse()
}

func (h *UserDataHandler) folderError(action string, err error) events.APIGatewayProxyResponse {

	if errors.Is(err, repositories.ErrFolderNotFound) {
		return jsonResponse(http.StatusNotFound, "That folder no longer exists.")
	}

	slog.Error("Could not "+action+" folder", "error", err)
	return jsonResponse(http.StatusInternalServerError, "We could not update your library. Please try again.")
}

// validFolderName trims a name and checks its length. It also strips control
// characters, which have no reason to appear in a folder name.
func validFolderName(raw string) (string, bool) {

	name := sanitizeText(raw)

	if name == "" || len([]rune(name)) > folderNameMaxLength {
		return "", false
	}

	return name, true
}

func requestFolderName(request events.APIGatewayProxyRequest) string {
	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeBody(request.Body, &payload); err != nil {
		return ""
	}

	return payload.Name
}
