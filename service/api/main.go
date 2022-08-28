package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/apex/gateway"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/wei-zhong90/lambdautils"
)

// ContentType contains the Content-Type header sent on all responses.
const ContentType = "application/json; charset=utf8"

var (
	username     = os.Getenv("USER_NAME")
	password     = os.Getenv("PASSWORD")
	hostEndpoint = os.Getenv("MQ_ENDPOINT")
	queue        = os.Getenv("QUEUE")
)

// time type
type UnixTime time.Time

func (u *UnixTime) UnmarshalJSON(b []byte) error {
	var timestamp int64
	err := json.Unmarshal(b, &timestamp)
	if err != nil {
		return err
	}
	*u = UnixTime(time.Unix(timestamp, 0))
	return nil
}

func (u UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", (time.Time(u).Unix()))), nil
}

// Testing event
type ConfigurationEvent struct {
	Timestamp     UnixTime `json:"timestamp"`
	UserId        string   `json:"userid"`
	Configuration []Config `json:"configuration"`
}

// Configuration struct
type Config struct {
	DeviceId string   `json:"deviceid"`
	Cron     string   `json:"cronexpression"`
	Location []string `json:"location"`
	Duration float32  `json:"duration"`
}

// RootHandler is a http.HandlerFunc for the / endpoint.
// func RootHandler(w http.ResponseWriter, _ *http.Request) {
// 	json.NewEncoder(w).Encode(WelcomeMessageResponse)
// }

// JobHandler implements http.Handler for the /configure endpoint.
type JobHandler struct {
	svc *dynamodb.DynamoDB
}

// JobHandlerFunc returns a http.HandlerFunc for the /configure endpoint.
func JobHandlerFunc(svc *dynamodb.DynamoDB) http.HandlerFunc {
	jh := JobHandler{svc: svc}
	return jh.ServeHTTP
}

type stompMessage struct {
	OwnerId  string
	DeviceId string
	Location []string
	Duration float32
}

// JobHandler implements http.Handler for the /configure endpoint.
func (h JobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := ConfigurationEvent{}
	if err := json.Unmarshal(body, &s); err != nil {
		log.Fatalln(err)
	}

	netConn, err := tls.Dial("tcp", hostEndpoint, &tls.Config{})
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer netConn.Close()

	// Now create the stomp connection
	stompConn, err := stomp.Connect(netConn,
		stomp.ConnOpt.Login(username, password))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer stompConn.Disconnect()

	// Persist data to dynamodb
	table := lambdautils.DDBtable()

	for _, v := range s.Configuration {
		params := &dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"UserId": {
					S: aws.String(s.UserId),
				},
				"DeviceId": {
					S: aws.String(v.DeviceId),
				},
				"CronExpression": {
					S: aws.String(v.Cron),
				},
				"Location": {
					L: convertToAttr(v.Location),
				},
				"Duration": {
					N: aws.String(fmt.Sprintf("%f", v.Duration)),
				},
			},
			TableName: aws.String(table),
		}
		if _, err := h.svc.PutItem(params); err != nil {
			log.Fatal(err)
		}
		var message = &stompMessage{
			OwnerId:  s.UserId,
			DeviceId: v.DeviceId,
			Location: v.Location,
			Duration: v.Duration,
		}
		e, err := json.Marshal(message)
		if err != nil {
			log.Fatal(err)
		}
		if err := stompConn.Send(queue, "application/json", []byte(e), func(f *frame.Frame) error {
			// f.Header.Add("AMQ_SCHEDULED_REPEAT", "1000")
			f.Header.Add("AMQ_SCHEDULED_CRON", v.Cron)
			return nil
		}); err != nil {
			log.Fatal(err)
		}
	}

	log.Println(stompConn.Version().String())

	json.NewEncoder(w).Encode(s)
	w.WriteHeader(http.StatusCreated)
}

func convertToAttr(attrList []string) []*dynamodb.AttributeValue {
	var a []*dynamodb.AttributeValue
	for _, v := range attrList {
		a = append(a, &dynamodb.AttributeValue{S: aws.String(v)})
	}
	return a
}

// RegisterRoutes registers the API's routes.
func RegisterRoutes() {
	sess := session.Must(session.NewSession())
	svc := dynamodb.New(sess)

	// http.Handle("/", h(RootHandler))
	http.Handle("/configure", h(JobHandlerFunc(svc)))
}

// h wraps a http.HandlerFunc and adds common headers.
func h(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentType)
		next.ServeHTTP(w, r)
	})
}

func main() {
	lambdautils.Mustenv(lambdautils.EnvDDBtable)
	RegisterRoutes()
	log.Fatal(gateway.ListenAndServe(":3000", nil))
}
