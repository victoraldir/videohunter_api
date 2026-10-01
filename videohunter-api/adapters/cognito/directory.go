package cognito

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cognitoprovider "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

// UserDirectory is the part of Cognito this service needs: deleting the
// account itself, once everything held against it is gone.
type UserDirectory interface {
	DeleteUser(userId string) error
}

type directory struct {
	client     *cognitoprovider.CognitoIdentityProvider
	userPoolID string
}

func NewUserDirectory(client *cognitoprovider.CognitoIdentityProvider, userPoolID string) UserDirectory {
	return &directory{client: client, userPoolID: userPoolID}
}

// NewCognitoClient creates the Cognito client used by the account endpoints.
func NewCognitoClient(region string) *cognitoprovider.CognitoIdentityProvider {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	}))

	return cognitoprovider.New(sess)
}

// DeleteUser removes the account. The pool signs users in by email, so the
// generated username is the Cognito subject: the same identifier everything
// else is keyed on.
func (d *directory) DeleteUser(userId string) error {
	_, err := d.client.AdminDeleteUser(&cognitoprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(d.userPoolID),
		Username:   aws.String(userId),
	})
	if err != nil {
		var notFound *cognitoprovider.UserNotFoundException
		if errors.As(err, &notFound) {
			// Already gone. Deleting an account twice is not a failure.
			return nil
		}

		return err
	}

	return nil
}
