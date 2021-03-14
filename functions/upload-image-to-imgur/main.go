package main

import (
	"context"
	"encoding/json"
	"log"
	"reddit-image-recognition-fetcher/shared/environment"
	"reddit-image-recognition-fetcher/shared/models"
	"reddit-image-recognition-fetcher/shared/s3"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// SqsPayload is the struct for the sqs message data
type SqsPayload struct {
	Key        string `json:"key"`
	Bucket     string `json:"bucket"`
	Prediction string `json:"prediction"`
}

var (
	clientID  = environment.GetString("IMGUR_CLIENT_ID", "")
	albumHash = environment.GetString("ALBUM_HASH", "")
)

func apiGatewayHandler(ctx context.Context, sqsEvent *events.SQSEvent) (string, error) {
	bucketData, err := newSqsPayload([]byte(sqsEvent.Records[0].Body))
	if err != nil {
		log.Println(err)
		return "", nil
	}

	image, err := s3.GetObject(ctx, bucketData.Bucket, bucketData.Key)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	imgur, err := models.NewImgur(clientID)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	link, err := imgur.Upload(image, albumHash)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	return link, nil
}

func newSqsPayload(body []byte) (*SqsPayload, error) {
	payload := &SqsPayload{}
	err := json.Unmarshal(body, payload)

	if err != nil {
		return nil, err
	}

	return payload, nil
}

func main() {
	lambda.Start(apiGatewayHandler)
}
