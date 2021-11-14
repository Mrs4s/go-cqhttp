// Package selfdiagnosis 自我诊断相关
package selfdiagnosis

import (
	"github.com/Mrs4s/MiraiGo/client"
	log "github.com/sirupsen/logrus"
)

// NetworkDiagnosis 诊断网络状态并输出结果
func NetworkDiagnosis(c *client.QQClient) {
	log.Infof("开始诊断网络情况")
	qualityInfo := c.ConnectionQualityTest()
	log.Debugf("聊天服务器连接延迟: %vms", qualityInfo.ChatServerLatency)
	log.Debugf("聊天服务器丢包率: %v%%", qualityInfo.ChatServerPacketLoss*10)
	log.Debugf("长消息服务器连接延迟: %vms", qualityInfo.LongMessageServerLatency)
	log.Debugf("长消息服务器响应延迟: %vms", qualityInfo.LongMessageServerResponseLatency)
	log.Debugf("媒体服务器连接延迟: %vms", qualityInfo.SrvServerLatency)
	log.Debugf("媒体服务器丢包率: %v%%", qualityInfo.SrvServerPacketLoss*10)

	const (
		chatServerErrorMessage        = "可能出现消息丢失/延迟或频繁掉线等情况, 请检查本地网络状态."
		longMessageServerErrorMessage = "可能导致无法接收/发送长消息的情况, 请检查本地网络状态."
		mediaServerErrorMessage       = "可能导致无法上传/下载媒体文件, 无法上传群共享, 无法发送消息等情况, 请检查本地网络状态."
	)

	if qualityInfo.ChatServerLatency > 1000 {
		if qualityInfo.ChatServerLatency == 9999 {
			log.Errorf("错误: 聊天服务器延迟测试失败, %v", chatServerErrorMessage)
		} else {
			log.Warnf("警告: 聊天服务器延迟为 %vms，大于 1000ms, %v", qualityInfo.ChatServerLatency, chatServerErrorMessage)
		}
	}

	if qualityInfo.ChatServerPacketLoss > 0 {
		log.Warnf("警告: 本地连接聊天服务器丢包率为 %v%%, %v", qualityInfo.ChatServerPacketLoss*10, chatServerErrorMessage)
	}

	if qualityInfo.LongMessageServerLatency > 1000 {
		if qualityInfo.LongMessageServerLatency == 9999 {
			log.Errorf("错误: 长消息服务器延迟测试失败, %v 如果您使用的腾讯云服务器, 请修改DNS到114.114.114.114", longMessageServerErrorMessage)
		} else {
			log.Warnf("警告: 长消息延迟为 %vms, 大于 1000ms, %v", qualityInfo.LongMessageServerLatency, longMessageServerErrorMessage)
		}
	}

	if qualityInfo.LongMessageServerResponseLatency > 2000 {
		if qualityInfo.LongMessageServerResponseLatency == 9999 {
			log.Errorf("错误: 长消息服务器响应延迟测试失败, %v 如果您使用的腾讯云服务器, 请修改DNS到114.114.114.114", longMessageServerErrorMessage)
		} else {
			log.Warnf("警告: 长消息响应延迟为 %vms, 大于 1000ms, %v", qualityInfo.LongMessageServerResponseLatency, longMessageServerErrorMessage)
		}
	}

	if qualityInfo.SrvServerLatency > 1000 {
		if qualityInfo.SrvServerPacketLoss == 9999 {
			log.Errorf("错误: 媒体服务器延迟测试失败, %v", mediaServerErrorMessage)
		} else {
			log.Warnf("警告: 媒体服务器延迟为 %vms，大于 1000ms, %v", qualityInfo.SrvServerLatency, mediaServerErrorMessage)
		}
	}

	if qualityInfo.SrvServerPacketLoss > 0 {
		log.Warnf("警告: 本地连接媒体服务器丢包率为 %v%%, %v", qualityInfo.SrvServerPacketLoss*10, mediaServerErrorMessage)
	}

	if qualityInfo.ChatServerLatency > 1000 || qualityInfo.ChatServerPacketLoss > 0 || qualityInfo.LongMessageServerLatency > 1000 || qualityInfo.SrvServerLatency > 1000 || qualityInfo.SrvServerPacketLoss > 0 {
		log.Infof("网络诊断完成. 发现问题, 请检查日志.")
	} else {
		log.Infof("网络诊断完成. 未发现问题")
	}
}

// DNSDiagnosis 诊断DNS状态并输出结果
func DNSDiagnosis() {
	// todo
}

// EnvironmentDiagnosis 诊断本地环境状态并输出结果
func EnvironmentDiagnosis() {
	// todo
}
