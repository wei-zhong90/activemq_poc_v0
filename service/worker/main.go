package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/aws/aws-sdk-go/service/iotdataplane"
	"github.com/wei-zhong90/lambdautils"
)

type MQMessage struct {
	OwnerId  string
	DeviceId string
	Location []string
	Duration float32
}

// JobHandler implements http.Handler for the /configure endpoint.
func handler(ctx context.Context, event events.ActiveMQEvent) error {

	sess := session.Must(session.NewSession())
	svc := dynamodb.New(sess)

	// we need to use an IoT control plane client to get an endpoint address
	ctrlSvc := iot.New(sess)
	descResp, err := ctrlSvc.DescribeEndpoint(&iot.DescribeEndpointInput{})
	if err != nil {
		log.Fatal("failed to get dataplane endpoint", err)
	}

	// fmt.Printf("ADDR: %s", *descResp.EndpointAddress)

	iot := iotdataplane.New(sess, &aws.Config{
		Endpoint: descResp.EndpointAddress,
	})

	messages := event.Messages

	// check data in dynamodb
	table := lambdautils.DDBtable()

	for _, v := range messages {

		data := v.Data
		rawDecodedText, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Fatalln(err)
		}

		var value MQMessage
		if err := json.Unmarshal([]byte(rawDecodedText), &value); err != nil {
			log.Fatalln(err)
		}

		attributeValue := map[string]*dynamodb.AttributeValue{":owner": {S: &value.OwnerId}, ":device": {S: &value.DeviceId}}
		expression := fmt.Sprint("UserId = :owner AND DeviceId = :device")
		capacity := "TOTAL"

		params := &dynamodb.QueryInput{
			ReturnConsumedCapacity:    &capacity,
			ExpressionAttributeValues: attributeValue,
			KeyConditionExpression:    &expression,
			TableName:                 aws.String(table),
		}
		output, err := svc.Query(params)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Println(*output)

		topic := fmt.Sprintf("/%s/%s", value.OwnerId, value.DeviceId)
		fmt.Printf("TOPIC: %s\n", topic)
		retain := true
		var qos int64 = 1
		for _, v := range output.Items {
			result, err := json.Marshal(v)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(result)

			params := iotdataplane.PublishInput{
				Payload: result,
				Topic:   &topic,
				Retain:  &retain,
				Qos:     &qos,
			}
			publishResult, err := iot.Publish(&params)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("*********************")
			fmt.Println(*publishResult)
			fmt.Println("*********************")
			// req, resp := iot.PublishRequest(&params)

			// if err := req.Send(); err == nil { // resp is now filled
			// 	fmt.Println(resp)
			// } else {
			// 	log.Fatal(err)
			// }
		}
	}
	return nil
}

func main() {
	lambdautils.Mustenv(lambdautils.EnvDDBtable)
	lambda.Start(handler)
}
