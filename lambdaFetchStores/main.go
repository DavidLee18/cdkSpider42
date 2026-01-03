package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	evtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type Store struct {
	LicenseNum   string    `json:"LCNS_NO" excel:"인허가번호"`
	Name         string    `json:"BSSH_NM" excel:"업소명"`
	Address      string    `json:"ADDR" excel:"주소"`
	PhoneNumber  string    `json:"TELNO" excel:"전화번호"`
	Category     string    `json:"INDUTY_CD_NM" excel:"업종"`
	OwnerName    string    `json:"PRSDNT_NM" excel:"대표자명"`
	ApprovalDate string    `json:"PRMS_DT" excel:"허가일자"`
	ID           uuid.UUID `json:"-" excel:"-"`
}

type StoreInfoResult struct {
	Message string `json:"MSG"`
	Code    string `json:"CODE"`
}

type StoreInfos struct {
	TotalCount string          `json:"total_count"`
	Row        []Store         `json:"row"`
	Result     StoreInfoResult `json:"RESULT"`
}

type Limit struct {
	TableName string    `json:"TABLE_NAME"`
	ID        uuid.UUID `json:"ID"`
	FieldName string    `json:"FIELD_NAME"`
	Value     string    `json:"VALUE"`
	LastFrom  int64     `json:"LAST_FROM"`
	LastUntil int64     `json:"LAST_UNTIL"`
}

type PayloadType uint8

const (
	PayloadStart PayloadType = iota
	PayloadBetween
	PayloadEnd
)

type Payload struct {
	Type  PayloadType `json:"TYPE"`
	From  int64       `json:"FROM"`
	Until int64       `json:"UNTIL"`
}

var (
	dbClient    *dynamodb.Client
	eventBridge *eventbridge.Client
	s3Client    *s3.Client
	secrets     *secretsmanager.Client
	sqsClient   *sqs.Client
	snsClient   *sns.Client
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	dbClient = dynamodb.NewFromConfig(cfg)
	eventBridge = eventbridge.NewFromConfig(cfg)
	s3Client = s3.NewFromConfig(cfg)
	secrets = secretsmanager.NewFromConfig(cfg)
	sqsClient = sqs.NewFromConfig(cfg)
	snsClient = sns.NewFromConfig(cfg)
}

