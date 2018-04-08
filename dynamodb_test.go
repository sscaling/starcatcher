package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func TestDynamoDb(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("eu-west-1"),
		Credentials: credentials.NewSharedCredentials("", "starcatcher"),
	})

	// Log every request made and its payload
	sess.Handlers.Send.PushFront(func(r *request.Request) {
		fmt.Printf("Request: %s/%s, Payload: %s\n",
			r.ClientInfo.ServiceName, r.Operation.Name, r.Params)
	})

	_, err = sess.Config.Credentials.Get()
	if err != nil {
		t.FailNow()
	}

	// Create DynamoDB client
	// and expose HTTP requests/responses
	// svc := dynamodb.New(sess, aws.NewConfig().WithLogLevel(aws.LogDebugWithHTTPBody))
	svc := dynamodb.New(sess)

	// Call ListTables just to see HTTP request/response
	// The request should have the CustomHeader set to 10
	tab, err := svc.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(tab)

	// retrieve an item from the DB
	getitem, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("starcatcher"),
		Key: map[string]*dynamodb.AttributeValue{
			"date": {
				S: aws.String("201804071400"),
			},
			"id": {
				S: aws.String("3A902A76-FD67-46E6-B6BF-055E49A4CFC7"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// getitem
	var data map[string]interface{}
	dynamodbattribute.UnmarshalMap(getitem.Item, data)

	fmt.Printf("%#v\n", data)
}
