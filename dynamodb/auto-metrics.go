package main

const (
	Period = 3600
)

// 从 CloudWatch 中读取预设 metrics 和实际使用情况,对 capacity 进行自动调整
// http://docs.aws.amazon.com/amazondynamodb/latest/developerguide/MonitoringDynamoDB.html

func main() {
}

// aws dynamodb describe-table --table-name acms-oa2clients_dev 查看表格信息
// http://docs.aws.amazon.com/amazondynamodb/latest/developerguide/MonitoringDynamoDB.html#dynamodb-metrics
/*
通过命令行工具取数据

Sum 取已经使用的吞吐量,除 period(60) 得到每秒平均值,并以此作为预设吞吐量的调整依据


aws cloudwatch get-metric-statistics  \
    --namespace "AWS/DynamoDB" \
    --metric-name "ConsumedReadCapacityUnits" \
    --dimensions Name=TableName,Value="cmb-sign-service-pay-record-prod" \
    --statistics Sum \
    --period 60 \
    --start-time "2016-03-04T00:00:00Z" \
    --end-time "2016-03-04T23:59:59Z"

取 1 周，以1个小时为单位
aws cloudwatch get-metric-statistics  \
    --namespace "AWS/DynamoDB" \
    --metric-name "ConsumedReadCapacityUnits" \
    --dimensions Name=TableName,Value="cmb-sign-service-pay-record-prod" \
    --statistics Sum \
    --period 3600 \
    --start-time "2016-02-26T00:00:00Z" \
    --end-time "2016-03-04T23:59:59Z"


aws cloudwatch get-metric-statistics  \
    --namespace "AWS/DynamoDB" \
    --metric-name "ConsumedReadCapacityUnits" \
    --dimensions Name=TableName,Value="cmb-sign-service-pay-record-prod" Name=GlobalSecondaryIndexName,Value="appid-date-index" \
    --statistics Sum \
    --period 3600 \
    --start-time "2016-02-26T00:00:00Z" \
    --end-time "2016-03-04T23:59:59Z"


*/

/*
工具的执行顺序:
1. 得到所有的表格
2. 分别读取表格的预设吞吐量
3. 分别读取表格实际使用的吞吐量
4. 实际使用吞吐量和预设吞吐量进行比较,如果符合条件则调整,取一周的平均值做参照
*/
