package sqlite3

import (
	"encoding/json"
	"hash/crc64"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	sql "github.com/FloatTech/sqlite"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/Mrs4s/go-cqhttp/db"
)

type database struct {
	sync.RWMutex
	db  *sql.Sqlite
	ttl time.Duration
}

// config mongodb 相关配置
type config struct {
	Enable   bool          `yaml:"enable"`
	CacheTTL time.Duration `yaml:"cachettl"`
}

func init() {
	db.Register("sqlite3", func(node yaml.Node) db.Database {
		conf := new(config)
		_ = node.Decode(conf)
		if !conf.Enable {
			return nil
		}
		return &database{db: new(sql.Sqlite), ttl: conf.CacheTTL}
	})
}

func (s *database) Open() error {
	s.db.DBPath = path.Join("data", "sqlite3")
	_ = os.MkdirAll(s.db.DBPath, 0755)
	err := s.db.Open(s.ttl)
	if err != nil {
		return errors.Wrap(err, "open sqlite3 error")
	}
	return nil
}

func (s *database) GetMessageByGlobalID(id int32) (db.StoredMessage, error) {
	if r, err := s.GetGroupMessageByGlobalID(id); err == nil {
		return r, nil
	}
	return s.GetPrivateMessageByGlobalID(id)
}

func (s *database) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
	var ret db.StoredGroupMessage
	var grpmsg StoredGroupMessage
	s.RLock()
	err := s.db.Find(Sqlite3GroupMessageTableName, &grpmsg, "WHERE GlobalID="+strconv.Itoa(int(id)))
	s.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	ret.ID = grpmsg.ID
	ret.GlobalID = grpmsg.GlobalID
	ret.SubType = grpmsg.SubType
	ret.GroupCode = grpmsg.GroupCode
	ret.AnonymousID = grpmsg.AnonymousID
	_ = json.Unmarshal(utils.S2B(grpmsg.Content), &ret.Content)
	var attr StoredMessageAttribute
	s.RLock()
	err = s.db.Find(Sqlite3MessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(grpmsg.AttributeID, 10))
	s.RUnlock()
	if err == nil {
		ret.Attribute = &db.StoredMessageAttribute{
			MessageSeq: attr.MessageSeq,
			InternalID: attr.InternalID,
			SenderUin:  attr.SenderUin,
			SenderName: attr.SenderName,
			Timestamp:  attr.Timestamp,
		}
	}
	var quoinf QuotedInfo
	s.RLock()
	err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(grpmsg.QuotedInfoID, 10))
	s.RUnlock()
	if err == nil {
		ret.QuotedInfo = &db.QuotedInfo{
			PrevID:       quoinf.PrevID,
			PrevGlobalID: quoinf.PrevGlobalID,
		}
		_ = json.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo.QuotedContent)
	}
	return &ret, nil
}

func (s *database) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
	var ret db.StoredPrivateMessage
	var privmsg StoredPrivateMessage
	s.RLock()
	err := s.db.Find(Sqlite3PrivateMessageTableName, &privmsg, "WHERE GlobalID="+strconv.Itoa(int(id)))
	s.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	ret.ID = privmsg.ID
	ret.GlobalID = privmsg.GlobalID
	ret.SubType = privmsg.SubType
	ret.SessionUin = privmsg.SessionUin
	ret.TargetUin = privmsg.TargetUin
	_ = json.Unmarshal(utils.S2B(privmsg.Content), &ret.Content)
	var attr StoredMessageAttribute
	s.RLock()
	err = s.db.Find(Sqlite3MessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(privmsg.AttributeID, 10))
	s.RUnlock()
	if err == nil {
		ret.Attribute = &db.StoredMessageAttribute{
			MessageSeq: attr.MessageSeq,
			InternalID: attr.InternalID,
			SenderUin:  attr.SenderUin,
			SenderName: attr.SenderName,
			Timestamp:  attr.Timestamp,
		}
	}
	var quoinf QuotedInfo
	s.RLock()
	err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(privmsg.QuotedInfoID, 10))
	s.RUnlock()
	if err == nil {
		ret.QuotedInfo = &db.QuotedInfo{
			PrevID:       quoinf.PrevID,
			PrevGlobalID: quoinf.PrevGlobalID,
		}
		_ = json.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo.QuotedContent)
	}
	return &ret, nil
}

