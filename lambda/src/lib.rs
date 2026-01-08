use rust_xlsxwriter::XlsxSerialize;
use uuid::Uuid;

#[derive(Debug)]
pub enum MyError {
    UnsupportedEventType,
    EmptyBody,
    SqsParseError(String),
    StoreInfoError(StoreInfoResult),
    FailedToSendMessagesToQueue(Vec<aws_sdk_sqs::types::BatchResultErrorEntry>),
    NoSecrets,
    GetQueueAttributesError,
    QueueSizeParseError(std::num::ParseIntError),
    NoReceiptHandle,
    StoreInfosEmpty,
    NoLimit,
}

impl std::fmt::Display for MyError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            MyError::UnsupportedEventType => write!(f, "Unsupported event type"),
            MyError::EmptyBody => write!(f, "Empty body in sqs event record"),
            MyError::SqsParseError(err) => write!(f, "Failed to parse SQS event record: {}", err),
            MyError::StoreInfoError(err) => write!(f, "Failed to get store info: {:?}", err),
            MyError::FailedToSendMessagesToQueue(errors) => {
                write!(f, "Failed to send messages to queue: {:?}", errors)
            }
            MyError::NoSecrets => write!(f, "No secrets found"),
            MyError::GetQueueAttributesError => write!(f, "Failed to get queue attributes"),
            MyError::QueueSizeParseError(err) => write!(f, "Failed to parse queue size: {}", err),
            MyError::NoReceiptHandle => write!(f, "No receipt handle found"),
            MyError::StoreInfosEmpty => write!(f, "No store infos found"),
            MyError::NoLimit => write!(f, "No limit found"),
        }
    }
}

impl std::error::Error for MyError {}

#[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize, XlsxSerialize)]
#[xlsx(table_default)]
pub struct Store {
    #[serde(rename = "LCNS_NO")]
    #[xlsx(rename = "인허가번호")]
    pub license_num: String,
    #[serde(rename = "BSSH_NM")]
    #[xlsx(rename = "업소명")]
    pub name: String,
    #[serde(rename = "ADDR")]
    #[xlsx(rename = "주소")]
    pub address: String,
    #[serde(rename = "TELNO")]
    #[xlsx(rename = "전화번호")]
    pub phone_number: String,
    #[serde(rename = "INDUTY_CD_NM")]
    #[xlsx(rename = "업종")]
    pub category: String,
    #[serde(rename = "PRSDNT_NM")]
    #[xlsx(rename = "대표자명")]
    pub owner_name: String,
    #[serde(rename = "PRMS_DT")]
    #[xlsx(rename = "허가일자")]
    pub approval_date: String,
    #[serde(rename = "ID")]
    #[serde(skip_serializing_if = "Option::is_none")]
    #[xlsx(skip)]
    pub id: Option<Uuid>,
}

#[derive(Debug, PartialEq)]
pub enum StoreInfoResultCode {
    Info000,
    Info100,
    Info200,
    Info300,
    Info400,
    Error300,
    Error301,
    Error310,
    Error331,
    Error332,
    Error334,
    Error336,
    Error500,
    Error601,
}

impl serde::Serialize for StoreInfoResultCode {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        match self {
            StoreInfoResultCode::Info000 => serializer.serialize_str("INFO-000"),
            StoreInfoResultCode::Info100 => serializer.serialize_str("INFO-100"),
            StoreInfoResultCode::Info200 => serializer.serialize_str("INFO-200"),
            StoreInfoResultCode::Info300 => serializer.serialize_str("INFO-300"),
            StoreInfoResultCode::Info400 => serializer.serialize_str("INFO-400"),
            StoreInfoResultCode::Error300 => serializer.serialize_str("ERROR-300"),
            StoreInfoResultCode::Error301 => serializer.serialize_str("ERROR-301"),
            StoreInfoResultCode::Error310 => serializer.serialize_str("ERROR-310"),
            StoreInfoResultCode::Error331 => serializer.serialize_str("ERROR-331"),
            StoreInfoResultCode::Error332 => serializer.serialize_str("ERROR-332"),
            StoreInfoResultCode::Error334 => serializer.serialize_str("ERROR-334"),
            StoreInfoResultCode::Error336 => serializer.serialize_str("ERROR-336"),
            StoreInfoResultCode::Error500 => serializer.serialize_str("ERROR-500"),
            StoreInfoResultCode::Error601 => serializer.serialize_str("ERROR-601"),
        }
    }
}

