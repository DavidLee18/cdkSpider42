package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsscheduler"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsschedulertargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

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

type CdkSpider42StackProps struct {
	awscdk.StackProps
}

func NewCdkSpider42Stack(scope constructs.Construct, id string, props *CdkSpider42StackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// The code that defines your stack goes here

	logPolicy := awsiam.ManagedPolicy_FromManagedPolicyArn(stack, aws.String("CloudWatchLogsFullAccess"), aws.String("arn:aws:iam::aws:policy/CloudWatchLogsFullAccess"))

	// create role
	lambdaRole := awsiam.NewRole(stack, aws.String("spider42LambdaRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(aws.String("lambda.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			logPolicy,
			awsiam.ManagedPolicy_FromManagedPolicyArn(stack, aws.String("AWSLambda_ReadOnlyAccess"), aws.String("arn:aws:iam::aws:policy/AWSLambda_ReadOnlyAccess")),
		},
	})

	// create SQS queue
	storeDeadQueue := awssqs.NewQueue(stack, jsii.String("spider42StoreDeadQueue"), &awssqs.QueueProps{
		EnforceSSL:             jsii.Bool(true),
		QueueName:              jsii.String("Spider42StoreDeadQueue"),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		RetentionPeriod:        awscdk.Duration_Hours(jsii.Number(12)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(90)),
	})
	storeQueue := awssqs.NewQueue(stack, jsii.String("spider42StoreQueue"), &awssqs.QueueProps{
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(5),
			Queue:           storeDeadQueue,
		},
		EnforceSSL:             jsii.Bool(true),
		QueueName:              jsii.String("Spider42StoreQueue"),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		RetentionPeriod:        awscdk.Duration_Hours(jsii.Number(12)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(90)),
	})
	updateDeadQueue := awssqs.NewQueue(stack, jsii.String("spider42UpdateDeadQueue"), &awssqs.QueueProps{
		EnforceSSL:             jsii.Bool(true),
		QueueName:              jsii.String("Spider42UpdateDeadQueue"),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		RetentionPeriod:        awscdk.Duration_Hours(jsii.Number(12)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(90)),
	})
	updateQueue := awssqs.NewQueue(stack, jsii.String("spider42UpdateQueue"), &awssqs.QueueProps{
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(5),
			Queue:           updateDeadQueue,
		},
		EnforceSSL:             jsii.Bool(true),
		QueueName:              jsii.String("Spider42UpdateQueue"),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		RetentionPeriod:        awscdk.Duration_Hours(jsii.Number(12)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(90)),
	})

	storeQueue.GrantConsumeMessages(lambdaRole)
	storeQueue.GrantSendMessages(lambdaRole)
	updateQueue.GrantConsumeMessages(lambdaRole)
	updateQueue.GrantSendMessages(lambdaRole)

	// create EventBridge scheduler
	when := time.Now().UTC()
	if when.Hour() > 10 || (when.Hour() == 10 && when.Minute() >= 30) {
		when = time.Date(when.Year(), when.Month(), when.Day()+1, 7, 55, 0, 0, time.UTC)
	} else {
		when = time.Date(when.Year(), when.Month(), when.Day(), 7, 55, 0, 0, time.UTC)
	}
	scheduleGroup := awsscheduler.NewScheduleGroup(stack, jsii.String("Spider42ScheduleGroup"), &awsscheduler.ScheduleGroupProps{
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		ScheduleGroupName: jsii.String("Spider42ScheduleGroup"),
	})
	storeSchedule := awsscheduler.NewSchedule(stack, jsii.String("Spider42StoreSchedule"), &awsscheduler.ScheduleProps{
		ScheduleGroup: scheduleGroup,
		Schedule: awsscheduler.ScheduleExpression_Cron(&awsscheduler.CronOptionsWithTimezone{
			Minute: jsii.String(fmt.Sprintf("%02d", when.Minute())),
			Hour:   jsii.String(fmt.Sprintf("%02d", when.Hour())),
			Day:    jsii.String(fmt.Sprintf("%02d", when.Day())),
			Month:  jsii.String(fmt.Sprintf("%02d", when.Month())),
			Year:   jsii.String(fmt.Sprintf("%04d", when.Year())),
		}),
		Target: awsschedulertargets.NewSqsSendMessage(storeQueue, &awsschedulertargets.SqsSendMessageProps{
			DeadLetterQueue: storeDeadQueue,
			Input: awsscheduler.ScheduleTargetInput_FromObject(Payload{
				Type:  PayloadStart,
				From:  0,
				Until: 999,
			}),
		})})
	updateSchedule := awsscheduler.NewSchedule(stack, jsii.String("Spider42UpdateSchedule"), &awsscheduler.ScheduleProps{
		ScheduleGroup: scheduleGroup,
		Schedule: awsscheduler.ScheduleExpression_Cron(&awsscheduler.CronOptionsWithTimezone{
			Minute: jsii.String(fmt.Sprintf("%02d", when.Minute())),
			Hour:   jsii.String(fmt.Sprintf("%02d", when.Hour())),
			Day:    jsii.String(fmt.Sprintf("%02d", when.Day())),
			Month:  jsii.String(fmt.Sprintf("%02d", when.Month())),
			Year:   jsii.String(fmt.Sprintf("%04d", when.Year())),
		}),
		Target: awsschedulertargets.NewSqsSendMessage(updateQueue, &awsschedulertargets.SqsSendMessageProps{
			DeadLetterQueue: updateDeadQueue,
			Input: awsscheduler.ScheduleTargetInput_FromObject(Payload{
				Type:  PayloadStart,
				From:  0,
				Until: 999,
			}),
		})})

	// create DynamoDB table
	storesTable := awsdynamodb.NewTable(stack, jsii.String("Spider42Stores"), &awsdynamodb.TableProps{
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		PartitionKey: &awsdynamodb.Attribute{
			Name: aws.String("PRMS_DT"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		SortKey: &awsdynamodb.Attribute{
			Name: aws.String("ID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Spider42Stores"),
	})
	storesUpdatesTable := awsdynamodb.NewTable(stack, jsii.String("Spider42Updates"), &awsdynamodb.TableProps{
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		PartitionKey: &awsdynamodb.Attribute{
			Name: aws.String("CHNG_DT"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		SortKey: &awsdynamodb.Attribute{
			Name: aws.String("ID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Spider42Updates"),
	})
	limitTable := awsdynamodb.NewTable(stack, jsii.String("Spider42Limits"), &awsdynamodb.TableProps{
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		PartitionKey: &awsdynamodb.Attribute{
			Name: aws.String("TABLE_NAME"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
		SortKey: &awsdynamodb.Attribute{
			Name: aws.String("ID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Spider42Limits"),
	})

	storesTable.GrantReadWriteData(lambdaRole)
	storesUpdatesTable.GrantReadWriteData(lambdaRole)
	limitTable.GrantReadWriteData(lambdaRole)

	spider42Bucket := awss3.NewBucket(stack, jsii.String("Spider42Bucket"), &awss3.BucketProps{
		AutoDeleteObjects: jsii.Bool(true),
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		EnforceSSL:        jsii.Bool(true),
		PublicReadAccess:  jsii.Bool(false),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		Versioned:         jsii.Bool(true),
	})

	snsLogRole := awsiam.NewRole(stack, aws.String("spider42SnsLogRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(aws.String("sns.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			logPolicy,
		},
	})

	spider42JobDone := awssns.NewTopic(stack, jsii.String("Spider42JobDone"), &awssns.TopicProps{
		DisplayName: jsii.String("Spider42JobDone"),
		LoggingConfigs: &[]*awssns.LoggingConfig{
			{
				Protocol:                  awssns.LoggingProtocol_APPLICATION,
				FailureFeedbackRole:       snsLogRole,
				SuccessFeedbackRole:       snsLogRole,
				SuccessFeedbackSampleRate: jsii.Number(100),
			},
		},
		TopicName: jsii.String("Spider42JobDone"),
	})

	spider42JobDone.GrantPublish(lambdaRole)
	spider42JobDone.GrantSubscribe(snsLogRole)

	awssns.NewSubscription(stack, jsii.String("Spider42JobDoneSubscription"), &awssns.SubscriptionProps{
		Topic:    spider42JobDone,
		Endpoint: jsii.String(os.Getenv("SPIDER42_JOB_DONE_EMAIL")),
		Protocol: awssns.SubscriptionProtocol_EMAIL,
	})

	// Creates Origin Access Identity (OAI) to only allow CloudFront to get content
	cloudfrontDefaultBehavior := &awscloudfront.BehaviorOptions{
		// Sets the S3 Bucket as the origin and tells CloudFront to use the created OAI to access it
		Origin: awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(spider42Bucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{
			OriginAccessControl: awscloudfront.NewS3OriginAccessControl(stack, jsii.String("Spider42OAC"), &awscloudfront.S3OriginAccessControlProps{
				Signing: awscloudfront.Signing_SIGV4_NO_OVERRIDE(),
			}),
		}),
		ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
	}
	dist := awscloudfront.NewDistribution(stack, jsii.String("Spider42CloudFront"), &awscloudfront.DistributionProps{
		DefaultBehavior: cloudfrontDefaultBehavior,
	})

	spider42Bucket.GrantPut(lambdaRole, "*")

	// create lambda functions
	lambdaFetchStores := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("spider42FetchStores"), &awscdklambdagoalpha.GoFunctionProps{
		Architecture: awslambda.Architecture_ARM_64(),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOARCH":      aws.String("arm64"),
				"CGO_ENABLED": aws.String("0"),
			},
			GoBuildFlags: jsii.Strings(`-ldflags "-s -w"`),
		},
		Entry: jsii.String("./lambdaFetchStores"),
		Environment: &map[string]*string{
			"API_ACTION":       jsii.String(os.Getenv("SPIDER42_FETCH_ACTION")),
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        storeQueue.QueueUrl(),
			"TABLE_NAME":       storesTable.TableName(),
			"SNS_ARN":          spider42JobDone.TopicArn(),
			"DIST_URL":         dist.DomainName(),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(storeQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(2),
			}),
		},
		RecursiveLoop: awslambda.RecursiveLoop_ALLOW,
		Role:          lambdaRole,
		Runtime:       awslambda.Runtime_PROVIDED_AL2023(),
		Timeout:       awscdk.Duration_Seconds(jsii.Number(90)),
	})
	lambdaFetchUpdates := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("spider42FetchUpdates"), &awscdklambdagoalpha.GoFunctionProps{
		Architecture: awslambda.Architecture_ARM_64(),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOARCH":      aws.String("arm64"),
				"CGO_ENABLED": aws.String("0"),
			},
			GoBuildFlags: jsii.Strings(`-ldflags "-s -w"`),
		},
		Entry: jsii.String("./lambdaFetchUpdates"),
		Environment: &map[string]*string{
			"API_ACTION":       jsii.String(os.Getenv("SPIDER42_UPDATE_ACTION")),
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        updateQueue.QueueUrl(),
			"TABLE_NAME":       storesUpdatesTable.TableName(),
			"SNS_ARN":          spider42JobDone.TopicArn(),
			"DIST_URL":         dist.DomainName(),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(updateQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(2),
			}),
		},
		RecursiveLoop: awslambda.RecursiveLoop_ALLOW,
		Role:          lambdaRole,
		Runtime:       awslambda.Runtime_PROVIDED_AL2023(),
		Timeout:       awscdk.Duration_Seconds(jsii.Number(90)),
	})

	// secret
	spider42Secret := awssecretsmanager.NewSecret(stack, jsii.String("Spider42SecretsManager"), &awssecretsmanager.SecretProps{
		Description:       jsii.String("A secret for Spider42"),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		SecretName:        jsii.String("Spider42Secret"),
		SecretStringValue: awscdk.SecretValue_UnsafePlainText(jsii.String(os.Getenv("SPIDER42_SECRET"))),
	})

	spider42Secret.GrantRead(lambdaRole, nil)

	// log schedule output
	awscdk.NewCfnOutput(stack, jsii.String("spider42StoreSchedule"), &awscdk.CfnOutputProps{
		Description: jsii.String("Spider42 Store Schedule"),
		Value:       storeSchedule.ScheduleName(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("spider42UpdateSchedule"), &awscdk.CfnOutputProps{
		Description: jsii.String("Spider42 Update Schedule"),
		Value:       updateSchedule.ScheduleName(),
	})

	// log queue URLs
	awscdk.NewCfnOutput(stack, jsii.String("spider42StoreQueueUrl"), &awscdk.CfnOutputProps{
		Description: jsii.String("Store SQS URL"),
		Value:       storeQueue.QueueUrl(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("spider42UpdateQueueUrl"), &awscdk.CfnOutputProps{
		Description: jsii.String("Update SQS URL"),
		Value:       updateQueue.QueueUrl(),
	})

	// log lambda function ARN
	awscdk.NewCfnOutput(stack, jsii.String("lambdaFetchStoresArn"), &awscdk.CfnOutputProps{
		Description: jsii.String("Lambda Fetch Stores function ARN"),
		Value:       lambdaFetchStores.FunctionArn(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("lambdaFetchUpdatesArn"), &awscdk.CfnOutputProps{
		Description: jsii.String("Lambda Fetch Updates function ARN"),
		Value:       lambdaFetchUpdates.FunctionArn(),
	})

	awscdk.NewCfnOutput(stack, jsii.String("spider42CloudFrontURL"), &awscdk.CfnOutputProps{
		Description: jsii.String("Spider42 CloudFront URL"),
		Value:       dist.DistributionDomainName(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewCdkSpider42Stack(app, "CdkSpider42Stack", &CdkSpider42StackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// `env` determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	//return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