func (s *database) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	var ret db.StoredGuildChannelMessage
	var guildmsg StoredGuildChannelMessage
	s.RLock()
	err := s.db.Find(Sqlite3GuildChannelMessageTableName, &guildmsg, "WHERE ID='"+id+"'")
	s.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	ret.ID = guildmsg.ID
	ret.GuildID = uint64(guildmsg.GuildID)
	ret.ChannelID = uint64(guildmsg.ChannelID)
	_ = json.Unmarshal(utils.S2B(guildmsg.Content), &ret.Content)
	var attr StoredGuildMessageAttribute
	s.RLock()
	err = s.db.Find(Sqlite3GuildMessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(guildmsg.AttributeID, 10))
	s.RUnlock()
	if err == nil {
		ret.Attribute = &db.StoredGuildMessageAttribute{
			MessageSeq:   uint64(attr.MessageSeq),
			InternalID:   uint64(attr.InternalID),
			SenderTinyID: uint64(attr.SenderTinyID),
			SenderName:   attr.SenderName,
			Timestamp:    attr.Timestamp,
		}
	}
	var quoinf QuotedInfo
	s.RLock()
	err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(guildmsg.QuotedInfoID, 10))
	s.RUnlock()
	if err == nil {
		ret.QuotedInfo = &db.QuotedInfo{
			PrevID:       quoinf.PrevID,
			PrevGlobalID: quoinf.PrevGlobalID,
		}
		_ = json.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo.QuotedContent)
	}
	return &ret, nil
}

func (s *database) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	grpmsg := &StoredGroupMessage{
		GlobalID:    msg.GlobalID,
		ID:          msg.ID,
		SubType:     msg.SubType,
		GroupCode:   msg.GroupCode,
		AnonymousID: msg.AnonymousID,
	}
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	if msg.Attribute != nil {
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.Attribute.MessageSeq))
			w.WriteUInt32(uint32(msg.Attribute.InternalID))
			w.WriteUInt64(uint64(msg.Attribute.SenderUin))
			w.WriteUInt64(uint64(msg.Attribute.Timestamp))
		}))
		h.Write(utils.S2B(msg.Attribute.SenderName))
		id := int64(h.Sum64())
		s.Lock()
		err := s.db.Insert(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{
			ID:         id,
			MessageSeq: msg.Attribute.MessageSeq,
			InternalID: msg.Attribute.InternalID,
			SenderUin:  msg.Attribute.SenderUin,
			SenderName: msg.Attribute.SenderName,
			Timestamp:  msg.Attribute.Timestamp,
		})
		s.Unlock()
		if err == nil {
			grpmsg.AttributeID = id
		}
		h.Reset()
	}
	if msg.QuotedInfo != nil {
		h.Write(utils.S2B(msg.QuotedInfo.PrevID))
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.QuotedInfo.PrevGlobalID))
		}))
		content, err := json.Marshal(&msg.QuotedInfo.QuotedContent)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		s.Lock()
		err = s.db.Insert(Sqlite3QuotedInfoTableName, &QuotedInfo{
			ID:            id,
			PrevID:        msg.QuotedInfo.PrevID,
			PrevGlobalID:  msg.QuotedInfo.PrevGlobalID,
			QuotedContent: utils.B2S(content),
		})
		s.Unlock()
		if err == nil {
			grpmsg.QuotedInfoID = id
		}
	}
	content, err := json.Marshal(&msg.Content)
	if err != nil {
		return errors.Wrap(err, "insert marshal Content error")
	}
	grpmsg.Content = utils.B2S(content)
	s.Lock()
	err = s.db.Insert(Sqlite3GroupMessageTableName, grpmsg)
	s.Unlock()
	if err != nil {
		return errors.Wrap(err, "insert error")
	}
	return nil
}

