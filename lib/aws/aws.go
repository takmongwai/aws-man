package awsman

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	Day             = time.Hour * 24
	Week            = Day * 7
	TimeLayout      = "2006-01-02 15:04:05"
	ImageTimeLayout = "20060102150405"
	DevMode         = false // 开发模式不会真正地建立修改资源
	HttpDebug       = false
)

// NewSession
func NewSession() *session.Session {
	// https://github.com/aws/aws-sdk-go/issues/430
	config := defaults.Config().WithRegion("cn-north-1").WithCredentials(nil)
	if HttpDebug {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}
	return session.New(config)
}

// WeekDate 获取当前时间之前一周的时间
func WeekDate() (time.Time, time.Time) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)
	return startTime, endTime
}
