// Package loghook 日志 hook 包
package loghook

import (
	"fmt"
	"sync"

	"github.com/Mrs4s/go-cqhttp/internal/base"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/db"
)

var messages sync.Map

// SaveMsg 对应uin下添加消息id，有序
func SaveMsg(uin int64, msgid int32) {
	if !base.WebUI {
		return
	}
	list := readmsg(uin)
	list.Add(msgid)
	savemsg(uin, list)
}

func readmsg(uin int64) common.FixedList {
	var (
		list common.FixedList
	)
	listO, ok := messages.Load(uin)
	if !ok {
		list = common.NewFixedList(20)
	} else {
		list = listO.(common.FixedList)
	}
	return list
}

func savemsg(uin int64, msg common.FixedList) {
	messages.Store(uin, msg)
}

// ReadMsg 通过uin读取消息记录
func ReadMsg(uin int64) []db.StoredMessage {
	list := readmsg(uin)
	msgList := make([]db.StoredMessage, 0)
	for _, v := range list.Data() {
		msgid := v.(int32)
		stdMsg, err := db.GetMessageByGlobalID(msgid)
		if err != nil {
			continue
		}
		msgList = append(msgList, stdMsg)
	}
	return msgList
}

// SaveGuildChannelMsg 保存频道消息历史记录
func SaveGuildChannelMsg(guild, channelid uint64, msgid string) {
	if !base.WebUI {
		return
	}
	list := readGuildChannelMsg(guild, channelid)
	list.Add(msgid)
	saveGuildChannelMsg(guild, channelid, list)
}

func readGuildChannelMsg(guild, channelid uint64) common.FixedList {
	var (
		list common.FixedList
	)
	listO, ok := messages.Load(fmt.Sprintf("%d-%d", guild, channelid))
	if !ok {
		list = common.NewFixedList(20)
	} else {
		list = listO.(common.FixedList)
	}
	return list
}
func saveGuildChannelMsg(guild, channelid uint64, list common.FixedList) {
	messages.Store(fmt.Sprintf("%d-%d", guild, channelid), list)
}

// ReadGuildChannelMsg 公共msgid读取频道 历史消息
func ReadGuildChannelMsg(guild, channelid uint64) []*db.StoredGuildChannelMessage {
	list := readGuildChannelMsg(guild, channelid)
	msgList := make([]*db.StoredGuildChannelMessage, 0)
	for _, v := range list.Data() {
		msgid, ok := v.(string)
		if !ok {
			continue
		}
		stdMsg, err := db.GetGuildChannelMessageByID(msgid)
		if err != nil {
			continue
		}
		msgList = append(msgList, stdMsg)
	}
	return msgList
}
