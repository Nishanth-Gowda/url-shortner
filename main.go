package main

import (
	"context"
	"math/rand"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var db *dynamodb.Client

type Response events.APIGatewayProxyResponse

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	db = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	switch request.HTTPMethod {
	case "POST":
		return shortenURL(ctx, request)
	case "GET":
		return redirectURL(ctx, request)
	default:
		return unhandledMethod()
	}
}

func shortenURL(ctx context.Context, requests events.APIGatewayProxyRequest) (Response, error) {
	u, err := url.Parse(requests.Body)
	if err != nil {
		return invalidRequest()
	}

	code := generateShortUrl()

	item := struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}{
		Code: code,
		URL:  u.String(),
	}

	attrValue, err := attributevalue.MarshalMap(item)
	if err != nil {
		return internalError(err)
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("url-shortener"),
		Item:      attrValue,
	})

	if err != nil {
		return internalError(err)
	}

	return Response{
		StatusCode: 200,
		Body:       code,
	}, nil
}

func redirectURL(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	code := request.PathParameters["code"]

	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("urlshortener"),
		Key: map[string]types.AttributeValue{
			"code": &types.AttributeValueMemberS{Value: code},
		},
	})

	if err != nil {
		return internalError(err)
	}

	if result.Item == nil {
		return NotFound()
	}

	var url string
	err = attributevalue.UnmarshalMap(result.Item, &url)
	if err != nil {
		return internalError(err)
	}

	return Response{
		StatusCode: 301,
		Headers: map[string]string{
			"Location": url,
		},
	}, nil
}

func generateShortUrl() string {
	// Generate a random short URL
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	rand.NewSource(time.Now().UnixNano())
	shortkey := make([]byte, keyLength)
	for i := range shortkey {
		shortkey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortkey)
}

// Helper functions

func internalError(err error) (Response, error) {
	return Response{
		StatusCode: 500,
		Body:       err.Error(),
	}, nil
}

func NotFound() (Response, error) {
	return Response{
		StatusCode: 404,
		Body:       "Not found",
	}, nil
}

func unhandledMethod() (Response, error) {
	return Response{
		StatusCode: 405,
		Body:       "Method not allowed",
	}, nil
}

func invalidRequest() (Response, error) {
	return Response{
		StatusCode: 400,
		Body:       "Invalid request",
	}, nil
}

func main() {
	lambda.Start(Handler)
}
