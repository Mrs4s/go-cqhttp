package coolq

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/db"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// CQBot CQBot结构体,存储Bot实例相关配置
type CQBot struct {
	Client *client.QQClient

	lock   sync.RWMutex
	events []func(*Event)

	friendReqCache   sync.Map
	tempSessionCache sync.Map
}

// Event 事件
type Event struct {
	RawMsg global.MSG

	once   sync.Once
	buffer *bytes.Buffer
}

func (e *Event) marshal() {
	if e.buffer == nil {
		e.buffer = global.NewBuffer()
	}
	_ = json.NewEncoder(e.buffer).Encode(e.RawMsg)
}

// JSONBytes return byes of json by lazy marshalling.
func (e *Event) JSONBytes() []byte {
	e.once.Do(e.marshal)
	return e.buffer.Bytes()
}

// JSONString return string of json without extra allocation
// by lazy marshalling.
func (e *Event) JSONString() string {
	e.once.Do(e.marshal)
	return utils.B2S(e.buffer.Bytes())
}

// NewQQBot 初始化一个QQBot实例
func NewQQBot(cli *client.QQClient) *CQBot {
	bot := &CQBot{
		Client: cli,
	}
	bot.Client.OnPrivateMessage(bot.privateMessageEvent)
	bot.Client.OnGroupMessage(bot.groupMessageEvent)
	if base.ReportSelfMessage {
		bot.Client.OnSelfPrivateMessage(bot.privateMessageEvent)
		bot.Client.OnSelfGroupMessage(bot.groupMessageEvent)
	}
	bot.Client.OnTempMessage(bot.tempMessageEvent)
	bot.Client.GuildService.OnGuildChannelMessage(bot.guildChannelMessageEvent)
	bot.Client.GuildService.OnGuildMessageReactionsUpdated(bot.guildMessageReactionsUpdatedEvent)
	bot.Client.GuildService.OnGuildChannelUpdated(bot.guildChannelUpdatedEvent)
	bot.Client.GuildService.OnGuildChannelCreated(bot.guildChannelCreatedEvent)
	bot.Client.GuildService.OnGuildChannelDestroyed(bot.guildChannelDestroyedEvent)
	bot.Client.OnGroupMuted(bot.groupMutedEvent)
	bot.Client.OnGroupMessageRecalled(bot.groupRecallEvent)
	bot.Client.OnGroupNotify(bot.groupNotifyEvent)
	bot.Client.OnFriendNotify(bot.friendNotifyEvent)
	bot.Client.OnMemberSpecialTitleUpdated(bot.memberTitleUpdatedEvent)
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
		if base.HeartbeatInterval == 0 {
			log.Warn("警告: 心跳功能已关闭，若非预期，请检查配置文件。")
			return
		}
		t := time.NewTicker(base.HeartbeatInterval)
		for {
			<-t.C
			bot.dispatchEventMessage(global.MSG{
				"time":            time.Now().Unix(),
				"self_id":         bot.Client.Uin,
				"post_type":       "meta_event",
				"meta_event_type": "heartbeat",
				"status":          bot.CQGetStatus()["data"],
				"interval":        base.HeartbeatInterval.Milliseconds(),
			})
		}
	}()
	return bot
}

// OnEventPush 注册事件上报函数
func (bot *CQBot) OnEventPush(f func(e *Event)) {
	bot.lock.Lock()
	bot.events = append(bot.events, f)
	bot.lock.Unlock()
}

// UploadLocalImageAsGroup 上传本地图片至群聊
func (bot *CQBot) UploadLocalImageAsGroup(groupCode int64, img *LocalImageElement) (i *message.GroupImageElement, err error) {
	if img.File != "" {
		f, err := os.Open(img.File)
		if err != nil {
			return nil, errors.Wrap(err, "open image error")
		}
		defer func() { _ = f.Close() }()
		img.Stream = f
	}
	if lawful, mime := base.IsLawfulImage(img.Stream); !lawful {
		return nil, errors.New("image type error: " + mime)
	}
	i, err = bot.Client.UploadGroupImage(groupCode, img.Stream)
	if i != nil {
		i.Flash = img.Flash
		i.EffectID = img.EffectID
	}
	return
}

// UploadLocalVideo 上传本地短视频至群聊
func (bot *CQBot) UploadLocalVideo(target int64, v *LocalVideoElement) (*message.ShortVideoElement, error) {
	video, err := os.Open(v.File)
	if err != nil {
		return nil, err
	}
	defer func() { _ = video.Close() }()
	hash, _ := utils.ComputeMd5AndLength(io.MultiReader(video, v.thumb))
	cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash)+".cache")
	_, _ = video.Seek(0, io.SeekStart)
	_, _ = v.thumb.Seek(0, io.SeekStart)
	return bot.Client.UploadGroupShortVideo(target, video, v.thumb, cacheFile)
}

