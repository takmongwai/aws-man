package cloudwatchman

import (
	"log"

	"e2u.io/aws-man/lib/aws"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

var (
	meCloudWatch *cloudwatch.CloudWatch
)

func init() {
	log.Println("init...")
	meCloudWatch = cloudwatch.New(awsman.NewSession())
}

func GetMetricStatistics(in *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
	return meCloudWatch.GetMetricStatistics(in)
}
