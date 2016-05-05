package dynamodbman

import (
	"log"
	"time"

	"e2u.io/aws-man/lib/aws"
	"e2u.io/aws-man/lib/cloudwatch"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var (
	meDynamoDB *dynamodb.DynamoDB
)

func init() {
	log.Println("init...")
	meDynamoDB = dynamodb.New(awsman.NewSession())
}

// Table 表格属性
type Table struct {
	TableName              string
	ProvisionedThroughput  *dynamodb.ProvisionedThroughputDescription
	GlobalSecondaryIndexes []*dynamodb.GlobalSecondaryIndexDescription
	LocalSecondaryIndexes  []*dynamodb.LocalSecondaryIndexDescription
	// *dynamodb.ConsumedCapacity
}

//	ListTables 列出所有的表格,同时整合所需信息
//  减少预设吞吐量每天不得大于4次
func ListTables() ([]*Table, error) {
	var rs []*Table
	listFn := func(p *dynamodb.ListTablesOutput, lastPage bool) (shouldContinue bool) {
		for _, tn := range p.TableNames {
			desc, err := meDynamoDB.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: tn,
			})
			if err != nil {
				log.Fatal(err)
				continue
			}

			rs = append(rs, &Table{
				TableName:              aws.StringValue(tn),               // 表名
				ProvisionedThroughput:  desc.Table.ProvisionedThroughput,  // 取吞吐量预设值,当天下调次数
				GlobalSecondaryIndexes: desc.Table.GlobalSecondaryIndexes, //
				// 需要从 CloudWatch 中获取实际使用的吞吐量
			})
		}
		return true
	}

	if err := meDynamoDB.ListTablesPages(&dynamodb.ListTablesInput{Limit: aws.Int64(100)}, listFn); err != nil {
		return nil, err
	}

	return rs, nil
}

// TableConsumedReadCaptcity 从 cloudwatch 中取 表 一定时间内实际使用的 读 吞吐量的最大值
func TableConsumedReadCaptcity(tableName string) (float64, error) {
	return consumedCaptcityMax(tableName, "ConsumedReadCapacityUnits", "")
}

// TableConsumedReadCaptcity 从 cloudwatch 中取 表 一定时间内实际使用的 写 吞吐量的最大值
func TableConsumedWriteCaptcity(tableName string) (float64, error) {
	return consumedCaptcityMax(tableName, "ConsumedWriteCapacityUnits", "")
}

// TableConsumedReadCaptcity 从 cloudwatch 中取 索引 一定时间内实际使用的 读 吞吐量的最大值
func IndexConsumedReadCaptcity(tableName, indexName string) (float64, error) {
	return consumedCaptcityMax(tableName, "ConsumedReadCapacityUnits", indexName)
}

// TableConsumedReadCaptcity 从 cloudwatch 中取 索引 一定时间内实际使用的 写 吞吐量的最大值
func IndexConsumedWriteCaptcity(tableName, indexName string) (float64, error) {
	return consumedCaptcityMax(tableName, "ConsumedWriteCapacityUnits", indexName)
}

// 取一定时间里间隔 period 秒 的实际使用吞吐量的最大值
// 取 10 分钟内最大的吞吐量
func consumedCaptcityMax(tableName string, metricName string, indexName string) (float64, error) {
	period := 60
	e := time.Now()
	s := e.Add(-(time.Minute * 10))
	dimensions := []*cloudwatch.Dimension{
		{
			Name:  aws.String("TableName"),
			Value: aws.String(tableName),
		},
	}

	if len(indexName) != 0 {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  aws.String("GlobalSecondaryIndexName"),
			Value: aws.String(indexName),
		})
	}

	in := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/DynamoDB"),
		MetricName: aws.String(metricName),
		StartTime:  aws.Time(s),
		EndTime:    aws.Time(e),
		Dimensions: dimensions,
		Period:     aws.Int64(int64(period)),
		Statistics: []*string{aws.String("Sum")},
	}
	o, err := cloudwatchman.GetMetricStatistics(in)
	if err != nil {
		return 0.0, err
	}

	var max float64
	for _, do := range o.Datapoints {
		if aws.Float64Value(do.Sum) > max {
			max = aws.Float64Value(do.Sum)
		}
	}
	return max / float64(period), nil
}
