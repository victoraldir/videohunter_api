package handlers

import (
	"encoding/json"
	"log"

	events_aws "github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
)

type CreateUrlBatchHandler struct {
	createUrlBatchUseCase usecases.CreateUrlBatchUseCase
}

type VideoBatchRequest struct {
	Uris []string `json:"uris"`
}

func NewCreateUrlBatchHandler(createUrlBatchUseCase usecases.CreateUrlBatchUseCase) CreateUrlBatchHandler {
	return CreateUrlBatchHandler{
		createUrlBatchUseCase: createUrlBatchUseCase,
	}
}

func (h *CreateUrlBatchHandler) Handle(request events_aws.APIGatewayProxyRequest) (events_aws.APIGatewayProxyResponse, error) {

	videoBatchRequest := &VideoBatchRequest{}

	log.Println("Request body: ", request.Body)

	body := request.Body

	err := json.Unmarshal([]byte(body), videoBatchRequest)
	if err != nil {
		log.Println("Error unmarshalling request body", err)
	}

	responses, err := h.createUrlBatchUseCase.Execute(videoBatchRequest.Uris)
	if err != nil {
		log.Println("Error executing createUrlBatchUseCase", err)
	}

	responseBody, err := json.Marshal(responses)
	if err != nil {
		log.Println("Error marshalling response body", err)
		return events_aws.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events_aws.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}
