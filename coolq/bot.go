package coolq

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/pkg/errors"
	"github.com/segmentio/asm/base64"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/db"
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
	nextTokenCache   *utils.Cache
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
		Client:         cli,
		nextTokenCache: utils.NewCache(time.Second * 10),
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
	bot.Client.GuildService.OnGuildMessageRecalled(bot.guildChannelMessageRecalledEvent)
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

type worker struct {
	wg sync.WaitGroup
}

func (w *worker) do(f func()) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		f()
	}()
}

func (w *worker) wait() {
	w.wg.Wait()
}

// uploadLocalImage 上传本地图片
func (bot *CQBot) uploadLocalImage(target message.Source, img *LocalImageElement) (message.IMessageElement, error) {
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
	i, err := bot.Client.UploadImage(target, img.Stream, 4)
	if err != nil {
		return nil, err
	}
	switch i := i.(type) {
	case *message.GroupImageElement:
		i.Flash = img.Flash
		i.EffectID = img.EffectID
	case *message.FriendImageElement:
		i.Flash = img.Flash
	}
	return i, err
}

// uploadLocalVideo 上传本地短视频至群聊
func (bot *CQBot) uploadLocalVideo(target message.Source, v *LocalVideoElement) (*message.ShortVideoElement, error) {
	video, err := os.Open(v.File)
	if err != nil {
		return nil, err
	}
	defer func() { _ = video.Close() }()
	return bot.Client.UploadShortVideo(target, video, v.thumb, 4)
}

func removeLocalElement(elements []message.IMessageElement) []message.IMessageElement {
	var j int
	for i, e := range elements {
		switch e.(type) {
		case *LocalImageElement, *LocalVideoElement:
		case *message.VoiceElement: // 未上传的语音消息， 也删除
		case nil:
		default:
			if j < i {
				elements[j] = e
			}
			j++
		}
	}
	return elements[:j]
}

const uploadFailedTemplate = "警告: %s %d %s上传失败: %v"

func (bot *CQBot) uploadMedia(target message.Source, elements []message.IMessageElement) []message.IMessageElement {
	var w worker
	var source string
	switch target.SourceType { // nolint:exhaustive
	case message.SourceGroup:
		source = "群"
	case message.SourcePrivate:
		source = "私聊"
	case message.SourceGuildChannel:
		source = "频道"
	}

	for i, m := range elements {
		p := &elements[i]
		switch e := m.(type) {
		case *LocalImageElement:
			w.do(func() {
				m, err := bot.uploadLocalImage(target, e)
				if err != nil {
					log.Warnf(uploadFailedTemplate, source, target.PrimaryID, "图片", err)
				} else {
					*p = m
				}
			})
		case *message.VoiceElement:
			w.do(func() {
				m, err := bot.Client.UploadVoice(target, bytes.NewReader(e.Data))
				if err != nil {
					log.Warnf(uploadFailedTemplate, source, target.PrimaryID, "语音", err)
				} else {
					*p = m
				}
			})
		case *LocalVideoElement:
			w.do(func() {
				m, err := bot.uploadLocalVideo(target, e)
				if err != nil {
					log.Warnf(uploadFailedTemplate, source, target.PrimaryID, "视频", err)
				} else {
					*p = m
				}
			})
		}
	}
	w.wait()
	return removeLocalElement(elements)
}

// SendGroupMessage 发送群消息
func (bot *CQBot) SendGroupMessage(groupID int64, m *message.SendingMessage) int32 {
	newElem := make([]message.IMessageElement, 0, len(m.Elements))
	group := bot.Client.FindGroup(groupID)
	source := message.Source{
		SourceType: message.SourceGroup,
		PrimaryID:  groupID,
	}
	m.Elements = bot.uploadMedia(source, m.Elements)
	for _, e := range m.Elements {
		switch i := e.(type) {
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
	source := message.Source{
		SourceType: message.SourcePrivate,
		PrimaryID:  target,
	}
	m.Elements = bot.uploadMedia(source, m.Elements)
	for _, e := range m.Elements {
		switch i := e.(type) {
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
		if !base.AllowTempSession {
			log.Warnf("发送临时会话消息失败: 已关闭临时会话信息发送功能")
			return -1
		}
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
	source := message.Source{
		SourceType:  message.SourceGuildChannel,
		PrimaryID:   int64(guildID),
		SecondaryID: int64(channelID),
	}
	m.Elements = bot.uploadMedia(source, m.Elements)
	for _, e := range m.Elements {
		switch i := e.(type) {
		case *message.MusicShareElement:
			bot.Client.SendGuildMusicShare(guildID, channelID, i)
			return "-1" // todo: fix this

		case *message.VoiceElement, *PokeElement:
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

// InsertGuildChannelMessage 频道消息入数据库
func (bot *CQBot) InsertGuildChannelMessage(m *message.GuildChannelMessage) string {
	id := encodeGuildMessageID(m.GuildId, m.ChannelId, m.Id, message.SourceGuildChannel)
	msg := &db.StoredGuildChannelMessage{
		ID: id,
		Attribute: &db.StoredGuildMessageAttribute{
			MessageSeq:   m.Id,
			InternalID:   m.InternalId,
			SenderTinyID: m.Sender.TinyId,
			SenderName:   m.Sender.Nickname,
			Timestamp:    m.Time,
		},
		GuildID:   m.GuildId,
		ChannelID: m.ChannelId,
		Content:   ToMessageContent(m.Elements),
	}
	if err := db.InsertGuildChannelMessage(msg); err != nil {
		log.Warnf("记录聊天数据时出现错误: %v", err)
		return ""
	}
	return msg.ID
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
	if event.buffer != nil {
		global.PutBuffer(event.buffer)
	}
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

// encodeMessageID 临时先这样, 暂时用不上
func encodeMessageID(target int64, seq int32) string {
	return hex.EncodeToString(binary.NewWriterF(func(w *binary.Writer) {
		w.WriteUInt64(uint64(target))
		w.WriteUInt32(uint32(seq))
	}))
}

// encodeGuildMessageID 将频道信息编码为字符串
// 当信息来源为 Channel 时 primaryID 为 guildID , subID 为 channelID
// 当信息来源为 Direct 时 primaryID 为 guildID , subID 为 tinyID
func encodeGuildMessageID(primaryID, subID, seq uint64, source message.SourceType) string {
	return base64.StdEncoding.EncodeToString(binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(byte(source))
		w.WriteUInt64(primaryID)
		w.WriteUInt64(subID)
		w.WriteUInt64(seq)
	}))
}

func decodeGuildMessageID(id string) (source *message.Source, seq uint64) {
	b, _ := base64.StdEncoding.DecodeString(id)
	if len(b) < 25 {
		return
	}
	r := binary.NewReader(b)
	source = &message.Source{
		SourceType:  message.SourceType(r.ReadByte()),
		PrimaryID:   r.ReadInt64(),
		SecondaryID: r.ReadInt64(),
	}
	seq = uint64(r.ReadInt64())
	return
}
