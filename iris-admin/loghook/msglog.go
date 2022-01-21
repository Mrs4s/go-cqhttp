package loghook

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"sync"
)

var messages sync.Map

//对应uin下添加消息id，有序
func SaveMsg(uin int64, msgid int32) {
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

// 通过uin读取消息记录
func ReadMsg(uin int64) []db.StoredMessage {
	list := readmsg(uin)
	var msgList []db.StoredMessage
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

// 保存频道消息历史记录
func SaveGuildChannelMsg(guild, channelid uint64, msgid string) {
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

// 公共msgid读取频道 历史消息
func ReadGuildChannelMsg(guild, channelid uint64) []*db.StoredGuildChannelMessage {
	list := readGuildChannelMsg(guild, channelid)
	var msgList []*db.StoredGuildChannelMessage
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
