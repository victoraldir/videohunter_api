package dynamodb

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// isConditionFailed reports whether a write lost its ConditionExpression,
// which is how the repositories find out that a row does not exist or that a
// conditional update was rejected.
func isConditionFailed(err error) bool {
	var awsError awserr.Error

	if errors.As(err, &awsError) {
		return awsError.Code() == dynamodb.ErrCodeConditionalCheckFailedException
	}

	return false
}
