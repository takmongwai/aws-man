package disk

import (
	"os/exec"
	"strings"
	"syscall"
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

type DiskStatus struct {
	InstanceID string `json:"instance_id"`
	Path       string `json:"file_system"`
	All        uint64 `json:"all"`
	Used       uint64 `json:"used"`
	Free       uint64 `json:"free"`
}

func (d DiskStatus) AllGB() float64 {
	return float64(d.All) / float64(GB)
}

func (d DiskStatus) UsedGB() float64 {
	return float64(d.Used) / float64(GB)
}

func (d DiskStatus) FreeGB() float64 {
	return float64(d.Free) / float64(GB)
}

func (d DiskStatus) UsedPercent() float64 {
	return float64(float64(d.Used)/float64(d.All)) * 100
}

func (d DiskStatus) FreePercent() float64 {
	return float64(float64(d.Free)/float64(d.All)) * 100
}

// 返回磁盘使用信息
func NewDiskUsage(path string) (*DiskStatus, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return nil, err
	}
	disk := &DiskStatus{
		Path: path,
		All:  fs.Blocks * uint64(fs.Bsize),
		Free: fs.Bfree * uint64(fs.Bsize),
	}
	disk.Used = disk.All - disk.Free
	return disk, nil
}

// 列出系统中所有已经加载的文件系统
func ListMountedFileSystems() ([]string, error) {
	var rs []string
	cmd := `df -l | grep "^/dev/" `
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return rs, err
	}
	for _, o := range strings.Split(string(out), "\n") {
		o := strings.TrimSpace(o)
		if o == "" {
			continue
		}
		ms := strings.Split(o, " ")
		if len(ms) == 0 {
			continue
		}
		rs = append(rs, ms[len(ms)-1])
	}
	return rs, nil
}
