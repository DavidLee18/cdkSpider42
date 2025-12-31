package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

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

	// create AmazonDynamoDBFullAccess role
	lambdaRole := awsiam.NewRole(stack, aws.String("spider42LambdaRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(aws.String("lambda.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromManagedPolicyArn(stack, aws.String("AmazonDynamoDBFullAccess"), aws.String("arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess")),
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

	// create EventBridge
	eventBus := awsevents.NewEventBus(stack, jsii.String("Spider42EventBus"), &awsevents.EventBusProps{
		EventBusName: jsii.String("Spider42EventBus"),
	})
	rule := awsevents.NewRule(stack, jsii.String("Spider42EventBusRule"), &awsevents.RuleProps{
		EventBus: eventBus,
		Schedule: awsevents.Schedule_Expression(jsii.String("cron(30 10 * 12 ? 2025)")),
	})
	rule.AddTarget(awseventstargets.NewSqsQueue(storeQueue, &awseventstargets.SqsQueueProps{}))
	rule.AddTarget(awseventstargets.NewSqsQueue(updateQueue, &awseventstargets.SqsQueueProps{}))

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
		SortKey: &awsdynamodb.Attribute{
			Name: aws.String("ID"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Spider42Updates"),
	})
	limitTable := awsdynamodb.NewTable(stack, jsii.String("Spider42Limits"), &awsdynamodb.TableProps{
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		PartitionKey: &awsdynamodb.Attribute{
			Name: aws.String("table_name"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true),
		},
		SortKey: &awsdynamodb.Attribute{
			Name: aws.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Spider42Limits"),
	})

	spider42Bucket := awss3.NewBucket(stack, jsii.String("Spider42Bucket"), &awss3.BucketProps{
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		EnforceSSL:        jsii.Bool(true),
		Versioned:         jsii.Bool(true),
	})

	// create lambda function
	lambdaFetchStores := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("spider42FetchStores"), &awscdklambdagoalpha.GoFunctionProps{
		Architecture: awslambda.Architecture_ARM_64(),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOARCH":      aws.String("arm64"),
				"CGO_ENABLED": aws.String("0"),
			},
			GoBuildFlags: jsii.Strings(`-ldflags "-s -w"`),
		},
		Entry: jsii.String("./lambdaFetch"),
		Environment: &map[string]*string{
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        storeQueue.QueueUrl(),
			"TABLE_NAME":       storesTable.TableName(),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(storeQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(1),
			}),
		},
		Role:    lambdaRole,
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Timeout: awscdk.Duration_Seconds(jsii.Number(90)),
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
		Entry: jsii.String("./lambdaFetch"),
		Environment: &map[string]*string{
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        storeQueue.QueueUrl(),
			"TABLE_NAME":       storesUpdatesTable.TableName(),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(storeQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(1),
			}),
		},
		Role:    lambdaRole,
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Timeout: awscdk.Duration_Seconds(jsii.Number(90)),
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