impl<'de> serde::Deserialize<'de> for StoreInfoResultCode {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        match s.as_str() {
            "INFO-000" => Ok(StoreInfoResultCode::Info000),
            "INFO-100" => Ok(StoreInfoResultCode::Info100),
            "INFO-200" => Ok(StoreInfoResultCode::Info200),
            "INFO-300" => Ok(StoreInfoResultCode::Info300),
            "INFO-400" => Ok(StoreInfoResultCode::Info400),
            "ERROR-300" => Ok(StoreInfoResultCode::Error300),
            "ERROR-301" => Ok(StoreInfoResultCode::Error301),
            "ERROR-310" => Ok(StoreInfoResultCode::Error310),
            "ERROR-331" => Ok(StoreInfoResultCode::Error331),
            "ERROR-332" => Ok(StoreInfoResultCode::Error332),
            "ERROR-334" => Ok(StoreInfoResultCode::Error334),
            "ERROR-336" => Ok(StoreInfoResultCode::Error336),
            "ERROR-500" => Ok(StoreInfoResultCode::Error500),
            "ERROR-601" => Ok(StoreInfoResultCode::Error601),
            _ => Err(serde::de::Error::custom(format!(
                "Invalid StoreInfoResultCode: {}",
                s
            ))),
        }
    }
}

#[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct StoreInfoResult {
    #[serde(rename = "MSG")]
    pub msg: String,
    #[serde(rename = "CODE")]
    pub code: StoreInfoResultCode,
}

#[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct StoreInfos {
    pub total_count: String,
    #[serde(default)]
    pub row: Vec<Store>,
    #[serde(rename = "RESULT")]
    pub result: StoreInfoResult,
}

#[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize, XlsxSerialize)]
pub struct StoreUpdate {
    #[serde(rename = "LCNS_NO")]
    #[xlsx(rename = "인허가번호")]
    pub license_num: String,
    #[serde(rename = "BSSH_NM")]
    #[xlsx(rename = "업소명")]
    pub name: String,
    #[serde(rename = "SITE_ADDR")]
    #[xlsx(rename = "주소")]
    pub address: String,
    #[serde(rename = "TELNO")]
    #[xlsx(rename = "전화번호")]
    pub phone_number: String,
    #[serde(rename = "INDUTY_CD_NM")]
    #[xlsx(rename = "업종명")]
    pub category: String,
    #[serde(rename = "CHNG_DT")]
    #[xlsx(rename = "변경일자")]
    pub update_date: String,
    #[serde(rename = "CHNG_BF_CN")]
    #[xlsx(rename = "변경전내용")]
    pub before: String,
    #[serde(rename = "CHNG_AF_CN")]
    #[xlsx(rename = "변경후내용")]
    pub after: String,
    #[serde(rename = "CHNG_PRVNS")]
    #[xlsx(rename = "변경사유")]
    pub update_reason: String,
    #[serde(rename = "ID")]
    #[serde(skip_serializing_if = "Option::is_none")]
    #[xlsx(skip)]
    pub id: Option<Uuid>,
}

#[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct StoreUpdates {
    pub total_count: String,
    #[serde(default)]
    pub row: Vec<StoreUpdate>,
    #[serde(rename = "RESULT")]
    pub result: StoreInfoResult,
}

#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub enum Payload {
    Start(u32, u32),
    Between(u32, u32),
    End,
}

#[derive(Debug, PartialEq, PartialOrd, serde::Serialize, serde::Deserialize)]
pub struct Limit {
    #[serde(rename = "TABLE_NAME")]
    pub table_name: String,
    #[serde(rename = "ID")]
    pub id: Uuid,
    #[serde(rename = "FIELD_NAME")]
    pub field_name: String,
    #[serde(rename = "VALUE")]
    pub value: String,
    #[serde(default, rename = "LAST_IDX", skip_serializing_if = "Option::is_none")]
    pub last_idx: Option<(u32, u32)>,
}

