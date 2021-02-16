package coolq

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type CQBot struct {
	Client *client.QQClient

	events         []func(MSG)
	db             *leveldb.DB
	friendReqCache sync.Map
	tempMsgCache   sync.Map
	oneWayMsgCache sync.Map
}

type MSG map[string]interface{}

var ForceFragmented = false

func NewQQBot(cli *client.QQClient, conf *global.JSONConfig) *CQBot {
	bot := &CQBot{
		Client: cli,
	}
	if conf.EnableDB {
		p := path.Join("data", "leveldb")
		db, err := leveldb.OpenFile(p, nil)
		if err != nil {
			log.Fatalf("打开数据库失败, 如果频繁遇到此问题请清理 data/leveldb 文件夹或关闭数据库功能。")
		}
		bot.db = db
		gob.Register(message.Sender{})
		log.Info("信息数据库初始化完成.")
	} else {
		log.Warn("警告: 信息数据库已关闭，将无法使用 [回复/撤回] 等功能。")
	}
	bot.Client.OnPrivateMessage(bot.privateMessageEvent)
	bot.Client.OnGroupMessage(bot.groupMessageEvent)
	bot.Client.OnTempMessage(bot.tempMessageEvent)
	bot.Client.OnGroupMuted(bot.groupMutedEvent)
	bot.Client.OnGroupMessageRecalled(bot.groupRecallEvent)
	bot.Client.OnGroupNotify(bot.groupNotifyEvent)
	bot.Client.OnFriendNotify(bot.friendNotifyEvent)
	bot.Client.OnFriendMessageRecalled(bot.friendRecallEvent)
	bot.Client.OnReceivedOfflineFile(bot.offlineFileEvent)
	bot.Client.OnJoinGroup(bot.joinGroupEvent)
	bot.Client.OnLeaveGroup(bot.leaveGroupEvent)
	bot.Client.OnGroupMemberJoined(bot.memberJoinEvent)
	bot.Client.OnGroupMemberLeaved(bot.memberLeaveEvent)
	bot.Client.OnGroupMemberPermissionChanged(bot.memberPermissionChangedEvent)
	bot.Client.OnGroupMemberCardUpdated(bot.memberCardUpdatedEvent)
	bot.Client.OnNewFriendRequest(bot.friendRequestEvent)
	bot.Client.OnNewFriendAdded(bot.friendAddedEvent)
	bot.Client.OnGroupInvited(bot.groupInvitedEvent)
	bot.Client.OnUserWantJoinGroup(bot.groupJoinReqEvent)
	bot.Client.OnOtherClientStatusChanged(bot.otherClientStatusChangedEvent)
	bot.Client.OnGroupDigest(bot.groupEssenceMsg)
	go func() {
		i := conf.HeartbeatInterval
		if i < 0 {
			log.Warn("警告: 心跳功能已关闭，若非预期，请检查配置文件。")
			return
		}
		if i == 0 {
			i = 5
		}
		for {
			time.Sleep(time.Second * i)
			bot.dispatchEventMessage(MSG{
				"time":            time.Now().Unix(),
				"self_id":         bot.Client.Uin,
				"post_type":       "meta_event",
				"meta_event_type": "heartbeat",
				"status":          bot.CQGetStatus()["data"],
				"interval":        1000 * i,
			})
		}
	}()
	return bot
}

func (bot *CQBot) OnEventPush(f func(m MSG)) {
	bot.events = append(bot.events, f)
}

func (bot *CQBot) GetMessage(mid int32) MSG {
	if bot.db != nil {
		m := MSG{}
		data, err := bot.db.Get(binary.ToBytes(mid), nil)
		if err == nil {
			buff := new(bytes.Buffer)
			buff.Write(binary.GZipUncompress(data))
			err = gob.NewDecoder(buff).Decode(&m)
			if err == nil {
				return m
			}
		}
		log.Warnf("获取信息时出现错误: %v id: %v", err, mid)
	}
	return nil
}

func (bot *CQBot) UploadLocalImageAsGroup(groupCode int64, img *LocalImageElement) (*message.GroupImageElement, error) {
	if img.Stream != nil {
		return bot.Client.UploadGroupImage(groupCode, img.Stream)
	}
	return bot.Client.UploadGroupImageByFile(groupCode, img.File)
}

func (bot *CQBot) UploadLocalVideo(target int64, v *LocalVideoElement) (*message.ShortVideoElement, error) {
	if v.File != "" {
		video, err := os.Open(v.File)
		if err != nil {
			return nil, err
		}
		defer video.Close()
		hash, _ := utils.ComputeMd5AndLength(io.MultiReader(video, v.thumb))
		cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash[:])+".cache")
		_, _ = video.Seek(0, io.SeekStart)
		_, _ = v.thumb.Seek(0, io.SeekStart)
		return bot.Client.UploadGroupShortVideo(target, video, v.thumb, cacheFile)
	}
	return &v.ShortVideoElement, nil
}

