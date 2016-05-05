package ec2man

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"e2u.io/aws-man/lib/aws"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	mec2 *ec2.EC2
)

func init() {
	log.Println("init...")
	mec2 = ec2.New(awsman.NewSession())
}

const (
	TagKeyRootDeviceSnapshot = "RootDeviceSnapshot"
	TagValueTrue             = "true"
	TagKeyName               = "Name"
	TagKeyAutoSnapshot       = "AutoSnapshot"
	TagKeyAutoAMI            = "AutoAMI"
	TagKeyBaseAMI            = "BaseAMI"
	TagKeyAutoscalling       = "AutoScallingAMI"
)

type Snapshot struct {
	*ec2.Snapshot
}

type Volume struct {
	*ec2.Volume
}

type Instance struct {
	*ec2.Instance
}

type Image struct {
	*ec2.Image
}

// SnapshotInstances 列出所有标记为 Tag:AutoSnapshot=true 和 running 的实例
func SnapshotInstances() ([]*Instance, error) {
	filter := []*ec2.Filter{
		{
			Name:   aws.String("tag:" + TagKeyAutoSnapshot),
			Values: []*string{aws.String("true")},
		},
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}

	return FilterInstances(filter)
}

// FilterInstances 列出符合指定过滤器的实例列表
func FilterInstances(filter []*ec2.Filter) ([]*Instance, error) {
	var rs []*Instance
	drs, err := mec2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filter,
	})
	if err != nil {
		return nil, err
	}
	for _, irs := range drs.Reservations {
		for _, is := range irs.Instances {
			rs = append(rs, &Instance{is})
		}
	}
	return rs, nil
}

// Images 列出所有标记为 Tag:AutoAMI=true 且  available,同时 Tag 不能是 BaseAMI 和 AutoScalling 的镜像
func Images() ([]*Image, error) {
	var rs []*Image

	irs, err := mec2.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + TagKeyAutoAMI),
				Values: []*string{aws.String(TagValueTrue)},
			},
			{
				Name:   aws.String("is-public"),
				Values: []*string{aws.String("false")},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, image := range irs.Images {
		rs = append(rs, &Image{image})
	}
	return rs, nil
}

// Deregister 取消注册一个镜像
func (i *Image) Deregister() error {

	if !i.CanDeregister() {
		return errors.New("AMI not allow delete.")
	}

	if awsman.DevMode {
		log.Println("[DevMode] AMI Deregister done")
		return nil
	}

	_, err := mec2.DeregisterImage(&ec2.DeregisterImageInput{
		ImageId: i.ImageId,
	})
	return err

}

// CanDeregister 判断是否可以取消注册一个镜像,AutoAMI 可以取消,如果有 BaseAMI,Autoscalling Tag 则不能取消注册
func (i *Image) CanDeregister() bool {

	for _, tag := range i.Tags {
		if (aws.StringValue(tag.Key) == TagKeyBaseAMI && aws.StringValue(tag.Value) == TagValueTrue) ||
			(aws.StringValue(tag.Key) == TagKeyAutoscalling && aws.StringValue(tag.Value) == TagValueTrue) {
			return false
		}
	}

	for _, tag := range i.Tags {
		if aws.StringValue(tag.Key) == TagKeyAutoAMI &&
			aws.StringValue(tag.Value) == TagValueTrue {
			return true
		}
	}

	return false
}

// Name  取  Tags -> Name 的值
func (i *Instance) Name() string {
	for _, t := range i.Tags {
		if aws.StringValue(t.Key) == "Name" {
			return aws.StringValue(t.Value)
		}
	}
	return "NO NAME " + aws.StringValue(i.InstanceId)
}

// Volumes 取实例的所有卷
func (i *Instance) Volumes() ([]*Volume, error) {
	var rs []*Volume
	vrs, err := mec2.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.instance-id"),
				Values: []*string{i.InstanceId},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, ds := range vrs.Volumes {
		rs = append(rs, &Volume{ds})
	}
	return rs, nil
}

// Snapshots 取卷的快照,只取 completed 和标记有 tag:AutoSnapshot=true 的快照列表
func (v *Volume) Snapshots() ([]*Snapshot, error) {
	var rs []*Snapshot

	srs, err := mec2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + TagKeyAutoSnapshot),
				Values: []*string{aws.String(TagValueTrue)},
			},
			{
				Name:   aws.String("volume-id"),
				Values: []*string{v.VolumeId},
			},
			{
				Name:   aws.String("status"),
				Values: []*string{aws.String("completed")},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, ss := range srs.Snapshots {
		rs = append(rs, &Snapshot{ss})
	}
	return rs, nil
}

