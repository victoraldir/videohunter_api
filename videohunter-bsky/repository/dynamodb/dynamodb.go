package dynamodb

import "github.com/victoraldir/myvideohunterbsky/domain"

type DynamodbRepository interface {
	SaveLastPostScanTime(time string) error
	GetLastPostScanTime() (string, error)
	SavePost(post domain.Post) error
	GetPostByCid(cid string) (domain.Post, error)
}

type dynamodbRepository struct {
	postTable     map[string]domain.Post
	lastScanTable map[string]string
}

func NewDynamodbRepository() *dynamodbRepository {
	return &dynamodbRepository{
		postTable:     make(map[string]domain.Post),
		lastScanTable: map[string]string{},
	}
}

func (d *dynamodbRepository) SaveLastPostScanTime(time string) error {
	d.lastScanTable["lastScanTime_bsky"] = time
	return nil
}

func (d *dynamodbRepository) GetLastPostScanTime() (string, error) {
	return d.lastScanTable["lastScanTime_bsky"], nil
}

func (d *dynamodbRepository) GetPostByCid(cid string) (domain.Post, error) {
	return d.postTable[cid], nil
}

func (d *dynamodbRepository) SavePost(post domain.Post) error {
	d.postTable[post.Cid] = post
	return nil
}
