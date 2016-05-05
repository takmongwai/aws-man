package main

import (
	"fmt"
	"testing"

	"e2u.io/aws-man/lib/aws"
	"e2u.io/aws-man/lib/dynamodb"
)

func TestWeekDate(t *testing.T) {
	t.SkipNow()
	fmt.Println(awsman.WeekDate())
}

func TestListTables(t *testing.T) {
	t.SkipNow()
	fmt.Println(dynamodbman.ListTables())
}

func TestTableConsumedReadCaptcityAverage(t *testing.T) {
	tableName := "cmb-sign-service-pay-record-prod"
	indexName := "appid-date-index"
	fmt.Print("consumed table read: ")
	fmt.Println(dynamodbman.TableConsumedReadCaptcity(tableName))
	fmt.Print("consumed table write: ")
	fmt.Println(dynamodbman.TableConsumedWriteCaptcity(tableName))
	fmt.Print("consumed index read: ")
	fmt.Println(dynamodbman.IndexConsumedReadCaptcity(tableName, indexName))
	fmt.Print("consumed index write: ")
	fmt.Println(dynamodbman.IndexConsumedWriteCaptcity(tableName, indexName))
}