func (bot *CQBot) UploadLocalImageAsPrivate(userId int64, img *LocalImageElement) (*message.FriendImageElement, error) {
	if img.Stream != nil {
		return bot.Client.UploadPrivateImage(userId, img.Stream)
	}
	// need update.
	f, err := os.Open(img.File)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return bot.Client.UploadPrivateImage(userId, f)
}

func (bot *CQBot) SendGroupMessage(groupId int64, m *message.SendingMessage) int32 {
	var newElem []message.IMessageElement
	for _, elem := range m.Elements {
		if i, ok := elem.(*LocalImageElement); ok {
			gm, err := bot.UploadLocalImageAsGroup(groupId, i)
			if err != nil {
				log.Warnf("警告: 群 %v 消息图片上传失败: %v", groupId, err)
				continue
			}
			newElem = append(newElem, gm)
			continue
		}
		if i, ok := elem.(*message.VoiceElement); ok {
			gv, err := bot.Client.UploadGroupPtt(groupId, bytes.NewReader(i.Data))
			if err != nil {
				log.Warnf("警告: 群 %v 消息语音上传失败: %v", groupId, err)
				continue
			}
			newElem = append(newElem, gv)
			continue
		}
		if i, ok := elem.(*LocalVideoElement); ok {
			gv, err := bot.UploadLocalVideo(groupId, i)
			if err != nil {
				log.Warnf("警告: 群 %v 消息短视频上传失败: %v", groupId, err)
				continue
			}
			newElem = append(newElem, gv)
			continue
		}
		if i, ok := elem.(*PokeElement); ok {
			if group := bot.Client.FindGroup(groupId); group != nil {
				if mem := group.FindMember(i.Target); mem != nil {
					mem.Poke()
					return 0
				}
			}
		}
		if i, ok := elem.(*GiftElement); ok {
			bot.Client.SendGroupGift(uint64(groupId), uint64(i.Target), i.GiftId)
			return 0
		}
		if i, ok := elem.(*message.MusicShareElement); ok {
			ret, err := bot.Client.SendGroupMusicShare(groupId, i)
			if err != nil {
				log.Warnf("警告: 群 %v 富文本消息发送失败: %v", groupId, err)
				return -1
			}
			return bot.InsertGroupMessage(ret)
		}
		newElem = append(newElem, elem)
	}
	if len(newElem) == 0 {
		log.Warnf("群消息发送失败: 消息为空.")
		return -1
	}
	m.Elements = newElem
	bot.checkMedia(newElem)
	ret := bot.Client.SendGroupMessage(groupId, m, ForceFragmented)
	if ret == nil || ret.Id == -1 {
		log.Warnf("群消息发送失败: 账号可能被风控.")
		return -1
	}
	return bot.InsertGroupMessage(ret)
}

func (bot *CQBot) SendPrivateMessage(target int64, m *message.SendingMessage) int32 {
	var newElem []message.IMessageElement
	for _, elem := range m.Elements {
		if i, ok := elem.(*LocalImageElement); ok {
			fm, err := bot.UploadLocalImageAsPrivate(target, i)
			if err != nil {
				log.Warnf("警告: 私聊 %v 消息图片上传失败.", target)
				continue
			}
			newElem = append(newElem, fm)
			continue
		}
		if i, ok := elem.(*PokeElement); ok {
			bot.Client.SendFriendPoke(i.Target)
			return 0
		}
		if i, ok := elem.(*message.VoiceElement); ok {
			fv, err := bot.Client.UploadPrivatePtt(target, i.Data)
			if err != nil {
				log.Warnf("警告: 私聊 %v 消息语音上传失败: %v", target, err)
				continue
			}
			newElem = append(newElem, fv)
			continue
		}
		if i, ok := elem.(*LocalVideoElement); ok {
			gv, err := bot.UploadLocalVideo(target, i)
			if err != nil {
				log.Warnf("警告: 私聊 %v 消息短视频上传失败: %v", target, err)
				continue
			}
			newElem = append(newElem, gv)
			continue
		}
		if i, ok := elem.(*message.MusicShareElement); ok {
			bot.Client.SendFriendMusicShare(target, i)
			return 0
		}
		newElem = append(newElem, elem)
	}
	if len(newElem) == 0 {
		log.Warnf("好友消息发送失败: 消息为空.")
		return -1
	}
	m.Elements = newElem
	bot.checkMedia(newElem)
	var id int32 = -1
	if bot.Client.FindFriend(target) != nil { // 双向好友
		msg := bot.Client.SendPrivateMessage(target, m)
		if msg != nil {
			id = bot.InsertPrivateMessage(msg)
		}
	} else if code, ok := bot.tempMsgCache.Load(target); ok { // 临时会话
		msg := bot.Client.SendTempMessage(code.(int64), target, m)
		if msg != nil {
			id = msg.Id
		}
	} else if _, ok := bot.oneWayMsgCache.Load(target); ok { // 单向好友
		msg := bot.Client.SendPrivateMessage(target, m)
		if msg != nil {
			id = bot.InsertPrivateMessage(msg)
		}
	}
	if id == -1 {
		return -1
	}
	return id
}

