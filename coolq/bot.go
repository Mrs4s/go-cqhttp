package coolq

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"
)

// CQBot CQBot结构体,存储Bot实例相关配置
type CQBot struct {
	Client *client.QQClient

	lock   sync.RWMutex
	events []func(*Event)

	db               *leveldb.DB
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

// ForceFragmented 是否启用强制分片
var ForceFragmented = false

// SkipMimeScan 是否跳过Mime扫描
var SkipMimeScan bool

// keep sync with /docs/file.md#MINE
var lawfulImageTypes = [...]string{
	"image/bmp",
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

var lawfulAudioTypes = [...]string{
	"audio/aac",
	"audio/aiff",
	"audio/amr",
	"audio/ape",
	"audio/flac",
	"audio/midi",
	"audio/mp4",
	"audio/mpeg",
	"audio/ogg",
	"audio/wav",
	"audio/x-m4a",
}

// NewQQBot 初始化一个QQBot实例
func NewQQBot(cli *client.QQClient, conf *config.Config) *CQBot {
	bot := &CQBot{
		Client: cli,
	}
	enableLevelDB := false
	node, ok := conf.Database["leveldb"]
	if ok {
		lconf := new(config.LevelDBConfig)
		_ = node.Decode(lconf)
		enableLevelDB = lconf.Enable
	}
	if enableLevelDB {
		p := path.Join("data", "leveldb")
		db, err := leveldb.OpenFile(p, &opt.Options{
			WriteBuffer: 128 * opt.KiB,
		})
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
	if conf.Message.ReportSelfMessage {
		bot.Client.OnSelfPrivateMessage(bot.privateMessageEvent)
		bot.Client.OnSelfGroupMessage(bot.groupMessageEvent)
	}
	bot.Client.OnTempMessage(bot.tempMessageEvent)
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
		i := conf.Heartbeat.Interval
		if i < 0 || conf.Heartbeat.Disabled {
			log.Warn("警告: 心跳功能已关闭，若非预期，请检查配置文件。")
			return
		}
		if i == 0 {
			i = 5
		}
		t := time.NewTicker(time.Second * time.Duration(i))
		for {
			<-t.C
			bot.dispatchEventMessage(global.MSG{
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

// OnEventPush 注册事件上报函数
func (bot *CQBot) OnEventPush(f func(e *Event)) {
	bot.lock.Lock()
	bot.events = append(bot.events, f)
	bot.lock.Unlock()
}

// GetMessage 获取给定消息id对应的消息
func (bot *CQBot) GetMessage(mid int32) global.MSG {
	if bot.db != nil {
		m := global.MSG{}
		data, err := bot.db.Get(binary.ToBytes(mid), nil)
		if err == nil {
			err = gob.NewDecoder(bytes.NewReader(data)).Decode(&m)
			if err == nil {
				return m
			}
		}
		log.Warnf("获取信息时出现错误: %v id: %v", err, mid)
	}
	return nil
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
	if lawful, mime := IsLawfulImage(img.Stream); !lawful {
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
	defer video.Close()
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
	if lawful, mime := IsLawfulImage(img.Stream); !lawful {
		return nil, errors.New("image type error: " + mime)
	}
	i, err = bot.Client.UploadPrivateImage(userID, img.Stream)
	if i != nil {
		i.Flash = img.Flash
	}
	return
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
		case *GiftElement:
			bot.Client.SendGroupGift(uint64(groupID), uint64(i.Target), i.GiftID)
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
	bot.checkMedia(newElem)
	ret := bot.Client.SendGroupMessage(groupID, m, ForceFragmented)
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
	bot.checkMedia(newElem)

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
				if msg != nil {
					id = bot.InsertTempMessage(target, msg)
				}
				break
			}
			msg, err := session.(*client.TempSessionInfo).SendMessage(m)
			if err != nil {
				log.Errorf("发送临时会话消息失败: %v", err)
				break
			}
			if msg != nil {
				id = bot.InsertTempMessage(target, msg)
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

// InsertGroupMessage 群聊消息入数据库
func (bot *CQBot) InsertGroupMessage(m *message.GroupMessage) int32 {
	val := global.MSG{
		"message-id":  m.Id,
		"internal-id": m.InternalId,
		"group":       m.GroupCode,
		"group-name":  m.GroupName,
		"sender":      m.Sender,
		"time":        m.Time,
		"message":     ToStringMessage(m.Elements, m.GroupCode, true),
	}
	id := toGlobalID(m.GroupCode, m.Id)
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

// InsertPrivateMessage 私聊消息入数据库
func (bot *CQBot) InsertPrivateMessage(m *message.PrivateMessage) int32 {
	val := global.MSG{
		"message-id":  m.Id,
		"internal-id": m.InternalId,
		"target":      m.Target,
		"sender":      m.Sender,
		"time":        m.Time,
		"message":     ToStringMessage(m.Elements, 0, true),
	}
	id := toGlobalID(m.Sender.Uin, m.Id)
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
	id := toGlobalID(m.Sender.Uin, m.Id)
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

// toGlobalID 构建`code`-`msgID`的字符串并返回其CRC32 Checksum的值
func toGlobalID(code int64, msgID int32) int32 {
	return int32(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d-%d", code, msgID))))
}

// Release 释放Bot实例
func (bot *CQBot) Release() {
	if bot.db != nil {
		_ = bot.db.Close()
	}
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

func (bot *CQBot) formatGroupMessage(m *message.GroupMessage) global.MSG {
	cqm := ToStringMessage(m.Elements, m.GroupCode, true)
	gm := global.MSG{
		"anonymous":    nil,
		"font":         0,
		"group_id":     m.GroupCode,
		"message":      ToFormattedMessage(m.Elements, m.GroupCode, false),
		"message_type": "group",
		"message_seq":  m.Id,
		"post_type": func() string {
			if m.Sender.Uin == bot.Client.Uin {
				return "message_sent"
			}
			return "message"
		}(),
		"raw_message": cqm,
		"self_id":     bot.Client.Uin,
		"sender": global.MSG{
			"age":     0,
			"area":    "",
			"level":   "",
			"sex":     "unknown",
			"user_id": m.Sender.Uin,
		},
		"sub_type": "normal",
		"time":     m.Time,
		"user_id":  m.Sender.Uin,
	}
	if m.Sender.IsAnonymous() {
		gm["anonymous"] = global.MSG{
			"flag": m.Sender.AnonymousInfo.AnonymousId + "|" + m.Sender.AnonymousInfo.AnonymousNick,
			"id":   m.Sender.Uin,
			"name": m.Sender.AnonymousInfo.AnonymousNick,
		}
		gm["sender"].(global.MSG)["nickname"] = "匿名消息"
		gm["sub_type"] = "anonymous"
	} else {
		group := bot.Client.FindGroup(m.GroupCode)
		mem := group.FindMember(m.Sender.Uin)
		if mem == nil {
			log.Warnf("获取 %v 成员信息失败，尝试刷新成员列表", m.Sender.Uin)
			t, err := bot.Client.GetGroupMembers(group)
			if err != nil {
				log.Warnf("刷新群 %v 成员列表失败: %v", group.Uin, err)
				return nil
			}
			group.Members = t
			mem = group.FindMember(m.Sender.Uin)
			if mem == nil {
				return nil
			}
		}
		ms := gm["sender"].(global.MSG)
		switch mem.Permission {
		case client.Owner:
			ms["role"] = "owner"
		case client.Administrator:
			ms["role"] = "admin"
		case client.Member:
			ms["role"] = "member"
		default:
			ms["role"] = "member"
		}
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

// IsLawfulImage 判断给定流是否为合法图片
// 返回 是否合法, 实际Mime
// 判断后会自动将 Stream Seek 至 0
func IsLawfulImage(r io.ReadSeeker) (bool, string) {
	if SkipMimeScan {
		return true, ""
	}
	_, _ = r.Seek(0, io.SeekStart)
	defer func() { _, _ = r.Seek(0, io.SeekStart) }()
	t, err := mimetype.DetectReader(r)
	if err != nil {
		log.Debugf("扫描 Mime 时出现问题: %v", err)
		return false, ""
	}
	for _, lt := range lawfulImageTypes {
		if t.Is(lt) {
			return true, t.String()
		}
	}
	return false, t.String()
}
