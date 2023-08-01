package sqlite3

import (
	"encoding/base64"
	"hash/crc64"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	sql "github.com/FloatTech/sqlite"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	Enable   bool   `yaml:"enable"`
	CacheTTL string `yaml:"cachettl"`
}

func init() {
	sql.DriverName = "sqlite"
	db.Register("sqlite3", func(node yaml.Node) db.Database {
		conf := new(config)
		_ = node.Decode(conf)
		if !conf.Enable {
			return nil
		}
		duration, err := time.ParseDuration(conf.CacheTTL)
		if err != nil {
			log.Fatalf("illegal ttl config: %v", err)
		}
		return &database{db: new(sql.Sqlite), ttl: duration}
	})
}

func (s *database) Open() error {
	s.db.DBPath = path.Join("data", "sqlite3")
	_ = os.MkdirAll(s.db.DBPath, 0755)
	s.db.DBPath += "/msg.db"
	err := s.db.Open(s.ttl)
	if err != nil {
		return errors.Wrap(err, "open sqlite3 error")
	}
	_, err = s.db.DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return errors.Wrap(err, "enable foreign_keys error")
	}
	err = s.db.Create(Sqlite3UinInfoTableName, &UinInfo{})
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Insert(Sqlite3UinInfoTableName, &UinInfo{Name: "null"})
	if err != nil {
		return errors.Wrap(err, "insert into sqlite3 table "+Sqlite3UinInfoTableName+" error")
	}
	err = s.db.Create(Sqlite3TinyInfoTableName, &TinyInfo{})
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Insert(Sqlite3TinyInfoTableName, &TinyInfo{Name: "null"})
	if err != nil {
		return errors.Wrap(err, "insert into sqlite3 table "+Sqlite3TinyInfoTableName+" error")
	}
	err = s.db.Create(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{},
		"FOREIGN KEY(SenderUin) REFERENCES "+Sqlite3UinInfoTableName+"(Uin)",
	)
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Insert(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{})
	if err != nil {
		return errors.Wrap(err, "insert into sqlite3 table "+Sqlite3MessageAttributeTableName+" error")
	}
	err = s.db.Create(Sqlite3GuildMessageAttributeTableName, &StoredGuildMessageAttribute{},
		"FOREIGN KEY(SenderTinyID) REFERENCES "+Sqlite3TinyInfoTableName+"(ID)",
	)
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Insert(Sqlite3GuildMessageAttributeTableName, &StoredGuildMessageAttribute{})
	if err != nil {
		return errors.Wrap(err, "insert into sqlite3 table "+Sqlite3GuildMessageAttributeTableName+" error")
	}
	err = s.db.Create(Sqlite3QuotedInfoTableName, &QuotedInfo{})
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Insert(Sqlite3QuotedInfoTableName, &QuotedInfo{QuotedContent: "null"})
	if err != nil {
		return errors.Wrap(err, "insert into sqlite3 table "+Sqlite3QuotedInfoTableName+" error")
	}
	err = s.db.Create(Sqlite3GroupMessageTableName, &StoredGroupMessage{},
		"FOREIGN KEY(AttributeID) REFERENCES "+Sqlite3MessageAttributeTableName+"(ID)",
		"FOREIGN KEY(QuotedInfoID) REFERENCES "+Sqlite3QuotedInfoTableName+"(ID)",
	)
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Create(Sqlite3PrivateMessageTableName, &StoredPrivateMessage{},
		"FOREIGN KEY(AttributeID) REFERENCES "+Sqlite3MessageAttributeTableName+"(ID)",
		"FOREIGN KEY(QuotedInfoID) REFERENCES "+Sqlite3QuotedInfoTableName+"(ID)",
	)
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
	}
	err = s.db.Create(Sqlite3GuildChannelMessageTableName, &StoredGuildChannelMessage{},
		"FOREIGN KEY(AttributeID) REFERENCES "+Sqlite3MessageAttributeTableName+"(ID)",
		"FOREIGN KEY(QuotedInfoID) REFERENCES "+Sqlite3QuotedInfoTableName+"(ID)",
	)
	if err != nil {
		return errors.Wrap(err, "create sqlite3 table error")
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
	_ = yaml.Unmarshal(utils.S2B(grpmsg.Content), &ret)
	if grpmsg.AttributeID != 0 {
		var attr StoredMessageAttribute
		s.RLock()
		err = s.db.Find(Sqlite3MessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(grpmsg.AttributeID, 10))
		s.RUnlock()
		if err == nil {
			var uin UinInfo
			s.RLock()
			err = s.db.Find(Sqlite3UinInfoTableName, &uin, "WHERE Uin="+strconv.FormatInt(attr.SenderUin, 10))
			s.RUnlock()
			if err == nil {
				ret.Attribute = &db.StoredMessageAttribute{
					MessageSeq: attr.MessageSeq,
					InternalID: attr.InternalID,
					SenderUin:  attr.SenderUin,
					SenderName: uin.Name,
					Timestamp:  attr.Timestamp,
				}
			}
		}
	}
	if grpmsg.QuotedInfoID != 0 {
		var quoinf QuotedInfo
		s.RLock()
		err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(grpmsg.QuotedInfoID, 10))
		s.RUnlock()
		if err == nil {
			ret.QuotedInfo = &db.QuotedInfo{
				PrevID:       quoinf.PrevID,
				PrevGlobalID: quoinf.PrevGlobalID,
			}
			_ = yaml.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo)
		}
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
	_ = yaml.Unmarshal(utils.S2B(privmsg.Content), &ret)
	if privmsg.AttributeID != 0 {
		var attr StoredMessageAttribute
		s.RLock()
		err = s.db.Find(Sqlite3MessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(privmsg.AttributeID, 10))
		s.RUnlock()
		if err == nil {
			var uin UinInfo
			s.RLock()
			err = s.db.Find(Sqlite3UinInfoTableName, &uin, "WHERE Uin="+strconv.FormatInt(attr.SenderUin, 10))
			s.RUnlock()
			if err == nil {
				ret.Attribute = &db.StoredMessageAttribute{
					MessageSeq: attr.MessageSeq,
					InternalID: attr.InternalID,
					SenderUin:  attr.SenderUin,
					SenderName: uin.Name,
					Timestamp:  attr.Timestamp,
				}
			}
		}
	}
	if privmsg.QuotedInfoID != 0 {
		var quoinf QuotedInfo
		s.RLock()
		err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(privmsg.QuotedInfoID, 10))
		s.RUnlock()
		if err == nil {
			ret.QuotedInfo = &db.QuotedInfo{
				PrevID:       quoinf.PrevID,
				PrevGlobalID: quoinf.PrevGlobalID,
			}
			_ = yaml.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo)
		}
	}
	return &ret, nil
}