// CreateSnapshot 创建快照,并为新快照打标记
func (v *Volume) CreateSnapshot(instanceName string, tags []*ec2.Tag) (*ec2.Snapshot, error) {
	if awsman.DevMode {
		log.Println("[DevMode] Create Snapshot done")
		return nil, nil
	}
	desc := aws.String(fmt.Sprintf("Snapshot %s %s", instanceName, time.Now().Format(awsman.TimeLayout)))
	snapshot, err := mec2.CreateSnapshot(&ec2.CreateSnapshotInput{
		Description: desc,
		VolumeId:    v.VolumeId,
	})
	if err != nil {
		return nil, err
	}
	_, err = mec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{snapshot.SnapshotId},
		Tags:      tags,
	})
	if err != nil {
		return nil, err
	}
	return snapshot, nil

}

// IsRootDevice 判断当前卷是否在实例中是根设备
func (v *Volume) IsRootDevice(i *Instance) bool {
	for _, bd := range i.BlockDeviceMappings {
		if aws.StringValue(bd.DeviceName) == aws.StringValue(i.RootDeviceName) &&
			aws.StringValue(bd.Ebs.VolumeId) == aws.StringValue(v.VolumeId) {
			return true
		}
	}
	return false
}

// Delete  删除指定的快照,只能删除带有 Tag:AutoSnapshot=true 的快照
func (s *Snapshot) Delete() error {

	if !s.CanDelete() {
		return errors.New("Snapshot not allow delete.")
	}

	if awsman.DevMode {
		log.Println("[DevMode] Delete Snapshot done")
		return nil
	}

	_, err := mec2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: s.SnapshotId,
	})
	return err
}

// CanDelete 判断镜像是否可以删除
func (s *Snapshot) CanDelete() bool {
	for _, tag := range s.Tags {
		if aws.StringValue(tag.Key) == TagKeyAutoSnapshot && aws.StringValue(tag.Value) == TagValueTrue {
			return true
		}
	}
	return false
}

// CreateAMI 从快照创建一个 AMI,AutoSnapshot 才能创建
func (s *Snapshot) RegisterImage(instanceName string) error {
	if awsman.DevMode {
		log.Println("[DevMode] Create AMI done")
		return nil
	}
	desc := aws.String(fmt.Sprintf("Image %s %s", instanceName, time.Now().Format(awsman.TimeLayout)))
	// Image 名字不能相同 ec2实例名-当前时间
	year, week := time.Now().ISOWeek()
	name := aws.String(fmt.Sprintf("%s-%d-%d", instanceName, year, week))
	ri, err := mec2.RegisterImage(&ec2.RegisterImageInput{
		Architecture: aws.String("x86_64"),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          s.SnapshotId,
					VolumeSize:          s.VolumeSize,
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		Description:        desc,
		Name:               name,
		RootDeviceName:     aws.String("/dev/xvda"),
		SriovNetSupport:    aws.String("simple"),
		VirtualizationType: aws.String("hvm"),
	})

	if err != nil {
		return err
	}

	tags := []*ec2.Tag{
		{
			Key:   aws.String(TagKeyAutoAMI),
			Value: aws.String("true"),
		},
	}

	_, err = mec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{ri.ImageId},
		Tags:      tags,
	})

	return err
}

// 取当前卷快照集合中最新的一个,且状态是可用的镜像
func (v *Volume) LatestSnapshot() (*Snapshot, error) {
	ss, err := v.Snapshots()
	if err != nil {
		return nil, err
	}
	if len(ss) <= 0 {
		return nil, errors.New("No Latest Snapshot")
	}
	sort.Sort(SnapshotsSortByCreateTime{ss})
	return ss[0], nil
}

// 对 Snapshots 进行排序

type SortSnapshots []*Snapshot

func (ss SortSnapshots) Len() int { return len(ss) }

func (ss SortSnapshots) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }

type SnapshotsSortByCreateTime struct{ SortSnapshots }

func (sc SnapshotsSortByCreateTime) Less(i, j int) bool {
	return (sc.SortSnapshots[i].StartTime.Sub(*sc.SortSnapshots[j].StartTime)) > 0
}

func EC2Matedata() *ec2metadata.EC2Metadata {
	return ec2metadata.New(awsman.NewSession())
}