func (bot *CQBot) InsertGroupMessage(m *message.GroupMessage) int32 {
	val := MSG{
		"message-id":  m.Id,
		"internal-id": m.InternalId,
		"group":       m.GroupCode,
		"group-name":  m.GroupName,
		"sender":      m.Sender,
		"time":        m.Time,
		"message":     ToStringMessage(m.Elements, m.GroupCode, true),
	}
	id := ToGlobalId(m.GroupCode, m.Id)
	if bot.db != nil {
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(val); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
		if err := bot.db.Put(binary.ToBytes(id), binary.GZipCompress(buf.Bytes()), nil); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
	}
	return id
}

func (bot *CQBot) InsertPrivateMessage(m *message.PrivateMessage) int32 {
	val := MSG{
		"message-id":  m.Id,
		"internal-id": m.InternalId,
		"target":      m.Target,
		"sender":      m.Sender,
		"time":        m.Time,
		"message":     ToStringMessage(m.Elements, m.Sender.Uin, true),
	}
	id := ToGlobalId(m.Sender.Uin, m.Id)
	if bot.db != nil {
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(val); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
		if err := bot.db.Put(binary.ToBytes(id), binary.GZipCompress(buf.Bytes()), nil); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
	}
	return id
}

func ToGlobalId(code int64, msgId int32) int32 {
	return int32(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d-%d", code, msgId))))
}

func (bot *CQBot) Release() {
	if bot.db != nil {
		_ = bot.db.Close()
	}
}

func (bot *CQBot) dispatchEventMessage(m MSG) {
	if global.EventFilter != nil && !global.EventFilter.Eval(global.MSG(m)) {
		log.Debug("Event filtered!")
		return
	}
	for _, f := range bot.events {
		go func(fn func(MSG)) {
			defer func() {
				if pan := recover(); pan != nil {
					log.Warnf("处理事件 %v 时出现错误: %v \n%s", m, pan, debug.Stack())
				}
			}()
			start := time.Now()
			fn(m)
			end := time.Now()
			if end.Sub(start) > time.Second*5 {
				log.Debugf("警告: 事件处理耗时超过 5 秒 (%v), 请检查应用是否有堵塞.", end.Sub(start))
			}
		}(f)
	}
}

func (bot *CQBot) formatGroupMessage(m *message.GroupMessage) MSG {
	cqm := ToStringMessage(m.Elements, m.GroupCode, true)
	gm := MSG{
		"anonymous":    nil,
		"font":         0,
		"group_id":     m.GroupCode,
		"message":      ToFormattedMessage(m.Elements, m.GroupCode, false),
		"message_type": "group",
		"message_seq":  m.Id,
		"post_type":    "message",
		"raw_message":  cqm,
		"self_id":      bot.Client.Uin,
		"sender": MSG{
			"age":     0,
			"area":    "",
			"level":   "",
			"sex":     "unknown",
			"user_id": m.Sender.Uin,
		},
		"sub_type": "normal",
		"time":     time.Now().Unix(),
		"user_id":  m.Sender.Uin,
	}
	if m.Sender.IsAnonymous() {
		gm["anonymous"] = MSG{
			"flag": m.Sender.AnonymousInfo.AnonymousId + "|" + m.Sender.AnonymousInfo.AnonymousNick,
			"id":   m.Sender.Uin,
			"name": m.Sender.AnonymousInfo.AnonymousNick,
		}
		gm["sender"].(MSG)["nickname"] = "匿名消息"
		gm["sub_type"] = "anonymous"
	} else {
		mem := bot.Client.FindGroup(m.GroupCode).FindMember(m.Sender.Uin)
		ms := gm["sender"].(MSG)
		ms["role"] = func() string {
			switch mem.Permission {
			case client.Owner:
				return "owner"
			case client.Administrator:
				return "admin"
			default:
				return "member"
			}
		}()
		ms["nickname"] = mem.Nickname
		ms["card"] = mem.CardName
		ms["title"] = mem.SpecialTitle
	}
	return gm
}

func formatGroupName(group *client.GroupInfo) string {
	return fmt.Sprintf("%s(%d)", group.Name, group.Code)
}

func formatMemberName(mem *client.GroupMemberInfo) string {
	if mem == nil {
		return "未知"
	}
	return fmt.Sprintf("%s(%d)", mem.DisplayName(), mem.Uin)
}

func (m MSG) ToJson() string {
	b, _ := json.Marshal(m)
	return string(b)
}
