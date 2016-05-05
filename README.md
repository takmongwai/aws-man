# aws-man

aws 的自动化管理工具，并由一台 ec2 实例运行，负责管理所有账号下的 aws 资源

## ec2 snapshot

 ec2/ec2_snapshot 功能

一、
 
* 对所有正在运行且 Tag `tag:AutoSnapshot=true` (需对 ec2 手工打 Tag) 的 ec2 实例的所有卷创建快照,并删除1周前的快照,<font color="red">ec2 实例名属性不能重复</font> 既 `tag:Name=xxx` 不能有重复的,否则会因为 AMI 名重复而产生错误
* 如果是 ec2 root 卷的快照,则附加 Tag `tag:RootDeviceSnapshot=true` 
* 卷快照是增量进行，在保留卷和最后一个版本的快照的情况下就可以把快照转换成镜像启动实例。<font color="red">如果卷被删除,则其所有的快照也会被一同删除</font>


二、
 
为了防止卷失效导致有快照也不能恢复，需要定期将快照转换成 AMI 保存

* 将标记有 Tag `RootDeviceSnapshot=true` 的快照转换成 AMI，同时标记 Tag `tag:AutoAMI=true`

### 执行逻辑

#### AMIs 处理逻辑

取消 AMI 注册

1. 列出所有 Tag `tag:AutoAMI=true` and `is-public=false` and `state=available` 的 AMIs
2. 遍历 AMIs;找到一周前创建的 AMI
3. 判断 AMI 是否可以`取消注册`:
	1.	如果 AMI Tag `tag:BaseAMI=true` or Tag `tag:Autoscalling=true` 则不能`取消注册`
	2. 如果 AMI Tag `tag:AutoAMI=true` 则可以`取消注册`
4. 对可以`取消注册`的 AMI 取消注册





#### Instance -> Volumn -> Snapshot 处理逻辑

1. 列出 EC2-Instance 中 Tag `tag:AutoSnapshot=true`(<font color="red">手工设置</font>) 中的所有 Volumns
	1. 	遍历 Volumns;在 Volumn 中列出所有 Snapshots
	
		1. 遍历 Snapshots;找到一周前创建的 Snapshot
			1. 判断 Snapshot 是否可以 `删除`
				1. 如果 Snapshot Tag `tag:AutoSnapshot=true` 可以删除
			2. `删除`  符合条件的 Snapshot
			3. 完成 
		1. 为 Volumn 创建 Snapshot,打标记 Tag `tag:AutoSnapshot=true`,并判断是否是 RootDevice,如是则打 标记 Tag `tag:RootDeviceSnapshot=true`,只有 Tag `tag:RootDeviceSnapshot=true` 才会创建注册成 AMI
		
		1. 找到 Snapshots 中最新的一个 Snapshot(只有一个 Snapshot 则不会创建 AMI),创建 AMI,打标记为 Tag `tag:AutoAMI=true`,每周只创建一个 AMI,用 `ec2实例名-当前自然年周数` 命名

		
	2. 继续处理下一个 Volumn




## IAM 配置

IAM 策略

用户名 ec2_snapshot_runner

<b>Policy Name: auto-snapshot</b>

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1447935618000",
            "Effect": "Allow",
            "Action": [
                "ec2:CreateSnapshot",
                "ec2:CreateTags",
                "ec2:DeleteSnapshot",
                "ec2:DeregisterImage",
                "ec2:DescribeImages",
                "ec2:DescribeInstances",
                "ec2:DescribeSnapshots",
                "ec2:DescribeVolumes",
                "ec2:RegisterImage"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": "cloudwatch:*",
            "Resource": "*"
        },
        {
            "Effect":"Allow",
            "Action":"iam:ListUsers",
            "Resource":"*"
        }
    ]
}

## 部署和运行 

在某实例上运行该工具

* 编译 `go build ec2_snapshot.go`
* 上传到服务器上 `scp ec2_snapshot aws_tfa:/srv/scripts/aws-man/ec2/`
* 编写执行脚本 `<account_id>.sh` 具体内容见 [运行脚本](#运行脚本)



## 运行脚本
<a name="运行脚本"></a>

```bash

#!/bin/bash

export TZ='Asia/Shanghai'

export AWS_REGION=cn-north-1
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=
ACCOUNT_ID=<aws account>



DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
LOG_DIR=/srv/logs/aws-man

mkdir -p $LOG_DIR

$DIR/ec2_snapshot >> $LOG_DIR/ec2_snapshot_$ACCOUNT_ID.log 2>&1
$DIR/ec2_networkout_alerm >> $LOG_DIR/ec2_networkout_alerm_$ACCOUNT_ID.log 2>&1
```


每天运行一次,以 ec2-user 用户执行


<b>定时任务 crontab -e </b>

```
 0 4 * * *  /srv/scripts/aws-man/ec2/<account_id>.sh
```

<b>日志分割 /etc/logrotate.d</b>

```
/srv/logs/aws-man/ec2_*.log
{
    create 0644 ec2-user ec2-user
    copytruncate
    daily
    rotate 90
    missingok
    compress
    delaycompress
    sharedscripts
    endscript
}
```




## ec2 networkout alerm

ec2/ec2_networkout_alerm 功能

处理逻辑

* 遍历账号下所有的 ec2 实例，并为每个 ec2 实例创建一个以网络流出流量的 Alarm,如果 Alerm 已经存在，不会修改 Alerm 阈值
* 默认流出流量阈值: 20000000 Bytes/300 Sec,如果预设阈值不满足使用，可以在 `CloudWatch` 中针对每个 Alerm 做调整


依赖资源:

 * 建立 SNS Topics `ec2-NetworkOut-Alarm`
 * 向  SNS Topics 中添加订阅者接受 Alarm

 
 ## 部署和运行 

在某实例上运行该工具

* 编译 `go build ec2_networkout_alerm.go`
* 上传到服务器上 `scp ec2_networkout_alerm aws_tfa:/srv/scripts/aws-man/ec2/`
* 编写执行脚本 `<account_id>.sh` 具体内容见 [运行脚本](#运行脚本)