func (s *database) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	privmsg := &StoredPrivateMessage{
		GlobalID:   msg.GlobalID,
		ID:         msg.ID,
		SubType:    msg.SubType,
		SessionUin: msg.SessionUin,
		TargetUin:  msg.TargetUin,
	}
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	if msg.Attribute != nil {
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.Attribute.MessageSeq))
			w.WriteUInt32(uint32(msg.Attribute.InternalID))
			w.WriteUInt64(uint64(msg.Attribute.SenderUin))
			w.WriteUInt64(uint64(msg.Attribute.Timestamp))
		}))
		h.Write(utils.S2B(msg.Attribute.SenderName))
		id := int64(h.Sum64())
		s.Lock()
		err := s.db.Insert(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{
			ID:         id,
			MessageSeq: msg.Attribute.MessageSeq,
			InternalID: msg.Attribute.InternalID,
			SenderUin:  msg.Attribute.SenderUin,
			SenderName: msg.Attribute.SenderName,
			Timestamp:  msg.Attribute.Timestamp,
		})
		s.Unlock()
		if err == nil {
			privmsg.AttributeID = id
		}
		h.Reset()
	}
	if msg.QuotedInfo != nil {
		h.Write(utils.S2B(msg.QuotedInfo.PrevID))
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.QuotedInfo.PrevGlobalID))
		}))
		content, err := json.Marshal(&msg.QuotedInfo.QuotedContent)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		s.Lock()
		err = s.db.Insert(Sqlite3QuotedInfoTableName, &QuotedInfo{
			ID:            id,
			PrevID:        msg.QuotedInfo.PrevID,
			PrevGlobalID:  msg.QuotedInfo.PrevGlobalID,
			QuotedContent: utils.B2S(content),
		})
		s.Unlock()
		if err == nil {
			privmsg.QuotedInfoID = id
		}
	}
	content, err := json.Marshal(&msg.Content)
	if err != nil {
		return errors.Wrap(err, "insert marshal Content error")
	}
	privmsg.Content = utils.B2S(content)
	s.Lock()
	err = s.db.Insert(Sqlite3PrivateMessageTableName, privmsg)
	s.Unlock()
	if err != nil {
		return errors.Wrap(err, "insert error")
	}
	return nil
}

func (s *database) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	guildmsg := &StoredGuildChannelMessage{
		ID:        msg.ID,
		GuildID:   int64(msg.GuildID),
		ChannelID: int64(msg.ChannelID),
	}
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	if msg.Attribute != nil {
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.Attribute.MessageSeq))
			w.WriteUInt32(uint32(msg.Attribute.InternalID))
			w.WriteUInt64(uint64(msg.Attribute.SenderTinyID))
			w.WriteUInt64(uint64(msg.Attribute.Timestamp))
		}))
		h.Write(utils.S2B(msg.Attribute.SenderName))
		id := int64(h.Sum64())
		s.Lock()
		err := s.db.Insert(Sqlite3MessageAttributeTableName, &StoredGuildMessageAttribute{
			ID:           id,
			MessageSeq:   int64(msg.Attribute.MessageSeq),
			InternalID:   int64(msg.Attribute.InternalID),
			SenderTinyID: int64(msg.Attribute.SenderTinyID),
			SenderName:   msg.Attribute.SenderName,
			Timestamp:    msg.Attribute.Timestamp,
		})
		s.Unlock()
		if err == nil {
			guildmsg.AttributeID = id
		}
		h.Reset()
	}
	if msg.QuotedInfo != nil {
		h.Write(utils.S2B(msg.QuotedInfo.PrevID))
		h.Write(binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt32(uint32(msg.QuotedInfo.PrevGlobalID))
		}))
		content, err := json.Marshal(&msg.QuotedInfo.QuotedContent)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		s.Lock()
		err = s.db.Insert(Sqlite3QuotedInfoTableName, &QuotedInfo{
			ID:            id,
			PrevID:        msg.QuotedInfo.PrevID,
			PrevGlobalID:  msg.QuotedInfo.PrevGlobalID,
			QuotedContent: utils.B2S(content),
		})
		s.Unlock()
		if err == nil {
			guildmsg.QuotedInfoID = id
		}
	}
	content, err := json.Marshal(&msg.Content)
	if err != nil {
		return errors.Wrap(err, "insert marshal Content error")
	}
	guildmsg.Content = utils.B2S(content)
	s.Lock()
	err = s.db.Insert(Sqlite3GuildChannelMessageTableName, guildmsg)
	s.Unlock()
	if err != nil {
		return errors.Wrap(err, "insert error")
	}
	return nil
}
