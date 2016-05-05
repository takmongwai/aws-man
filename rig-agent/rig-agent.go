// 运行在 ec2 实例上的代理程序,获取实例的信息并发往 CloudWatch
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/robfig/cron"
	"github.com/takama/daemon"

	"net"
	"net/http"
	_ "net/http/pprof"

	"e2u.io/aws-man/lib/aws"
	"e2u.io/aws-man/lib/ec2"
	"e2u.io/aws-man/rig-agent/disk"
)

var clc *cloudwatch.CloudWatch

const (
	name        = "rig-agent"
	description = "ec2 instance status agent"
	version     = "0.0.1"
)

var stdlog, errlog *log.Logger

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
	clc = cloudwatch.New(awsman.NewSession())
}

type Service struct {
	daemon.Daemon
}

func (service *Service) Manage() (string, error) {
	usage := "Usage: " + name + " install | remove | start | stop | status"
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}
	return "run", nil
}

// SendDiskUsage 向 CloudWatch 发送磁盘使情况
func SendDiskUsage() {
	fs, err := disk.ListMountedFileSystems()
	if err != nil {
		errlog.Println(err)
	}

	for _, f := range fs {
		d, err := disk.NewDiskUsage(f)
		if err != nil || d == nil {
			errlog.Printf(`path "%s" error %s,skip`, f, err.Error())
			continue
		}
		em2 := ec2man.EC2Matedata()
		if em2.Available() {
			d.InstanceID, _ = em2.GetMetadata("instance-id")
		} else {

		}
		stdlog.Printf("All: %.2f GB\n", d.AllGB())
		stdlog.Printf("Used: %.2f GB,%.2f%%\n", d.UsedGB(), d.UsedPercent())
		stdlog.Printf("Free: %.2f GB,%.2f%%\n", d.FreeGB(), d.FreePercent())
	}
}

func (service *Service) Run() (string, error) {
	c := cron.New()
	c.AddFunc("@every 5s", SendDiskUsage)
	c.Start()

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		errlog.Println(err)
	}
	stdlog.Println(ln.Addr().String())
	s := &http.Server{}
	s.Serve(ln)

	return "runing...", nil
}

func main() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		<-interrupt
		os.Exit(1)
	}()

	srv, err := daemon.New(name, description)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}

	log.Println(status)
	if status == "run" {
		service.Run()
	}

}
