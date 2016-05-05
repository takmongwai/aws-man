package main

import (
	"fmt"
	"time"

	"e2u.io/aws-man/lib/aws"
	"e2u.io/aws-man/lib/ec2"

	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	var err error
	var images []*ec2man.Image
	var instances []*ec2man.Instance

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		images, err = ec2man.Images()
		if err != nil {
			panic(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		instances, err = ec2man.SnapshotInstances()
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()

	// 处理 AMI

	for ii, image := range images {
		log.Printf("[%03d] ImageId: %s AMIName: %s\n", ii, aws.StringValue(image.ImageId), aws.StringValue(image.Name))
		createAt, err := time.Parse("2006-01-02T15:04:05.000Z", aws.StringValue(image.CreationDate))
		if err != nil {
			log.Println("parse image create date err: ", err)
		}
		if time.Now().Sub(createAt) > awsman.Week {
			log.Println("删除", aws.StringValue(image.ImageId))
			if err := image.Deregister(); err != nil {
				log.Println("Deregister AMI error,", err)
			}
		}
		fmt.Println("-----------------------------------------------------")
	}

	// 处理快照
	for ii, instance := range instances {
		log.Printf("[%03d] InstanceId: %s Name: %s\n", ii, aws.StringValue(instance.InstanceId), instance.Name())
		volumes, err := instance.Volumes()
		if err != nil {
			panic(err)
		}
		for vi, volume := range volumes {
			log.Printf("\t[%03d] VolumeId: %s isRootDevice: %v\n", vi, aws.StringValue(volume.VolumeId), volume.IsRootDevice(instance))
			snapshots, err := volume.Snapshots()
			if err != nil {
				panic(err)
			}

			if latestSnapshot, err := volume.LatestSnapshot(); err != nil {
				log.Println("get Latest Snapshot error: ", err)
			} else if latestSnapshot != nil {
				log.Println("Latest Snapshot: ", *latestSnapshot.StartTime)
				if err := latestSnapshot.RegisterImage(instance.Name()); err != nil {
					log.Println("Register Image error: ", err)
				}
			}

			for si, snapshot := range snapshots {
				log.Printf("\t\t[%03d] SnapshotId: %s CreatedAt: %s\n", si, aws.StringValue(snapshot.SnapshotId), snapshot.StartTime)
				// 取消一周前注册的 AMI,取消注册的 AMI 需带有 tag AutoAMI=true 的标记,同时明确跳过  BaseAMI,AutoScalling 的 AMI
				// 删除一周之前创建的快照
				if time.Now().Sub(*snapshot.StartTime) > awsman.Week {
					if err := snapshot.Delete(); err != nil {
						log.Println("Delete Snapshot error: ", err)
					}
				}
			}
			// 创建一个快照,并打标记
			tags := []*ec2.Tag{
				{
					Key:   aws.String(ec2man.TagKeyName), // 快照 Name
					Value: aws.String(instance.Name()),
				},
				{
					Key:   aws.String(ec2man.TagKeyAutoSnapshot),
					Value: aws.String("true"),
				},
			}
			// 如果是根设备,还要多加一个标记,以便生成 AMI
			if volume.IsRootDevice(instance) {
				tags = append(tags, &ec2.Tag{Key: aws.String(ec2man.TagKeyRootDeviceSnapshot), Value: aws.String(ec2man.TagValueTrue)})
			}

			if _, err := volume.CreateSnapshot(instance.Name(), tags); err != nil {
				log.Println("Create Snapshot error: ", err)
			}
		}
		fmt.Println("-----------------------------------------------------")
	}
}

// aws ec2 describe-images --filters "Name=is-public,Values=false"