func handleRequest(ctx context.Context, event json.RawMessage) {
	log.Printf("Received event: %v", event)
	queueURL := os.Getenv("QUEUE_URL")
	if queueURL == "" {
		log.Fatalf("QUEUE_URL environment variable not set")
	}
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatalf("TABLE_NAME environment variable not set")
	}
	limitTable := os.Getenv("LIMIT_TABLE_NAME")
	if limitTable == "" {
		log.Fatalf("LIMIT_TABLE_NAME environment variable not set")
	}
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err != nil {
		var evEvent events.EventBridgeEvent
		if err := json.Unmarshal(event, &evEvent); err != nil {
			log.Fatalf("failed to unmarshal EventBridge event: %v", err)
		}
		var payload Payload
		if err := json.Unmarshal(evEvent.Detail, &payload); err != nil {
			recs, err := dbClient.Query(ctx, &dynamodb.QueryInput{
				TableName:              aws.String(limitTable),
				KeyConditionExpression: aws.String("TABLE_NAME = :table_name"),
				ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
					":table_name": &dbtypes.AttributeValueMemberS{Value: tableName},
				},
				ScanIndexForward: aws.Bool(false),
				ConsistentRead:   aws.Bool(true),
				Limit:            aws.Int32(1),
			})
			if err != nil {
				log.Fatalf("failed to query limit table: %v", err)
			}
			if len(recs.Items) == 0 {
				log.Fatalf("no limit found for table %s", tableName)
			}
			var l Limit
			if err := attributevalue.UnmarshalMap(recs.Items[0], &l); err != nil {
				log.Fatalf("failed to unmarshal limit: %v", err)
			}
			if l.LastFrom < 0 || l.LastUntil < 0 {
				log.Fatalf("invalid limit found for table %s", tableName)
			}
			sendQueue(ctx, Payload{
				Type:  PayloadBetween,
				From:  l.LastFrom,
				Until: l.LastUntil,
			})
		}
	}

	xlBucket := os.Getenv("BUCKET_NAME")
	if xlBucket == "" {
		log.Fatalf("BUCKET_NAME environment variable not set")
	}
	apiAction := os.Getenv("API_ACTION")
	if apiAction == "" {
		log.Fatalf("API_ACTION environment variable not set")
	}
	snsArn := os.Getenv("SNS_ARN")
	if snsArn == "" {
		log.Fatalf("SNS_ARN environment variable not set")
	}
	distUrl := os.Getenv("DIST_URL")
	if distUrl == "" {
		log.Fatalf("DIST_URL environment variable not set")
	}
	apiKey, err := secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("Spider42Secret"),
	})
	if err != nil {
		log.Fatalf("failed to get API_KEY: %v", err)
	}
	httpClient := &http.Client{
		Timeout: time.Second * 40,
	}
	for _, rec := range sqsEvent.Records {
		var payload Payload
		if err := json.Unmarshal([]byte(rec.Body), &payload); err != nil {
			log.Fatalf("failed to unmarshal SQS payload: %v", err)
		}
		switch payload.Type {
		case PayloadStart:
			resp, err := httpClient.Get(fmt.Sprintf("http://openapi.foodsafetykorea.go.kr/api/%s/%s/json/%d/%d", *apiKey.SecretString, apiAction, payload.From, payload.Until))
			if err != nil {
				if strings.Contains(err.Error(), "Timeout") {
					log.Printf("Request timed out; retrying...")
					sendQueue(ctx, payload)
					continue
				}
				log.Printf("failed to get API response: %v", err)
				continue
			}
			defer resp.Body.Close()
			resRaw := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(resRaw)
			if err != nil {
				log.Fatalf("failed to read API response: %v", err)
			}
			res := string(resRaw)
			log.Printf("raw response: %v", res)
			if strings.Contains(res, "alert(") || strings.Contains(res, "<script") {
				log.Println("\"alert()\" or \"<script>\" detected; retrying...")
				sendQueue(ctx, payload)
				continue
			}
			var infos StoreInfos
			if err := json.Unmarshal(resRaw, &infos); err != nil {
				log.Fatalf("failed to unmarshal API response: %v", err)
			}
			switch infos.Result.Code {
			case "INFO-000":
				count, err := strconv.Atoi(infos.TotalCount)
				if err != nil {
					log.Fatalf("failed to convert total count to int: %v", err)
				}
				if len(infos.Row) != count {
					log.Printf("expected %d rows, got %d", count, len(infos.Row))
				}
			case "INFO-300":
				log.Printf("API call limit reached; will retry tomorrow")
				retryTomorrow(ctx, payload)
				sendQueue(ctx, payload)
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			case "INFO-200":
			case "ERROR-332":
				width := payload.Until - payload.From
				if width > 1 {
					sendQueue(ctx, Payload{
						Type:  PayloadBetween,
						From:  payload.From,
						Until: payload.From + width/2,
					})
				} else {
					log.Printf("cannot fetch more; terminating...")
					retryTomorrow(ctx, Payload{
						Type:  PayloadStart,
						From:  payload.From,
						Until: payload.From + 1000,
					})
					sendQueue(ctx, Payload{
						Type: PayloadEnd,
					})
				}
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			case "ERROR-336":
				log.Printf("cannot fetch %d items; retrying...", payload.Until-payload.From)
				sendQueue(ctx, Payload{
					Type:  PayloadBetween,
					From:  payload.From,
					Until: payload.From + 1000,
				})
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			default:
				log.Fatalf("unexpected result code: %s (%s)", infos.Result.Code, infos.Result.Message)
			}
			if len(infos.Row) == 0 {
				log.Fatalf("fetched store infos are empty")
			}
			newUuid, err := uuid.NewV7()
			if err != nil {
				log.Fatalf("failed to generate uuid: %v", err)
			}
			firstLimit, err := attributevalue.MarshalMap(Limit{
				TableName: tableName,
				ID:        newUuid,
				FieldName: "PRMS_DT",
				Value:     infos.Row[0].ApprovalDate,
				LastFrom:  payload.From,
				LastUntil: payload.Until,
			})
			if err != nil {
				log.Fatalf("failed to marshal limit: %v", err)
			}
			_, err = dbClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
				TransactItems: []dbtypes.TransactWriteItem{
					{
						Put: &dbtypes.Put{
							TableName:           aws.String(limitTable),
							Item:                firstLimit,
							ConditionExpression: aws.String("attribute_not_exists(#id)"),
							ExpressionAttributeNames: map[string]string{
								"#id": "ID",
							},
						},
					},
				},
			})
			if err != nil {
				log.Fatalf("failed to transact write items: %v", err)
			}
			reqs := []dbtypes.TransactWriteItem{}
			for _, item := range infos.Row {
				if item.ApprovalDate == "" {
					log.Printf("item of license number %s has no approval date", item.LicenseNum)
					continue
				}
				item.ID, err = uuid.NewV7()
				if err != nil {
					log.Fatalf("failed to generate UUID: %v", err)
				}
				av, err := attributevalue.MarshalMap(item)
				if err != nil {
					log.Fatalf("failed to marshal item: %v", err)
				}
				reqs = append(reqs, dbtypes.TransactWriteItem{
					Put: &dbtypes.Put{
						TableName:           aws.String(tableName),
						Item:                av,
						ConditionExpression: aws.String("attribute_not_exists(ID) AND (attribute_not_exists(BSSH_NM) AND attribute_not_exists(LCNS_NO))"),
					},
				})
				if len(reqs) == 100 {
					_, err = dbClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
						TransactItems: reqs,
					})
					if err != nil {
						log.Fatalf("failed to batch write items: %v", err)
					}
					reqs = []dbtypes.TransactWriteItem{}
				}
			}
			if len(reqs) > 0 {
				_, err = dbClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
					TransactItems: reqs,
				})
				if err != nil {
					log.Fatalf("failed to batch write items: %v", err)
				}
			}
			width := payload.Until - payload.From
			sendQueue(ctx, Payload{
				Type:  PayloadBetween,
				From:  payload.From + width,
				Until: payload.Until + width,
			})
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: aws.String(rec.ReceiptHandle),
			})
			if err != nil {
				log.Fatalf("failed to delete message: %v", err)
			}
		case PayloadBetween:
			resp, err := httpClient.Get(fmt.Sprintf("http://openapi.foodsafetykorea.go.kr/api/%s/%s/json/%d/%d", *apiKey.SecretString, apiAction, payload.From, payload.Until))
			if err != nil {
				if strings.Contains(err.Error(), "Timeout") {
					log.Printf("Request timed out; retrying...")
					sendQueue(ctx, payload)
					continue
				}
				log.Printf("failed to get API response: %v", err)
				continue
			}
			defer resp.Body.Close()
			resRaw := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(resRaw)
			if err != nil {
				log.Fatalf("failed to read API response: %v", err)
			}
			res := string(resRaw)
			log.Printf("raw response: %v", res)
			if strings.Contains(res, "alert(") || strings.Contains(res, "<script") {
				log.Println("\"alert()\" or \"<script>\" detected; retrying...")
				sendQueue(ctx, payload)
				continue
			}
			var infos StoreInfos
			if err := json.Unmarshal(resRaw, &infos); err != nil {
				log.Fatalf("failed to unmarshal API response: %v", err)
			}
			switch infos.Result.Code {
			case "INFO-000":
				count, err := strconv.Atoi(infos.TotalCount)
				if err != nil {
					log.Fatalf("failed to convert total count to int: %v", err)
				}
				if len(infos.Row) != count {
					log.Printf("expected %d rows, got %d", count, len(infos.Row))
				}
			case "INFO-300":
				log.Printf("API call limit reached; will retry tomorrow")
				retryTomorrow(ctx, payload)
				sendQueue(ctx, payload)
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			case "INFO-200":
			case "ERROR-332":
				width := payload.Until - payload.From
				if width > 1 {
					sendQueue(ctx, Payload{
						Type:  PayloadBetween,
						From:  payload.From,
						Until: payload.From + width/2,
					})
				} else {
					log.Printf("cannot fetch more; terminating...")
					retryTomorrow(ctx, Payload{
						Type:  PayloadStart,
						From:  payload.From,
						Until: payload.From + 1000,
					})
					sendQueue(ctx, Payload{
						Type: PayloadEnd,
					})
				}
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			case "ERROR-336":
				log.Printf("cannot fetch %d items; retrying...", payload.Until-payload.From)
				sendQueue(ctx, Payload{
					Type:  PayloadBetween,
					From:  payload.From,
					Until: payload.From + 1000,
				})
				sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: &rec.ReceiptHandle,
				})
				continue
			default:
				log.Fatalf("unexpected result code: %s (%s)", infos.Result.Code, infos.Result.Message)
			}
			reqs := []dbtypes.TransactWriteItem{}
			for _, item := range infos.Row {
				if item.ApprovalDate == "" {
					log.Printf("item of license number %s has no approval date", item.LicenseNum)
					continue
				}
				item.ID, err = uuid.NewV7()
				if err != nil {
					log.Fatalf("failed to generate UUID: %v", err)
				}
				av, err := attributevalue.MarshalMap(item)
				if err != nil {
					log.Fatalf("failed to marshal item: %v", err)
				}
				reqs = append(reqs, dbtypes.TransactWriteItem{
					Put: &dbtypes.Put{
						TableName:           aws.String(tableName),
						Item:                av,
						ConditionExpression: aws.String("attribute_not_exists(ID) AND (attribute_not_exists(BSSH_NM) AND attribute_not_exists(LCNS_NO))"),
					},
				})
				if len(reqs) == 100 {
					_, err = dbClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
						TransactItems: reqs,
					})
					if err != nil {
						log.Fatalf("failed to batch write items: %v", err)
					}
					reqs = []dbtypes.TransactWriteItem{}
				}
			}
			if len(reqs) > 0 {
				_, err = dbClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
					TransactItems: reqs,
				})
				if err != nil {
					log.Fatalf("failed to batch write items: %v", err)
				}
			}
			width := payload.Until - payload.From
			sendQueue(ctx, Payload{
				Type:  PayloadBetween,
				From:  payload.From + width,
				Until: payload.Until + width,
			})
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: aws.String(rec.ReceiptHandle),
			})
			if err != nil {
				log.Fatalf("failed to delete message: %v", err)
			}
		case PayloadEnd:
			res, err := dbClient.Query(ctx, &dynamodb.QueryInput{
				TableName:              aws.String(limitTable),
				ConsistentRead:         aws.Bool(true),
				KeyConditionExpression: aws.String("TABLE_NAME = :table_name"),
				ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
					":table_name": &dbtypes.AttributeValueMemberS{
						Value: tableName,
					},
				},
				Limit:            aws.Int32(1),
				ScanIndexForward: aws.Bool(false),
			})
			if err != nil {
				log.Fatalf("failed to query table: %v", err)
			}
			if len(res.Items) == 0 {
				log.Fatalf("no items found in table")
			}
			var limit Limit
			if err := attributevalue.UnmarshalMap(res.Items[0], &limit); err != nil {
				log.Fatalf("failed to unmarshal item: %v", err)
			}
			if limit.LastFrom == 0 && limit.LastUntil == 0 {
				log.Fatalf("no limits found in table")
			}
			var infos []*Store
			recs, err := dbClient.Query(ctx, &dynamodb.QueryInput{
				TableName:              aws.String(limit.TableName),
				ConsistentRead:         aws.Bool(true),
				KeyConditionExpression: aws.String(fmt.Sprintf("%s = :val and ID >= :id", limit.FieldName)),
				ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
					":val": &dbtypes.AttributeValueMemberS{
						Value: limit.Value,
					},
					":id": &dbtypes.AttributeValueMemberS{
						Value: limit.ID.String(),
					},
				},
			})
			if err != nil {
				log.Fatalf("failed to query table: %v", err)
			}
			if err = attributevalue.UnmarshalListOfMaps(recs.Items, &infos); err != nil {
				log.Fatalf("failed to unmarshal items: %v", err)
			}
			for recs.LastEvaluatedKey != nil {
				var tempInfos []*Store
				recs, err = dbClient.Query(ctx, &dynamodb.QueryInput{
					TableName:              aws.String(limit.TableName),
					ConsistentRead:         aws.Bool(true),
					KeyConditionExpression: aws.String(fmt.Sprintf("%s = :val and ID >= :id", limit.FieldName)),
					ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
						":val": &dbtypes.AttributeValueMemberS{
							Value: limit.Value,
						},
						":id": &dbtypes.AttributeValueMemberS{
							Value: limit.ID.String(),
						},
					},
					ExclusiveStartKey: recs.LastEvaluatedKey,
				})
				if err != nil {
					log.Fatalf("failed to query table: %v", err)
				}
				if err = attributevalue.UnmarshalListOfMaps(recs.Items, &tempInfos); err != nil {
					log.Fatalf("failed to unmarshal items: %v", err)
				}
				infos = append(infos, tempInfos...)
			}
			xlfile := excelize.NewFile()
			defer func() { _ = xlfile.Close() }()
			_, err = xlfile.NewSheet("Stores")
			if err != nil {
				log.Fatalf("failed to create sheet: %v", err)
			}
			xlfile.SetSheetRow("Stores", "A1", &[]any{"인허가번호", "업소명", "주소", "전화번호", "업종", "대표자명", "허가일자"})
			for i, info := range infos {
				err = xlfile.SetSheetRow("Stores", fmt.Sprintf("A%d", i+2), &[]any{info.LicenseNum, info.Name, info.Address, info.PhoneNumber, info.Category, info.OwnerName, info.ApprovalDate})
				if err != nil {
					log.Fatalf("failed to set sheet row: %v", err)
				}
			}
			if err = xlfile.SaveAs("/tmp/stores.xlsx"); err != nil {
				log.Fatalf("failed to save excel file: %v", err)
			}
			f, err := os.Open("/tmp/stores.xlsx")
			if err != nil {
				log.Fatalf("failed to open excel file: %v", err)
			}
			defer f.Close()
			_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(os.Getenv("BUCKET_NAME")),
				Key:    aws.String("stores.xlsx"),
				Body:   f,
			})
			if err != nil {
				log.Fatalf("failed to upload excel file to S3: %v", err)
			}
			log.Printf("Uploaded excel file to S3")
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: aws.String(rec.ReceiptHandle),
			})
			if err != nil {
				log.Fatalf("failed to delete message from SQS: %v", err)
			}
			_, err = snsClient.Publish(ctx, &sns.PublishInput{
				Subject:  aws.String("등록사항 조회 완료"),
				Message:  aws.String(fmt.Sprintf("인허가정보 조회가 완료되었습니다. %s/stores.xlsx 에서 확인하세요!", distUrl)),
				TopicArn: aws.String(snsArn),
			})
			if err != nil {
				log.Fatalf("failed to publish message to SNS: %v", err)
			}
		default:
			log.Fatalf("unexpected main.PayloadType: %#v", payload.Type)
		}
	}
}

