package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var db *dynamodb.Client

type Response events.APIGatewayProxyResponse

// Define struct to match DynamoDB schema
type UrlItem struct {
    Code string `dynamodbav:"Code"`
    URL string `dynamodbav:"URL"`  
}


func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v", err)
	}

	db = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	log.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

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
	log.Printf("ShortenURL function called")

	var data map[string]string
    json.Unmarshal([]byte(requests.Body), &data)

    // Extract URL 
    rawURL := data["url"] 

    parsed, err := url.Parse(rawURL)
    if err != nil {
        return invalidRequest()
    }

	code := generateShortUrl()

	item := struct {
		Code  string `json:"code"`
		URL string `json:"url"`
	}{
		Code: code,
		URL: parsed.String(),
	}

	attrValue, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Printf("Error marshaling DynamoDB attribute value: %v", err)
		return internalError(err)
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("url-shortner"),
		Item:      attrValue,
	})

	if err != nil {
		log.Printf("Error putting item to DynamoDB: %v", err)
		return internalError(err)
	}

	log.Printf("URL shortened successfully. Code: %s", code)

	return Response{
		StatusCode: 200,
		Body:       code,
	}, nil
}

func redirectURL(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	log.Printf("RedirectURL function called")

	code := request.PathParameters["code"]

	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("url-shortner"),
		Key: map[string]types.AttributeValue{
			"Code": &types.AttributeValueMemberS{Value: code},
		},
	})

	if err != nil {
		log.Printf("Error getting item from DynamoDB: %v", err)
		return internalError(err)
	}

	if result.Item == nil {
		log.Printf("URL not found for code: %s", code)
		return NotFound()
	}

	var item UrlItem
    err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		log.Printf("Error unmarshaling DynamoDB attribute value: %v", err)
		return internalError(err)
	}

	log.Printf("Redirecting to URL: %s", item.URL)

	return Response{
		StatusCode: 301,
		Headers: map[string]string{
			"Location": item.URL,
		},
	}, nil
}

func generateShortUrl() string {
	log.Printf("Generating short URL")

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	rand.NewSource(time.Now().UnixNano())
	shortkey := make([]byte, keyLength)
	for i := range shortkey {
		shortkey[i] = charset[rand.Intn(len(charset))]
	}
	code := string(shortkey)

	log.Printf("Generated short URL: %s", code)

	return code
}

// Helper functions

func internalError(err error) (Response, error) {
	log.Printf("Internal error: %v", err)
	return Response{
		StatusCode: 500,
		Body:       err.Error(),
	}, nil
}

func NotFound() (Response, error) {
	log.Printf("Resource not found")
	return Response{
		StatusCode: 404,
		Body:       "Not found",
	}, nil
}

func unhandledMethod() (Response, error) {
	log.Printf("Unhandled method")
	return Response{
		StatusCode: 405,
		Body:       "Method not allowed",
	}, nil
}

func invalidRequest() (Response, error) {
	log.Printf("Invalid request")
	return Response{
		StatusCode: 400,
		Body:       "Invalid request",
	}, nil
}

func main() {
	lambda.Start(Handler)
}
