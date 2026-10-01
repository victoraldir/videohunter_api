package apigateway

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	management "github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
)

// ConnectionManager posts a payload back to one browser websocket.
type ConnectionManager interface {
	PostToConnection(connectionId string, payload []byte) error
}

// ErrConnectionGone means the browser is no longer there: it closed the tab,
// lost the network, or the connection expired. The caller stops counting that
// connection and prunes it.
var ErrConnectionGone = errors.New("connection gone")

type managementClient struct {
	api *management.ApiGatewayManagementApi
}

// NewConnectionManager points the management API at the websocket stage that
// owns the connection. The Lambda learns that endpoint from the event itself
// (requestContext.domainName and stage), so there is nothing to configure.
func NewConnectionManager(domainName, stage string) ConnectionManager {
	endpoint := fmt.Sprintf("https://%s/%s", domainName, stage)

	return &managementClient{
		api: management.New(
			session.Must(session.NewSession()),
			aws.NewConfig().WithEndpoint(endpoint),
		),
	}
}

func (m *managementClient) PostToConnection(connectionId string, payload []byte) error {
	_, err := m.api.PostToConnection(&management.PostToConnectionInput{
		ConnectionId: aws.String(connectionId),
		Data:         payload,
	})
	if err != nil {
		var gone *management.GoneException
		if errors.As(err, &gone) {
			return ErrConnectionGone
		}
		return err
	}

	return nil
}
