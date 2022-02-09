package leveldb

import (
	"bytes"
	"encoding/gob"
	"path"

	"github.com/Mrs4s/MiraiGo/utils"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/modules/config"
)

type LevelDBImpl struct {
	db *leveldb.DB
}

const (
	group        byte = 0x0
	private      byte = 0x1
	guildChannel byte = 0x2
)

func init() {
	gob.Register(db.StoredMessageAttribute{})
	gob.Register(db.StoredGuildMessageAttribute{})
	gob.Register(db.QuotedInfo{})
	gob.Register(global.MSG{})
	gob.Register(db.StoredGroupMessage{})
	gob.Register(db.StoredPrivateMessage{})
	gob.Register(db.StoredGuildChannelMessage{})

	db.Register("leveldb", func(node yaml.Node) db.Database {
		conf := new(config.LevelDBConfig)
		_ = node.Decode(conf)
		if !conf.Enable {
			return nil
		}
		return &LevelDBImpl{}
	})
}

func (ldb *LevelDBImpl) Open() error {
	p := path.Join("data", "leveldb-v2")
	d, err := leveldb.OpenFile(p, &opt.Options{
		WriteBuffer: 128 * opt.KiB,
	})
	if err != nil {
		return errors.Wrap(err, "open leveldb error")
	}
	ldb.db = d
	return nil
}

func (ldb *LevelDBImpl) GetMessageByGlobalID(id int32) (db.StoredMessage, error) {
	v, err := ldb.db.Get(binary.ToBytes(id), nil)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	r := binary.NewReader(v)
	switch r.ReadByte() {
	case group:
		g := &db.StoredGroupMessage{}
		if err = gob.NewDecoder(bytes.NewReader(r.ReadAvailable())).Decode(g); err != nil {
			return nil, errors.Wrap(err, "decode message error")
		}
		return g, nil
	case private:
		p := &db.StoredPrivateMessage{}
		if err = gob.NewDecoder(bytes.NewReader(r.ReadAvailable())).Decode(p); err != nil {
			return nil, errors.Wrap(err, "decode message error")
		}
		return p, nil
	default:
		return nil, errors.New("unknown message flag")
	}
}

func (ldb *LevelDBImpl) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
	i, err := ldb.GetMessageByGlobalID(id)
	if err != nil {
		return nil, err
	}
	g, ok := i.(*db.StoredGroupMessage)
	if !ok {
		return nil, errors.New("message type error")
	}
	return g, nil
}

func (ldb *LevelDBImpl) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
	i, err := ldb.GetMessageByGlobalID(id)
	if err != nil {
		return nil, err
	}
	p, ok := i.(*db.StoredPrivateMessage)
	if !ok {
		return nil, errors.New("message type error")
	}
	return p, nil
}

func (ldb *LevelDBImpl) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	v, err := ldb.db.Get([]byte(id), nil)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	r := binary.NewReader(v)
	switch r.ReadByte() {
	case guildChannel:
		g := &db.StoredGuildChannelMessage{}
		if err = gob.NewDecoder(bytes.NewReader(r.ReadAvailable())).Decode(g); err != nil {
			return nil, errors.Wrap(err, "decode message error")
		}
		return g, nil
	default:
		return nil, errors.New("unknown message flag")
	}
}

func (ldb *LevelDBImpl) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return errors.Wrap(err, "encode message error")
	}
	err := ldb.db.Put(binary.ToBytes(msg.GlobalID), binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(group)
		w.Write(buf.Bytes())
	}), nil)
	return errors.Wrap(err, "put data error")
}

func (ldb *LevelDBImpl) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return errors.Wrap(err, "encode message error")
	}
	err := ldb.db.Put(binary.ToBytes(msg.GlobalID), binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(private)
		w.Write(buf.Bytes())
	}), nil)
	return errors.Wrap(err, "put data error")
}

func (ldb *LevelDBImpl) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return errors.Wrap(err, "encode message error")
	}
	err := ldb.db.Put(utils.S2B(msg.ID), binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(guildChannel)
		w.Write(buf.Bytes())
	}), nil)
	return errors.Wrap(err, "put data error")
}