pub async fn get_secret(
    secretsm: &aws_sdk_secretsmanager::Client,
    key: &str,
) -> Result<String, lambda_runtime::Error> {
    let response = secretsm.get_secret_value().secret_id(key).send().await?;
    response.secret_string.ok_or(Box::new(MyError::NoSecrets))
}

pub async fn send_queue(
    sqs: &aws_sdk_sqs::Client,
    queue: &str,
    p: Payload,
) -> Result<(), lambda_runtime::Error> {
    let res = sqs
        .send_message_batch()
        .queue_url(queue)
        .entries(
            aws_sdk_sqs::types::SendMessageBatchRequestEntry::builder()
                .id(Uuid::new_v4().to_string())
                .message_body(serde_json::to_string(&p)?)
                .build()?,
        )
        .send()
        .await?;
    if res.failed().len() > 0 {
        lambda_runtime::tracing::error!("Failed to send messages to queue");
        Err(Box::new(MyError::FailedToSendMessagesToQueue(res.failed)))
    } else {
        Ok(())
    }
}

pub async fn delete_from_queue(
    sqs: &aws_sdk_sqs::Client,
    queue: &str,
    handle: &Option<String>,
) -> Result<(), lambda_runtime::Error> {
    sqs.delete_message()
        .queue_url(queue)
        .receipt_handle(handle.as_ref().ok_or(MyError::NoReceiptHandle)?)
        .send()
        .await?;
    Ok(())
}

pub async fn retry_tomorrow(
    ev: &aws_sdk_eventbridge::Client,
    payload: Payload,
) -> Result<(), lambda_runtime::Error> {
    let now = chrono::Utc::now();
    let mut today_time = now.clone();
    let today_time_res = today_time.with_time(chrono::NaiveTime::from_hms_opt(10, 30, 0).unwrap());
    if let chrono::LocalResult::Single(t) = today_time_res {
        today_time = t;
    } else {
        panic!("Failed to set time");
    }
    let diff = today_time - now;
    let diff2 = today_time + chrono::Duration::days(1) - now;
    let dest_time;
    if diff.abs() < diff2.abs() {
        dest_time = today_time;
    } else {
        dest_time = today_time + chrono::Duration::days(1);
    }
    let rule_name = format!("spider42_fetch_{}", dest_time.timestamp_micros());
    ev.put_rule()
        .name(&rule_name)
        .schedule_expression(format!("cron({})", dest_time.format("%M %H %d %m ? %Y")))
        .state(aws_sdk_eventbridge::types::RuleState::Enabled)
        .send()
        .await?;
    ev.put_targets()
        .rule(&rule_name)
        .targets(
            aws_sdk_eventbridge::types::Target::builder()
                .id(format!("{}_target", rule_name))
                .arn(std::env::var("LAMBDA_ARN")?)
                .input(serde_json::to_string(&payload)?)
                .build()?,
        )
        .send()
        .await?;
    Ok(())
}