func main() {
	lambda.Start(handleRequest)
}

func sendQueue(ctx context.Context, payload Payload) {
	queueURL := os.Getenv("QUEUE_URL")
	if queueURL == "" {
		log.Fatalf("QUEUE_URL environment variable not set")
	}

	json, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := sqsClient.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		Entries: []sqstypes.SendMessageBatchRequestEntry{
			{
				Id:          aws.String(uuid.NewString()),
				MessageBody: aws.String(string(json)),
			},
		},
		QueueUrl: aws.String(queueURL),
	})
	if err != nil || result.Failed != nil {
		log.Fatalf("failed to send message to queue: %v", err)
	}

	log.Printf("Message sent to queue: %v", result.Successful)
}

func retryTomorrow(ctx context.Context, payload Payload) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatalf("TABLE_NAME environment variable not set")
	}
	lambdaARN := os.Getenv("LAMBDA_ARN")
	if lambdaARN == "" {
		log.Fatalf("LAMBDA_ARN environment variable not set")
	}
	rules, err := eventBridge.ListRules(ctx, &eventbridge.ListRulesInput{
		EventBusName: aws.String("Spider42EventBus"),
		NamePrefix:   aws.String(tableName),
	})
	if err != nil {
		log.Fatalf("failed to list event rules: %v", err)
	}
	for rules.NextToken != nil {
		rules, err = eventBridge.ListRules(ctx, &eventbridge.ListRulesInput{
			EventBusName: aws.String("Spider42EventBus"),
			NamePrefix:   aws.String(tableName),
			NextToken:    rules.NextToken,
		})
		if err != nil {
			log.Fatalf("failed to list event rules: %v", err)
		}
	}
	lastRule := rules.Rules[len(rules.Rules)-1]
	var thenMin, thenHour, thenDay, thenYear int
	var thenMonth time.Month
	if _, err := fmt.Sscanf(*lastRule.ScheduleExpression, "cron(%d %d %d %d ? %d)", &thenMin, &thenHour, &thenDay, &thenMonth, &thenYear); err != nil {
		log.Fatalf("failed to parse last rule schedule expression: %v", err)
	}
	destTime := time.Now().UTC()
	if (destTime.Hour() > 11 && destTime.Minute() > 30) || destTime.Hour() > 12 {
		destTime = time.Date(destTime.Year(), destTime.Month(), destTime.Day()+1, 10, 30, 0, 0, time.UTC)
	} else if destTime.Day() > thenDay || destTime.Month() > thenMonth || destTime.Year() > thenYear {
		destTime = time.Date(destTime.Year(), destTime.Month(), destTime.Day(), 10, 30, 0, 0, time.UTC)
	}
	ruleName := fmt.Sprintf("%s_%d", tableName, destTime.Unix())

	json, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("failed to marshal payload: %v", err)
	}

	_, err = eventBridge.PutRule(ctx, &eventbridge.PutRuleInput{
		EventBusName:       aws.String("Spider42EventBus"),
		Name:               aws.String(ruleName),
		ScheduleExpression: aws.String(fmt.Sprintf("cron(%02d %02d %02d %02d ? %d)", destTime.Minute(), destTime.Hour(), destTime.Day(), destTime.Month(), destTime.Year())),
		State:              evtypes.RuleStateEnabled,
	})
	if err != nil {
		log.Fatalf("failed to create event rule: %v", err)
	}

	_, err = eventBridge.PutTargets(ctx, &eventbridge.PutTargetsInput{
		EventBusName: aws.String("Spider42EventBus"),
		Rule:         aws.String(ruleName),
		Targets: []evtypes.Target{
			{
				Id:    aws.String(uuid.NewString()),
				Arn:   aws.String(lambdaARN),
				Input: aws.String(string(json)),
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create event target: %v", err)
	}

	_, err = eventBridge.DisableRule(ctx, &eventbridge.DisableRuleInput{
		EventBusName: aws.String("Spider42EventBus"),
		Name:         aws.String("Spider42EventBusRule"),
	})
	if err != nil {
		log.Fatalf("failed to disable event rule: %v", err)
	}
}
