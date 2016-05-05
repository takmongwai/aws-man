package main

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/service/iam"

	"e2u.io/aws-man/lib/aws"
	"e2u.io/aws-man/lib/ec2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var clc *cloudwatch.CloudWatch
var awsAccountId string
var AlarmActions string
var instances []*ec2man.Instance

const (
	AlarmSuffix = "-网络流出流量警报"
)

type MetricAlarm struct {
	*cloudwatch.MetricAlarm
}

// 取 alarm 中 Dimension Name:InstanceId 的值
func (m *MetricAlarm) InstanceId() string {
	for _, d := range m.Dimensions {
		if aws.StringValue(d.Name) == "InstanceId" {
			return aws.StringValue(d.Value)
		}
	}
	return ""
}

func init() {
	clc = cloudwatch.New(awsman.NewSession())
	i := iam.New(awsman.NewSession())
	ial, err := i.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		panic(err)
	}
	are := regexp.MustCompile(`iam::(\d+):user`)
	for _, ia := range ial.Users {
		uarn := aws.StringValue(ia.Arn)
		match := are.FindStringSubmatch(uarn)
		if len(match[1]) > 0 {
			awsAccountId = match[1]
			AlarmActions = "arn:aws-cn:sns:cn-north-1:" + awsAccountId + ":ec2-NetworkOut-Alarm"
			break
		}
	}

	//取所有的 ec2 实例
	filter := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}
	instances, err = ec2man.FilterInstances(filter)
	if err != nil {
		panic(err)
	}
}

// 加载所有的 alarm
func loadAllAlarm() []*MetricAlarm {
	var alarms []*MetricAlarm
	clc := cloudwatch.New(awsman.NewSession())
	clc.DescribeAlarmsPages(&cloudwatch.DescribeAlarmsInput{}, func(p *cloudwatch.DescribeAlarmsOutput, lastPage bool) (shouldContinue bool) {
		for _, a := range p.MetricAlarms {
			alarms = append(alarms, &MetricAlarm{a})
		}
		return lastPage
	})
	return alarms
}

func main() {
	alarms := loadAllAlarm()
	// 遍历所有实例,判断是否存在网络流量警报,如没有则创建一个,有的覆盖
	for ii, instance := range instances {
		log.Printf("[%03d] InstanceId: %s Name: %s\n", ii, aws.StringValue(instance.InstanceId), instance.Name())
		if alarmExists(instance, alarms) {
			log.Printf("alarm exists skip.")
			continue
		}
		if err := crateAlarm(instance); err != nil {
			log.Printf("ERROR %#v", err)
		} else {
			log.Println("Crate Alarm OK.")
		}
	}
}

func crateAlarm(instance *ec2man.Instance) error {
	_, err := clc.PutMetricAlarm(&cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(aws.StringValue(instance.InstanceId) + AlarmSuffix),
		ComparisonOperator: aws.String("GreaterThanOrEqualToThreshold"),
		EvaluationPeriods:  aws.Int64(4),
		MetricName:         aws.String("NetworkOut"),
		Namespace:          aws.String("AWS/EC2"),
		Period:             aws.Int64(300),
		Statistic:          aws.String("Average"),
		Threshold:          aws.Float64(20000000),
		ActionsEnabled:     aws.Bool(true),
		AlarmActions:       []*string{aws.String(AlarmActions)},
		AlarmDescription:   aws.String(instance.Name() + " - 网络流出流量警报"),
		Dimensions:         []*cloudwatch.Dimension{{Name: aws.String("InstanceId"), Value: instance.InstanceId}},
		// Unit:               aws.String("Bytes/Second"), // 设置 Unit 会造成 Insufficient Data
	})

	return err
}

// 查找指定实例是否存在流量流出警报
func alarmExists(i *ec2man.Instance, alarms []*MetricAlarm) bool {
	for _, alarm := range alarms {
		aname := aws.StringValue(alarm.AlarmName)
		iid := aws.StringValue(i.InstanceId)
		if iid == alarm.InstanceId() && aname == iid+AlarmSuffix {
			return true
		}
	}
	return false
}