#[macro_export]
macro_rules! match_result_code {
    ($info:ident, $from: ident, $until: ident, $ev:ident, $sqs:ident, $queue:ident, $handle:expr) => {
        match $info.result.code {
            StoreInfoResultCode::Info000 => {
                if $info.row.len() != $info.total_count.parse::<usize>()? {
                    tracing::warn!(
                        "store infos' count mismatch: expected: {}, got: {}",
                        $info.total_count.parse::<usize>()?,
                        $info.row.len()
                    )
                }
            }
            StoreInfoResultCode::Info300 => {
                tracing::warn!("API calls' limit reached; will retry tomorrow");
                retry_tomorrow($ev, Payload::Start($from, $until)).await?;
                send_queue(&$sqs, &$queue, Payload::End).await?;
                delete_from_queue(&$sqs, &$queue, &$handle).await?;
                return Ok(());
            }
            StoreInfoResultCode::Info200 | StoreInfoResultCode::Error332 => {
                let width = $until - $from;
                if width > 1 {
                    send_queue(
                        &$sqs,
                        &$queue,
                        Payload::Between($from, $until + (width / 2)),
                    )
                    .await?;
                } else {
                    tracing::warn!("cannot fetch more; terminating...");
                    retry_tomorrow($ev, Payload::Start($from, $from + 1000)).await?;
                    send_queue(&$sqs, &$queue, Payload::End).await?;
                }
                delete_from_queue(&$sqs, &$queue, &$handle).await?;
                return Ok(());
            }
            StoreInfoResultCode::Error336 => {
                tracing::warn!("cannot fetch {} items; retrying...", $until - $from);
                send_queue(&$sqs, &$queue, Payload::Between($from, $from + 1000)).await?;
                delete_from_queue(&$sqs, &$queue, &$handle).await?;
                return Ok(());
            }
            StoreInfoResultCode::Info100
            | StoreInfoResultCode::Info400
            | StoreInfoResultCode::Error300
            | StoreInfoResultCode::Error301
            | StoreInfoResultCode::Error310
            | StoreInfoResultCode::Error331
            | StoreInfoResultCode::Error334
            | StoreInfoResultCode::Error500
            | StoreInfoResultCode::Error601 => {
                return Err(Box::new(MyError::StoreInfoError($info.result))
                    as Box<dyn std::error::Error + Send + Sync>);
            }
        }
    };
}

#[macro_export]
macro_rules! inject_id {
    ($items:expr) => {
        $items.iter_mut().for_each(
            |i: &mut HashMap<String, ::aws_sdk_dynamodb::types::AttributeValue>| {
                i.insert(
                    "ID".to_string(),
                    ::aws_sdk_dynamodb::types::AttributeValue::S(
                        ::uuid::Uuid::now_v7().to_string(),
                    ),
                );
            },
        );
    };
}

#[macro_export]
macro_rules! atomic_puts {
    ($items:ident, $db:ident, $name:literal, $no:literal) => {
        if $name == "BSSH_NM" {
            let table_name = env::var("TABLE_NAME")?;
            for ws in $items.into_iter().chunks(100).into_iter().map(|is| is.map( |i| { Put::builder()
                    .table_name(&table_name)
                    .set_item(Some(i))
                    .condition_expression(
                        format!("attribute_not_exists(ID) AND (attribute_not_exists({}) AND attribute_not_exists({}))", $name, $no),
                    )
                    .build()
            }).collect::<Result<Vec<_>, _>>()).collect::<Result<Vec<_>, _>>()? {
                let _ = $db.transact_write_items().set_transact_items(Some(
                        ws.into_iter()
                            .map(|w| {
                                TransactWriteItem::builder().put(w).build()
                            })
                            .collect(),
                    ))
                    .send()
                    .await?;
            }
        }
    }
}

