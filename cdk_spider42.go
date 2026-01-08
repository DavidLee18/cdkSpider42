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
			MaxReceiveCount: jsii.Number(50),
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
			MaxReceiveCount: jsii.Number(50),
			Queue:           updateDeadQueue,
		},
		EnforceSSL:             jsii.Bool(true),
		QueueName:              jsii.String("Spider42UpdateQueue"),
		ReceiveMessageWaitTime: awscdk.Duration_Seconds(jsii.Number(20)),
		RetentionPeriod:        awscdk.Duration_Hours(jsii.Number(12)),
		VisibilityTimeout:      awscdk.Duration_Seconds(jsii.Number(90)),
	})

	// create EventBridge scheduler
	when := time.Now().UTC()
	if when.Hour() > 10 || (when.Hour() == 10 && when.Minute() >= 30) || true {
		when = time.Date(when.Year(), when.Month(), when.Day()+1, 10, 30, 0, 0, time.UTC)
	} else {
		when = time.Date(when.Year(), when.Month(), when.Day(), 10, 30, 0, 0, time.UTC)
	}
	scheduleGroup := awsscheduler.NewScheduleGroup(stack, jsii.String("Spider42ScheduleGroup"), &awsscheduler.ScheduleGroupProps{
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		ScheduleGroupName: jsii.String("Spider42ScheduleGroup"),
	})
	storeSchedule := awsscheduler.NewSchedule(stack, jsii.String("Spider42StoreSchedule"), &awsscheduler.ScheduleProps{
		Enabled:       jsii.Bool(false),
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
			Input:           awsscheduler.ScheduleTargetInput_FromText(jsii.String("{\"Start\": [0, 999]}")),
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
			Input:           awsscheduler.ScheduleTargetInput_FromText(jsii.String("{\"Start\": [0, 999]}")),
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
			awsiam.ManagedPolicy_FromManagedPolicyArn(stack, aws.String("CloudWatchLogsFullAccess"), aws.String("arn:aws:iam::aws:policy/CloudWatchLogsFullAccess")),
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

	// create lambda functions
	lambdaFetchStores := awslambda.NewFunction(stack, jsii.String("spider42FetchStores"), &awslambda.FunctionProps{
		Architecture:           awslambda.Architecture_ARM_64(),
		Code:                   awslambda.AssetCode_FromAsset(jsii.String(os.Getenv("PWD")+"/lambda/target/lambda/spider42_fetch_stores"), nil),
		DeadLetterQueueEnabled: jsii.Bool(true),
		Environment: &map[string]*string{
			"API_ACTION":       jsii.String(os.Getenv("SPIDER42_FETCH_ACTION")),
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        storeQueue.QueueUrl(),
			"TABLE_NAME":       storesTable.TableName(),
			"SNS_ARN":          spider42JobDone.TopicArn(),
			"EMAIL_SUBJECT":    jsii.String(os.Getenv("EMAIL_STORE_SUBJECT")),
			"EMAIL_MESSAGE":    jsii.String(os.Getenv("EMAIL_STORE_MESSAGE") + *dist.DomainName() + "/stores.xlsx"),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(storeQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(2),
				MetricsConfig: &awslambda.MetricsConfig{
					Metrics: &[]awslambda.MetricType{
						awslambda.MetricType_EVENT_COUNT,
					},
				},
				ReportBatchItemFailures: jsii.Bool(true),
			}),
		},
		FunctionName:    jsii.String("Spider42FetchStores"),
		Handler:         jsii.String("bootstrap"),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_498_0(),
		RecursiveLoop:   awslambda.RecursiveLoop_ALLOW,
		Runtime:         awslambda.Runtime_PROVIDED_AL2023(),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(90)),
		Tracing:         awslambda.Tracing_ACTIVE,
	})
	lambdaFetchUpdates := awslambda.NewFunction(stack, jsii.String("spider42FetchUpdates"), &awslambda.FunctionProps{
		Architecture:           awslambda.Architecture_ARM_64(),
		Code:                   awslambda.AssetCode_FromAsset(jsii.String(os.Getenv("PWD")+"/lambda/target/lambda/spider42_fetch_updates"), nil),
		DeadLetterQueueEnabled: jsii.Bool(true),
		Environment: &map[string]*string{
			"API_ACTION":       jsii.String(os.Getenv("SPIDER42_UPDATE_ACTION")),
			"BUCKET_NAME":      spider42Bucket.BucketName(),
			"LIMIT_TABLE_NAME": limitTable.TableName(),
			"QUEUE_URL":        updateQueue.QueueUrl(),
			"TABLE_NAME":       storesUpdatesTable.TableName(),
			"SNS_ARN":          spider42JobDone.TopicArn(),
			"EMAIL_SUBJECT":    jsii.String(os.Getenv("EMAIL_UPDATE_SUBJECT")),
			"EMAIL_MESSAGE":    jsii.String(os.Getenv("EMAIL_UPDATE_MESSAGE") + *dist.DomainName() + "/updates.xlsx"),
		},
		Events: &[]awslambda.IEventSource{
			awslambdaeventsources.NewSqsEventSource(updateQueue, &awslambdaeventsources.SqsEventSourceProps{
				BatchSize:      jsii.Number(1),
				Enabled:        jsii.Bool(true),
				MaxConcurrency: jsii.Number(2),
				MetricsConfig: &awslambda.MetricsConfig{
					Metrics: &[]awslambda.MetricType{
						awslambda.MetricType_EVENT_COUNT,
					},
				},
				ReportBatchItemFailures: jsii.Bool(true),
			}),
		},
		FunctionName:    jsii.String("Spider42FetchUpdates"),
		Handler:         jsii.String("bootstrap"),
		InsightsVersion: awslambda.LambdaInsightsVersion_VERSION_1_0_498_0(),
		RecursiveLoop:   awslambda.RecursiveLoop_ALLOW,
		Runtime:         awslambda.Runtime_PROVIDED_AL2023(),
		Timeout:         awscdk.Duration_Seconds(jsii.Number(90)),
		Tracing:         awslambda.Tracing_ACTIVE,
	})

	lambdaFetchStores.AddEnvironment(jsii.String("LAMBDA_ARN"), jsii.Sprintf("arn:aws:lambda:%s:%s:function:Spider42FetchStores", os.Getenv("CDK_DEFAULT_REGION"), os.Getenv("CDK_DEFAULT_ACCOUNT")), nil)
	lambdaFetchUpdates.AddEnvironment(jsii.String("LAMBDA_ARN"), jsii.Sprintf("arn:aws:lambda:%s:%s:function:Spider42FetchUpdates", os.Getenv("CDK_DEFAULT_REGION"), os.Getenv("CDK_DEFAULT_ACCOUNT")), nil)

	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("s3:PutObject"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			spider42Bucket.BucketArn(),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("kms:GenerateDataKey"),
			jsii.String("kms:Decrypt"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:kms:*:*:key/*"),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			storeQueue.QueueArn(),
			storeDeadQueue.QueueArn(),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("secretsmanager:DescribeSecret"),
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("secretsmanager:ListSecretVersionIds"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:secretsmanager:*:*:secret:*"),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("dynamodb:Query"),
			jsii.String("dynamodb:PutItem"),
			jsii.String("dynamodb:UpdateItem"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:dynamodb:*:*:table/*"),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("events:PutRule"),
			jsii.String("events:PutTargets"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:events:*:*:rule/*"),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("s3:PutObject"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:s3:::*"),
		},
	}))
	lambdaFetchStores.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("SNS:Publish"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:sns:*:*:*"),
		},
	}))

	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("s3:PutObject"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			spider42Bucket.BucketArn(),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("kms:GenerateDataKey"),
			jsii.String("kms:Decrypt"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:kms:*:*:key/*"),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			updateQueue.QueueArn(),
			updateDeadQueue.QueueArn(),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("secretsmanager:DescribeSecret"),
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("secretsmanager:ListSecretVersionIds"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:secretsmanager:*:*:secret:*"),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("dynamodb:Query"),
			jsii.String("dynamodb:PutItem"),
			jsii.String("dynamodb:UpdateItem"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:dynamodb:*:*:table/*"),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("events:PutRule"),
			jsii.String("events:PutTargets"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:events:*:*:rule/*"),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("s3:PutObject"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:s3:::*"),
		},
	}))
	lambdaFetchUpdates.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("SNS:Publish"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			jsii.String("arn:aws:sns:*:*:*"),
		},
	}))

	storeQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("lambda:InvokeFunction"),
		},
		Resources: &[]*string{
			lambdaFetchStores.FunctionArn(),
		},
		Effect: awsiam.Effect_ALLOW,
		Principals: &[]awsiam.IPrincipal{
			awsiam.NewServicePrincipal(jsii.String("sqs.amazonaws.com"), &awsiam.ServicePrincipalOpts{
				Region: jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
			}),
		},
		Conditions: &map[string]any{
			"ArnEquals": &map[string]any{
				"aws:SourceArn": storeQueue.QueueArn(),
			},
		},
	}))

	updateQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("lambda:InvokeFunction"),
		},
		Resources: &[]*string{
			lambdaFetchUpdates.FunctionArn(),
		},
		Effect: awsiam.Effect_ALLOW,
		Principals: &[]awsiam.IPrincipal{
			awsiam.NewServicePrincipal(jsii.String("sqs.amazonaws.com"), &awsiam.ServicePrincipalOpts{
				Region: jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
			}),
		},
		Conditions: &map[string]any{
			"ArnEquals": &map[string]any{
				"aws:SourceArn": updateQueue.QueueArn(),
			},
		},
	}))

	storeQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			storeQueue.QueueArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchStores.Role(),
		},
	}))
	storeDeadQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			storeDeadQueue.QueueArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchStores.Role(),
		},
	}))

	updateQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			updateQueue.QueueArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchUpdates.Role(),
		},
	}))
	updateDeadQueue.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("sqs:ChangeMessageVisibility"),
			jsii.String("sqs:DeleteMessage"),
			jsii.String("sqs:ReceiveMessage"),
			jsii.String("sqs:GetQueueAttributes"),
			jsii.String("sqs:GetQueueUrl"),
			jsii.String("sqs:SendMessage"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			updateDeadQueue.QueueArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchUpdates.Role(),
		},
	}))

	// secret
	spider42Secret := awssecretsmanager.NewSecret(stack, jsii.String("Spider42SecretsManager"), &awssecretsmanager.SecretProps{
		Description:       jsii.String("A secret for Spider42"),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		SecretName:        jsii.String("Spider42Secret"),
		SecretStringValue: awscdk.SecretValue_UnsafePlainText(jsii.String(os.Getenv("SPIDER42_SECRET"))),
	})

	spider42Secret.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("secretsmanager:DescribeSecret"),
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("secretsmanager:ListSecretVersionIds"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			spider42Secret.SecretArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchStores.Role(),
		},
	}))

	spider42Secret.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: &[]*string{
			jsii.String("secretsmanager:DescribeSecret"),
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("secretsmanager:ListSecretVersionIds"),
		},
		Effect: awsiam.Effect_ALLOW,
		Resources: &[]*string{
			spider42Secret.SecretArn(),
		},
		Principals: &[]awsiam.IPrincipal{
			lambdaFetchUpdates.Role(),
		},
	}))

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
