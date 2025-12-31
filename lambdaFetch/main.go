package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Item struct {
	Message string
}

type MyEvent struct {
	ID      string `json:"ID"`
	Message string `json:"message"`
}

var (
	dbClient *dynamodb.Client
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func handleRequest(ctx context.Context, event MyEvent) {
	av, err := attributevalue.MarshalMap(event)
	if err != nil {
		log.Fatalf("Got error marshalling new movie item: %s", err)
	}

	tableName := "MyDynamoDB"

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      av,
		TableName: &tableName,
	})
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
	}
	log.Printf("Successfully added '%s' to table %s\n", event.Message, tableName)
}

func main() {
	lambda.Start(handleRequest)
}