#[macro_export]
macro_rules! handle {
    ($name: ident, $t:ty, $ts:ty, $date:ident, $pk:literal, $limfield:literal, $file_name:literal) => {
        async fn $name(
            event: lambda_runtime::LambdaEvent<serde_json::Value>,
            sqs: &aws_sdk_sqs::Client,
            db: &aws_sdk_dynamodb::Client,
            secretsm: &aws_sdk_secretsmanager::Client,
            ev: &aws_sdk_eventbridge::Client,
            s3: &aws_sdk_s3::Client,
            sns: &aws_sdk_sns::Client,
        ) -> Result<(), Error> {
            let payload = event.payload;
            tracing::info!("Payload: {:?}", payload);

            let queue = env::var("QUEUE_URL")?;
            let table_name = env::var("TABLE_NAME")?;
            let limit_table = env::var("LIMIT_TABLE_NAME")?;
            let api_action = env::var("API_ACTION")?;
            let sns_arn = env::var("SNS_ARN")?;

            if let Ok(qevent) = serde_json::from_value::<SqsEvent>(payload.clone()) {
                let xl_bucket = env::var("BUCKET_NAME")?;

                for record in qevent.records {
                    let body = record.body.ok_or(MyError::EmptyBody)?;
                    let de = &mut serde_json::Deserializer::from_str(&body);
                    let scrt = get_secret(&secretsm, "Spider42Secret").await?;
                    match serde_path_to_error::deserialize(de)? {
                        Payload::Start(from, until) => {
                            let url = format!(
                                "http://openapi.foodsafetykorea.go.kr/api/{}/{}/json/{}/{}",
                                scrt,
                                api_action,
                                from,
                                until
                            );
                            tracing::info!("sending request to: {}", url);
                            match minreq::get(&url).with_timeout(40).send() {
                                Ok(res) => {
                                    let txt = res.as_str()?;
                                    tracing::info!("raw response: {}", txt);
                                    if !txt.contains("alert(") && !txt.contains("<script") {
                                        let mut res: serde_json::Value = serde_json::from_str(txt)?;
                                        let raw = serde_json::to_string(&res[&api_action].take())?;
                                        let de = &mut serde_json::Deserializer::from_str(&raw);
                                        let sinfo: $ts = serde_path_to_error::deserialize(de)?;
                                        ::spider42::match_result_code!(sinfo, from, until, ev, sqs, queue, record.receipt_handle);

                                        let _ = db
                                            .transact_write_items()
                                            .transact_items(
                                                TransactWriteItem::builder()
                                                    .put(
                                                        sinfo
                                                            .row
                                                            .get(0)
                                                            .ok_or(MyError::StoreInfosEmpty)
                                                            .map(|info| {
                                                            Put::builder()
                                                                .table_name(&limit_table)
                                                                .set_item(
                                                                    serde_dynamo::to_item(Limit {
                                                                        table_name: table_name.clone(),
                                                                        id: Uuid::now_v7(),
                                                                        field_name: String::from($limfield),
                                                                        value: info.$date.clone(),
                                                                        last_idx: Some((from, until)),
                                                                    })
                                                                    .ok(),
                                                                )
                                                                .condition_expression(
                                                                    "attribute_not_exists(id)",
                                                                )
                                                                .build()
                                                        })??,
                                                    )
                                                    .build(),
                                            )
                                            .send()
                                            .await?;

                                        let mut items = sinfo
                                            .row
                                            .into_iter()
                                            .filter(|i| {
                                                if i.$date.is_empty() {
                                                    tracing::warn!(
                                                        "Approval date is empty for store {:?}",
                                                        i
                                                    );
                                                    false
                                                } else {
                                                    true
                                                }
                                            })
                                            .map(|i| serde_dynamo::to_item(i))
                                            .collect::<Result<Vec<_>, _>>()?;

                                        ::spider42::inject_id!(items);
                                        ::spider42::atomic_puts!(items, db, $pk, "LCNS_NO");

                                        let width = until - from;

                                        send_queue(
                                            &sqs,
                                            &queue,
                                            Payload::Between(from + width, until + width),
                                        )
                                        .await?;

                                        delete_from_queue(&sqs, &queue, &record.receipt_handle).await?;
                                    } else {
                                        tracing::error!(
                                            "Error from API response (\"alert()\" or \"<script>\" detected): {:?}",
                                            txt
                                        );
                                        send_queue(&sqs, &queue, Payload::Start(from, until)).await?;
                                    }
                                }
                                Err(minreq::Error::IoError(e))
                                    if e.kind() == std::io::ErrorKind::TimedOut =>
                                {
                                    tracing::warn!("API request timed out; re-queueing..");
                                    send_queue(&sqs, &queue, Payload::Start(from, until)).await?;
                                }
                                Err(e) => {
                                    tracing::error!("Error sending API request: {:?}", e);
                                }
                            }
                        }
                        Payload::Between(from, until) => {
                            let url = format!(
                                "http://openapi.foodsafetykorea.go.kr/api/{}/{}/json/{}/{}",
                                scrt,
                                api_action,
                                from,
                                until
                            );
                            tracing::info!("sending request to: {}", url);
                            match minreq::get(&url).with_timeout(40).send() {
                                Ok(res) => {
                                    let txt = res.as_str()?;
                                    tracing::info!("raw response: {}", txt);
                                    if !txt.contains("alert(") && !txt.contains("<script") {
                                        let mut res: serde_json::Value = serde_json::from_str(txt)?;
                                        let raw = serde_json::to_string(&res[&api_action].take())?;
                                        let de = &mut serde_json::Deserializer::from_str(&raw);
                                        let sinfo: $ts = serde_path_to_error::deserialize(de)?;
                                        ::spider42::match_result_code!(sinfo, from, until, ev, sqs, queue, record.receipt_handle);

                                        let mut items = sinfo
                                            .row
                                            .into_iter()
                                            .filter(|i| {
                                                if i.$date.is_empty() {
                                                    tracing::warn!(
                                                        "Approval date is empty for store {:?}",
                                                        i
                                                    );
                                                    false
                                                } else {
                                                    true
                                                }
                                            })
                                            .map(|i| serde_dynamo::to_item(i))
                                            .collect::<Result<Vec<_>, _>>()?;
                                        ::spider42::inject_id!(items);
                                        ::spider42::atomic_puts!(items, db, $pk, "LCNS_NO");

                                        let width = until - from;

                                        send_queue(
                                            &sqs,
                                            &queue,
                                            Payload::Between(from + width, until + width),
                                        )
                                        .await?;

                                        delete_from_queue(&sqs, &queue, &record.receipt_handle).await?;
                                    } else {
                                        tracing::error!(
                                            "Error from API response (\"alert()\" or \"<script>\" detected): {:?}",
                                            txt
                                        );
                                        send_queue(&sqs, &queue, Payload::Between(from, until)).await?;
                                    }
                                }
                                Err(minreq::Error::IoError(e))
                                    if e.kind() == std::io::ErrorKind::TimedOut =>
                                {
                                    tracing::warn!("API request timed out; re-queueing..");
                                    send_queue(&sqs, &queue, Payload::Between(from, until)).await?;
                                }
                                Err(e) => {
                                    tracing::error!("Error sending API request: {:?}", e);
                                }
                            }
                        }
                        Payload::End => {
                            tracing::info!("end of the payload, table names are: {:?}, {:?}", limit_table, table_name);
                            let res = db
                                .query()
                                .table_name(&limit_table)
                                .key_condition_expression("TABLE_NAME = :table_name")
                                .expression_attribute_values(
                                    ":table_name",
                                    aws_sdk_dynamodb::types::AttributeValue::S(table_name.clone()),
                                )
                                .scan_index_forward(false)
                                .consistent_read(true)
                                .limit(1)
                                .send()
                                .await?;

                            tracing::info!("{:?}", res);

                            if res.count < 1 {
                                return Err(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>);
                            }

                            let limit: Limit = serde_dynamo::from_item(
                                res.items
                                    .ok_or(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>)?
                                    .pop()
                                    .ok_or(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>)?,
                            )?;

                            let mut infos: Vec<$t>;

                            let mut recs = db
                                .query()
                                .table_name(&limit.table_name)
                                .key_condition_expression(format!(
                                    "{} = :val and ID >= :id",
                                    limit.field_name
                                ))
                                .expression_attribute_values(
                                    ":val",
                                    aws_sdk_dynamodb::types::AttributeValue::S(limit.value.clone()),
                                )
                                .expression_attribute_values(
                                    ":id",
                                    aws_sdk_dynamodb::types::AttributeValue::S(limit.id.to_string()),
                                )
                                .consistent_read(true)
                                .send()
                                .await?;

                            infos = serde_dynamo::from_items(recs.items.ok_or(MyError::StoreInfosEmpty)?)?;

                            while let Some(next) = recs.last_evaluated_key {
                                recs = db
                                    .query()
                                    .table_name(&limit.table_name)
                                    .key_condition_expression(format!(
                                        "{} = :val and ID >= :id",
                                        limit.field_name
                                    ))
                                    .expression_attribute_values(
                                        ":val",
                                        aws_sdk_dynamodb::types::AttributeValue::S(limit.value.clone()),
                                    )
                                    .expression_attribute_values(
                                        ":id",
                                        aws_sdk_dynamodb::types::AttributeValue::S(limit.id.to_string()),
                                    )
                                    .set_exclusive_start_key(Some(next))
                                    .consistent_read(true)
                                    .send()
                                    .await?;

                                infos.extend(serde_dynamo::from_items(
                                    recs.items.ok_or(MyError::StoreInfosEmpty)?,
                                )?);
                            }

                            let mut xl = rust_xlsxwriter::Workbook::new();
                            let wsheet = xl.add_worksheet().autofit();
                            wsheet.set_serialize_headers::<$t>(0, 0)?;
                            wsheet.serialize(&infos)?;
                            xl.save(format!("/tmp/{}", $file_name))?;

                            let res = s3
                                .put_object()
                                .bucket(&xl_bucket)
                                .key($file_name)
                                .content_type(
                                    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                                )
                                .body(std::fs::read(format!("/tmp/{}", $file_name))?.into())
                                .send()
                                .await?;
                            tracing::info!("Uploaded to S3: {:?}", res);
                            delete_from_queue(&sqs, &queue, &record.receipt_handle).await?;
                            sns
                                .publish()
                                .topic_arn(&sns_arn)
                                .subject(env::var("EMAIL_SUBJECT")?)
                                .message(env::var("EMAIL_MESSAGE")?)
                                .send()
                                .await?;
                        }
                    }
                }

                Ok(())
            } else if let Ok(_) = serde_json::from_value::<ApiGatewayProxyRequest>(payload.clone()) {
                let qstat = sqs
                    .get_queue_attributes()
                    .queue_url(&queue)
                    .attribute_names(QueueAttributeName::ApproximateNumberOfMessages)
                    .send()
                    .await?;
                let num = qstat
                    .attributes()
                    .ok_or(Box::new(MyError::GetQueueAttributesError) as Box<dyn std::error::Error+Send+Sync>)?
                    .get(&QueueAttributeName::ApproximateNumberOfMessages)
                    .ok_or(Box::new(MyError::GetQueueAttributesError) as Box<dyn std::error::Error+Send+Sync>)?;

                match num.parse::<u32>().map_err(MyError::QueueSizeParseError)? {
                    0 => send_queue(&sqs, &queue, Payload::Start(0, 999)).await,
                    _ => {
                        tracing::warn!("Queue is not empty; ignoring request...");
                        Ok(())
                    }
                }
            } else if let Ok(bevent) = serde_json::from_value::<EventBridgeEvent>(payload) {
                match serde_json::from_value::<Payload>(bevent.detail) {
                    Ok(p) => send_queue(&sqs, &queue, p).await,
                    Err(_) => {
                        let rec = db
                            .query()
                            .table_name(limit_table)
                            .key_condition_expression("TABLE_NAME = :table_name")
                            .expression_attribute_values(
                                ":table_name",
                                aws_sdk_dynamodb::types::AttributeValue::S(table_name),
                            )
                            .scan_index_forward(false)
                            .consistent_read(true)
                            .limit(1)
                            .send()
                            .await?;
                        match serde_dynamo::from_item::<_, Limit>(
                            rec.items
                                .ok_or(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>)?
                                .pop()
                                .ok_or(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>)?,
                        ) {
                            Ok(l) => match l.last_idx
                            {
                                Some((f, u)) => send_queue(&sqs, &queue, Payload::Between(f, u)).await,
                                None => Err(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>),
                            },
                            Err(e) => Err(Box::new(MyError::NoLimit) as Box<dyn std::error::Error+Send+Sync>),
                        }
                    }
                }
            } else {
                Err(Box::new(MyError::UnsupportedEventType) as Box<dyn std::error::Error+Send+Sync>)
            }
        }
    }
}

#[test]
fn simple_test() -> Result<(), Box<dyn std::error::Error>> {
    println!("{:?}", serde_json::to_string(&Payload::Start(0, 999))?);
    println!(
        "{:?}",
        serde_json::to_string(&Payload::Between(1000, 1999))?
    );
    println!("{:?}", serde_json::to_string(&Payload::End)?);
    Ok(())
}