// UploadLocalImageAsPrivate 上传本地图片至私聊
func (bot *CQBot) UploadLocalImageAsPrivate(userID int64, img *LocalImageElement) (i *message.FriendImageElement, err error) {
	if img.File != "" {
		f, err := os.Open(img.File)
		if err != nil {
			return nil, errors.Wrap(err, "open image error")
		}
		defer func() { _ = f.Close() }()
		img.Stream = f
	}
	if lawful, mime := base.IsLawfulImage(img.Stream); !lawful {
		return nil, errors.New("image type error: " + mime)
	}
	i, err = bot.Client.UploadPrivateImage(userID, img.Stream)
	if i != nil {
		i.Flash = img.Flash
	}
	return
}

// UploadLocalImageAsGuildChannel 上传本地图片至频道
func (bot *CQBot) UploadLocalImageAsGuildChannel(guildID, channelID uint64, img *LocalImageElement) (*message.GuildImageElement, error) {
	if img.File != "" {
		f, err := os.Open(img.File)
		if err != nil {
			return nil, errors.Wrap(err, "open image error")
		}
		defer func() { _ = f.Close() }()
		img.Stream = f
	}
	if lawful, mime := base.IsLawfulImage(img.Stream); !lawful {
		return nil, errors.New("image type error: " + mime)
	}
	return bot.Client.GuildService.UploadGuildImage(guildID, channelID, img.Stream)
}

// SendGroupMessage 发送群消息
func (bot *CQBot) SendGroupMessage(groupID int64, m *message.SendingMessage) int32 {
	newElem := make([]message.IMessageElement, 0, len(m.Elements))
	group := bot.Client.FindGroup(groupID)
	for _, e := range m.Elements {
		switch i := e.(type) {
		case *LocalImageElement, *message.VoiceElement, *LocalVideoElement:
			i, err := bot.uploadMedia(i, groupID, true)
			if err != nil {
				log.Warnf("警告: 群 %d 消息%s上传失败: %v", groupID, e.Type().String(), err)
				continue
			}
			e = i
		case *PokeElement:
			if group != nil {
				if mem := group.FindMember(i.Target); mem != nil {
					mem.Poke()
				}
			}
			return 0
		case *message.MusicShareElement:
			ret, err := bot.Client.SendGroupMusicShare(groupID, i)
			if err != nil {
				log.Warnf("警告: 群 %v 富文本消息发送失败: %v", groupID, err)
				return -1
			}
			return bot.InsertGroupMessage(ret)
		case *message.AtElement:
			if i.Target == 0 && group.SelfPermission() == client.Member {
				e = message.NewText("@全体成员")
			}
		}
		newElem = append(newElem, e)
	}
	if len(newElem) == 0 {
		log.Warnf("群消息发送失败: 消息为空.")
		return -1
	}
	m.Elements = newElem
	bot.checkMedia(newElem, groupID)
	ret := bot.Client.SendGroupMessage(groupID, m, base.ForceFragmented)
	if ret == nil || ret.Id == -1 {
		log.Warnf("群消息发送失败: 账号可能被风控.")
		return -1
	}
	return bot.InsertGroupMessage(ret)
}

