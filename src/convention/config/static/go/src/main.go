package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
)

type Event struct {
	Msg string `json:"msg"`
}

type Response struct {
	Echo string `json:"echo"`
}

func HandleLambdaEvent(event *Event) (*Response, error) {
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}

	return &Response{
		Echo: event.Msg,
	}, nil
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