func (s *database) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	b, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, errors.Wrap(err, "query invalid id error")
	}
	if len(b) < 25 {
		return nil, errors.New("query invalid id error: content too short")
	}
	var ret db.StoredGuildChannelMessage
	var guildmsg StoredGuildChannelMessage
	s.RLock()
	err = s.db.Find(Sqlite3GuildChannelMessageTableName, &guildmsg, "WHERE ID='"+id+"'")
	s.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	ret.ID = guildmsg.ID
	ret.GuildID = uint64(guildmsg.GuildID)
	ret.ChannelID = uint64(guildmsg.ChannelID)
	_ = yaml.Unmarshal(utils.S2B(guildmsg.Content), &ret)
	if guildmsg.AttributeID != 0 {
		var attr StoredGuildMessageAttribute
		s.RLock()
		err = s.db.Find(Sqlite3GuildMessageAttributeTableName, &attr, "WHERE ID="+strconv.FormatInt(guildmsg.AttributeID, 10))
		s.RUnlock()
		if err == nil {
			var tiny TinyInfo
			s.RLock()
			err = s.db.Find(Sqlite3TinyInfoTableName, &tiny, "WHERE ID="+strconv.FormatInt(attr.SenderTinyID, 10))
			s.RUnlock()
			if err == nil {
				ret.Attribute = &db.StoredGuildMessageAttribute{
					MessageSeq:   uint64(attr.MessageSeq),
					InternalID:   uint64(attr.InternalID),
					SenderTinyID: uint64(attr.SenderTinyID),
					SenderName:   tiny.Name,
					Timestamp:    attr.Timestamp,
				}
			}
		}
	}
	if guildmsg.QuotedInfoID != 0 {
		var quoinf QuotedInfo
		s.RLock()
		err = s.db.Find(Sqlite3QuotedInfoTableName, &quoinf, "WHERE ID="+strconv.FormatInt(guildmsg.QuotedInfoID, 10))
		s.RUnlock()
		if err == nil {
			ret.QuotedInfo = &db.QuotedInfo{
				PrevID:       quoinf.PrevID,
				PrevGlobalID: quoinf.PrevGlobalID,
			}
			_ = yaml.Unmarshal(utils.S2B(quoinf.QuotedContent), &ret.QuotedInfo)
		}
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
		if id == 0 {
			id++
		}
		s.Lock()
		err := s.db.Insert(Sqlite3UinInfoTableName, &UinInfo{
			Uin:  msg.Attribute.SenderUin,
			Name: msg.Attribute.SenderName,
		})
		if err == nil {
			err = s.db.Insert(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{
				ID:         id,
				MessageSeq: msg.Attribute.MessageSeq,
				InternalID: msg.Attribute.InternalID,
				SenderUin:  msg.Attribute.SenderUin,
				Timestamp:  msg.Attribute.Timestamp,
			})
		}
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
		content, err := yaml.Marshal(&msg.QuotedInfo)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		if id == 0 {
			id++
		}
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
	content, err := yaml.Marshal(&msg)
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
		if id == 0 {
			id++
		}
		s.Lock()
		err := s.db.Insert(Sqlite3UinInfoTableName, &UinInfo{
			Uin:  msg.Attribute.SenderUin,
			Name: msg.Attribute.SenderName,
		})
		if err == nil {
			err = s.db.Insert(Sqlite3MessageAttributeTableName, &StoredMessageAttribute{
				ID:         id,
				MessageSeq: msg.Attribute.MessageSeq,
				InternalID: msg.Attribute.InternalID,
				SenderUin:  msg.Attribute.SenderUin,
				Timestamp:  msg.Attribute.Timestamp,
			})
		}
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
		content, err := yaml.Marshal(&msg.QuotedInfo)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		if id == 0 {
			id++
		}
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
	content, err := yaml.Marshal(&msg)
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
		if id == 0 {
			id++
		}
		s.Lock()
		err := s.db.Insert(Sqlite3TinyInfoTableName, &TinyInfo{
			ID:   int64(msg.Attribute.SenderTinyID),
			Name: msg.Attribute.SenderName,
		})
		if err == nil {
			err = s.db.Insert(Sqlite3MessageAttributeTableName, &StoredGuildMessageAttribute{
				ID:           id,
				MessageSeq:   int64(msg.Attribute.MessageSeq),
				InternalID:   int64(msg.Attribute.InternalID),
				SenderTinyID: int64(msg.Attribute.SenderTinyID),
				Timestamp:    msg.Attribute.Timestamp,
			})
		}
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
		content, err := yaml.Marshal(&msg.QuotedInfo)
		if err != nil {
			return errors.Wrap(err, "insert marshal QuotedContent error")
		}
		h.Write(content)
		id := int64(h.Sum64())
		if id == 0 {
			id++
		}
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
	content, err := yaml.Marshal(&msg)
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
