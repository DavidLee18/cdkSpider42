use aws_lambda_events::{
    apigw::ApiGatewayProxyRequest, eventbridge::EventBridgeEvent, sqs::SqsEvent,
};
use aws_sdk_dynamodb::types::{Put, TransactWriteItem};
use aws_sdk_sqs::types::QueueAttributeName;
use itertools::Itertools;
use lambda_runtime::{Error, run, service_fn, tracing};
use spider42::{
    Limit, MyError, Payload, Store, StoreInfoResultCode, StoreInfos, delete_from_queue, get_secret,
    retry_later, retry_tomorrow, send_queue,
};
use std::{collections::HashMap, env};
use uuid::Uuid;

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing::init_default_subscriber();
    let config = aws_config::load_from_env().await;
    let sqs = aws_sdk_sqs::Client::new(&config);
    let db = aws_sdk_dynamodb::Client::new(&config);
    let secretsm = aws_sdk_secretsmanager::Client::new(&config);
    let sched = aws_sdk_scheduler::Client::new(&config);
    let s3 = aws_sdk_s3::Client::new(&config);
    let sns = aws_sdk_sns::Client::new(&config);

    run(service_fn(async |e| {
        spider42_fetch_stores(e, &sqs, &db, &secretsm, &sched, &s3, &sns).await
    }))
    .await
}

spider42::handle!(
    spider42_fetch_stores,
    StoreInfos,
    Store,
    approval_date,
    "BSSH_NM",
    "PRMS_DT",
    "stores.xlsx"
);