// SendPrivateMessage 发送私聊消息
func (bot *CQBot) SendPrivateMessage(target int64, groupID int64, m *message.SendingMessage) int32 {
	newElem := make([]message.IMessageElement, 0, len(m.Elements))
	for _, e := range m.Elements {
		switch i := e.(type) {
		case *LocalImageElement, *message.VoiceElement, *LocalVideoElement:
			i, err := bot.uploadMedia(i, target, false)
			if err != nil {
				log.Warnf("警告: 私聊 %d 消息%s上传失败: %v", target, e.Type().String(), err)
				continue
			}
			e = i
		case *PokeElement:
			bot.Client.SendFriendPoke(i.Target)
			return 0
		case *message.MusicShareElement:
			bot.Client.SendFriendMusicShare(target, i)
			return 0
		}
		newElem = append(newElem, e)
	}
	if len(newElem) == 0 {
		log.Warnf("好友消息发送失败: 消息为空.")
		return -1
	}
	m.Elements = newElem
	bot.checkMedia(newElem, bot.Client.Uin)

	// 单向好友是否存在
	unidirectionalFriendExists := func() bool {
		list, err := bot.Client.GetUnidirectionalFriendList()
		if err != nil {
			return false
		}
		for _, f := range list {
			if f.Uin == target {
				return true
			}
		}
		return false
	}

	session, ok := bot.tempSessionCache.Load(target)
	var id int32 = -1

	switch {
	case bot.Client.FindFriend(target) != nil: // 双向好友
		msg := bot.Client.SendPrivateMessage(target, m)
		if msg != nil {
			id = bot.InsertPrivateMessage(msg)
		}
	case ok || groupID != 0: // 临时会话
		switch {
		case groupID != 0 && bot.Client.FindGroup(groupID) == nil:
			log.Errorf("错误: 找不到群(%v)", groupID)
		case groupID != 0 && !bot.Client.FindGroup(groupID).AdministratorOrOwner():
			log.Errorf("错误: 机器人在群(%v) 为非管理员或群主, 无法主动发起临时会话", groupID)
		case groupID != 0 && bot.Client.FindGroup(groupID).FindMember(target) == nil:
			log.Errorf("错误: 群员(%v) 不在 群(%v), 无法发起临时会话", target, groupID)
		default:
			if session == nil && groupID != 0 {
				msg := bot.Client.SendGroupTempMessage(groupID, target, m)
				if msg != nil { // nolint
					// todo(Mrs4s)
					// id = bot.InsertTempMessage(target, msg)
				}
				break
			}
			msg, err := session.(*client.TempSessionInfo).SendMessage(m)
			if err != nil {
				log.Errorf("发送临时会话消息失败: %v", err)
				break
			}
			if msg != nil { // nolint
				// todo(Mrs4s)
				// id = bot.InsertTempMessage(target, msg)
			}
		}
	case unidirectionalFriendExists(): // 单向好友
		msg := bot.Client.SendPrivateMessage(target, m)
		if msg != nil {
			id = bot.InsertPrivateMessage(msg)
		}
	default:
		nickname := "Unknown"
		if summaryInfo, _ := bot.Client.GetSummaryInfo(target); summaryInfo != nil {
			nickname = summaryInfo.Nickname
		}
		log.Errorf("错误: 请先添加 %v(%v) 为好友", nickname, target)
	}
	return id
}

// SendGuildChannelMessage 发送频道消息
func (bot *CQBot) SendGuildChannelMessage(guildID, channelID uint64, m *message.SendingMessage) string {
	newElem := make([]message.IMessageElement, 0, len(m.Elements))
	for _, e := range m.Elements {
		switch i := e.(type) {
		case *LocalImageElement:
			n, err := bot.UploadLocalImageAsGuildChannel(guildID, channelID, i)
			if err != nil {
				log.Warnf("警告: 频道 %d 消息%s上传失败: %v", channelID, e.Type().String(), err)
				continue
			}
			e = n
		case *LocalVideoElement, *LocalVoiceElement, *PokeElement, *message.MusicShareElement:
			log.Warnf("警告: 频道暂不支持发送 %v 消息", i.Type().String())
			continue
		}
		newElem = append(newElem, e)
	}
	if len(newElem) == 0 {
		log.Warnf("频道消息发送失败: 消息为空.")
		return ""
	}
	m.Elements = newElem
	bot.checkMedia(newElem, bot.Client.Uin)
	ret, err := bot.Client.GuildService.SendGuildChannelMessage(guildID, channelID, m)
	if err != nil {
		log.Warnf("频道消息发送失败: %v", err)
		return ""
	}
	// todo: insert db
	return fmt.Sprintf("%v-%v", ret.Id, ret.InternalId)
}

// InsertGroupMessage 群聊消息入数据库
func (bot *CQBot) InsertGroupMessage(m *message.GroupMessage) int32 {
	t := &message.SendingMessage{Elements: m.Elements}
	replyElem := t.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	msg := &db.StoredGroupMessage{
		ID:       encodeMessageID(m.GroupCode, m.Id),
		GlobalID: db.ToGlobalID(m.GroupCode, m.Id),
		SubType:  "normal",
		Attribute: &db.StoredMessageAttribute{
			MessageSeq: m.Id,
			InternalID: m.InternalId,
			SenderUin:  m.Sender.Uin,
			SenderName: m.Sender.DisplayName(),
			Timestamp:  int64(m.Time),
		},
		GroupCode: m.GroupCode,
		AnonymousID: func() string {
			if m.Sender.IsAnonymous() {
				return m.Sender.AnonymousInfo.AnonymousId
			}
			return ""
		}(),
		Content: ToMessageContent(m.Elements),
	}
	if replyElem != nil {
		reply := replyElem.(*message.ReplyElement)
		msg.SubType = "quote"
		msg.QuotedInfo = &db.QuotedInfo{
			PrevID:        encodeMessageID(m.GroupCode, reply.ReplySeq),
			PrevGlobalID:  db.ToGlobalID(m.GroupCode, reply.ReplySeq),
			QuotedContent: ToMessageContent(reply.Elements),
		}
	}
	if err := db.InsertGroupMessage(msg); err != nil {
		log.Warnf("记录聊天数据时出现错误: %v", err)
		return -1
	}
	return msg.GlobalID
}

