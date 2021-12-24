package gocq

import (
	"os"

	"github.com/Mrs4s/MiraiGo/client"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
)

// LoadDevice 加载设备信息
func LoadDevice() {
	if !global.PathExists("device.json") {
		log.Warn("虚拟设备信息不存在, 将自动生成随机设备.")
		client.GenRandomDevice()
		_ = os.WriteFile("device.json", client.SystemDeviceInfo.ToJson(), 0o644)
		log.Info("已生成设备信息并保存到 device.json 文件.")
	} else {
		log.Info("将使用 device.json 内的设备信息运行Bot.")
		if err := client.SystemDeviceInfo.ReadJson([]byte(global.ReadAllText("device.json"))); err != nil {
			log.Fatalf("加载设备信息失败: %v", err)
		}
	}
}
