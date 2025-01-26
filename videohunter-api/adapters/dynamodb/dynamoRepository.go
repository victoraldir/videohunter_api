package dynamodb

import (
	"log/slog"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/utils"
)

//go:generate mockgen -destination=../dynamodb/mocks/mockDynamodDBClient.go -package=dynamodb github.com/victoraldir/myvideohunterapi/adapters/dynamodb DynamodDBClient
type DynamodDBClient interface {
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}

type dynamodbVideoRepository struct {
	client         DynamodDBClient
	videoTableName string
}

func NewDynamodbVideoRepository(client DynamodDBClient, videoTableName string) dynamodbVideoRepository {
	return dynamodbVideoRepository{client, videoTableName}
}

func (d dynamodbVideoRepository) SaveVideo(video *domain.Video) (*domain.Video, error) {

	// We might get a collision here. I will stick with this for now.
	video.IdDB = utils.Base64Encode(video.OriginalVideoUrl)

	variants := make(map[string]*dynamodb.AttributeValue)

	media := video.GetMedia()
	text := video.GetText()

	for _, variant := range media.VideoInfo.Variants {

		variantDb := variant

		if variantDb.ContentType == "video/mp4" {

			variants[variant.URL] = &dynamodb.AttributeValue{
				M: map[string]*dynamodb.AttributeValue{
					// "bitrate": { // TODO - this is not working
					// 	N: &variant.Bitrate,
					// },
					"url": {
						S: &variantDb.URL,
					},
					"content_type": {
						S: &variantDb.ContentType,
					},
				},
			}
		}
	}

	output, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: &d.videoTableName,
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &video.IdDB,
			},
			"originalVideoUrl": {
				S: &video.OriginalVideoUrl,
			},
			"originalId": {
				S: &video.OriginalId,
			},
			"text": {
				S: &text,
			},
			"thumbnailUrl": {
				S: &video.ThumbnailUrl,
			},
			"createdAt": {
				S: &video.CreatedAt,
			},
			"variants": {
				M: variants,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	slog.Debug("DynamoDB SaveVideo output: ", "output", output)

	return video, nil
}

func (d dynamodbVideoRepository) GetVideo(videoId string) (*domain.Video, error) {

	output, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: &d.videoTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &videoId,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if output == nil {
		slog.Debug("DynamoDB GetVideo output is nil")
		return nil, nil
	}

	if output.Item == nil {
		slog.Debug("DynamoDB GetVideo output.Item is nil")
		return nil, nil
	}

	slog.Debug("DynamoDB GetVideo output: ", "output", output)
	video := &domain.Video{}
	video.IdDB = *output.Item["id"].S
	video.OriginalVideoUrl = *output.Item["originalVideoUrl"].S
	video.Text = *output.Item["text"].S
	video.CreatedAt = *output.Item["createdAt"].S
	video.ExtendedEntities.Media = make([]domain.Media, 1)
	video.ExtendedEntities.Media[0].MediaUrl = *output.Item["originalVideoUrl"].S
	video.ExtendedEntities.Media[0].VideoInfo.Variants = make([]domain.Variants, len(output.Item["variants"].M))
	video.ThumbnailUrl = *output.Item["thumbnailUrl"].S

	idx := 0
	for _, variant := range output.Item["variants"].M {
		video.ExtendedEntities.Media[0].VideoInfo.Variants[idx].URL = *variant.M["url"].S
		video.ExtendedEntities.Media[0].VideoInfo.Variants[idx].ContentType = *variant.M["content_type"].S
		idx++
	}

	return video, nil

}