// InsertPrivateMessage 私聊消息入数据库
func (bot *CQBot) InsertPrivateMessage(m *message.PrivateMessage) int32 {
	t := &message.SendingMessage{Elements: m.Elements}
	replyElem := t.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	msg := &db.StoredPrivateMessage{
		ID:       encodeMessageID(m.Sender.Uin, m.Id),
		GlobalID: db.ToGlobalID(m.Sender.Uin, m.Id),
		SubType:  "normal",
		Attribute: &db.StoredMessageAttribute{
			MessageSeq: m.Id,
			InternalID: m.InternalId,
			SenderUin:  m.Sender.Uin,
			SenderName: m.Sender.DisplayName(),
			Timestamp:  int64(m.Time),
		},
		SessionUin: func() int64 {
			if m.Sender.Uin == m.Self {
				return m.Target
			}
			return m.Sender.Uin
		}(),
		TargetUin: m.Target,
		Content:   ToMessageContent(m.Elements),
	}
	if replyElem != nil {
		reply := replyElem.(*message.ReplyElement)
		msg.SubType = "quote"
		msg.QuotedInfo = &db.QuotedInfo{
			PrevID:        encodeMessageID(reply.Sender, reply.ReplySeq),
			PrevGlobalID:  db.ToGlobalID(reply.Sender, reply.ReplySeq),
			QuotedContent: ToMessageContent(reply.Elements),
		}
	}
	if err := db.InsertPrivateMessage(msg); err != nil {
		log.Warnf("记录聊天数据时出现错误: %v", err)
		return -1
	}
	return msg.GlobalID
}

/*
// InsertTempMessage 临时消息入数据库
func (bot *CQBot) InsertTempMessage(target int64, m *message.TempMessage) int32 {
	val := global.MSG{
		"message-id": m.Id,
		// FIXME(InsertTempMessage) InternalId missing
		"from-group": m.GroupCode,
		"group-name": m.GroupName,
		"target":     target,
		"sender":     m.Sender,
		"time":       int32(time.Now().Unix()),
		"message":    ToStringMessage(m.Elements, 0, true),
	}
	id := db.ToGlobalID(m.Sender.Uin, m.Id)
	if bot.db != nil {
		buf := global.NewBuffer()
		defer global.PutBuffer(buf)
		if err := gob.NewEncoder(buf).Encode(val); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
		if err := bot.db.Put(binary.ToBytes(id), buf.Bytes(), nil); err != nil {
			log.Warnf("记录聊天数据时出现错误: %v", err)
			return -1
		}
	}
	return id
}
*/

// Release 释放Bot实例
func (bot *CQBot) Release() {

}

func (bot *CQBot) dispatchEventMessage(m global.MSG) {
	bot.lock.RLock()
	defer bot.lock.RUnlock()

	event := &Event{RawMsg: m}
	wg := sync.WaitGroup{}
	wg.Add(len(bot.events))
	for _, f := range bot.events {
		go func(fn func(*Event)) {
			defer func() {
				wg.Done()
				if pan := recover(); pan != nil {
					log.Warnf("处理事件 %v 时出现错误: %v \n%s", m, pan, debug.Stack())
				}
			}()

			start := time.Now()
			fn(event)
			end := time.Now()
			if end.Sub(start) > time.Second*5 {
				log.Debugf("警告: 事件处理耗时超过 5 秒 (%v), 请检查应用是否有堵塞.", end.Sub(start))
			}
		}(f)
	}
	wg.Wait()
	global.PutBuffer(event.buffer)
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

func (bot *CQBot) uploadMedia(raw message.IMessageElement, target int64, group bool) (message.IMessageElement, error) {
	switch m := raw.(type) {
	case *LocalImageElement:
		if group {
			return bot.UploadLocalImageAsGroup(target, m)
		}
		return bot.UploadLocalImageAsPrivate(target, m)
	case *message.VoiceElement:
		if group {
			return bot.Client.UploadGroupPtt(target, bytes.NewReader(m.Data))
		}
		return bot.Client.UploadPrivatePtt(target, bytes.NewReader(m.Data))
	case *LocalVideoElement:
		return bot.UploadLocalVideo(target, m)
	}
	return nil, errors.New("unsupported message element type")
}

// encodeMessageID 临时先这样, 暂时用不上
func encodeMessageID(target int64, seq int32) string {
	return hex.EncodeToString(binary.NewWriterF(func(w *binary.Writer) {
		w.WriteUInt64(uint64(target))
		w.WriteUInt32(uint32(seq))
	}))
}
