package leveldb

import (
	"path"

	"github.com/Mrs4s/MiraiGo/utils"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/modules/config"
)

type database struct {
	db *leveldb.DB
}

const (
	group        = 0x0
	private      = 0x1
	guildChannel = 0x2
)

func init() {
	db.Register("leveldb", func(node yaml.Node) db.Database {
		conf := new(config.LevelDBConfig)
		_ = node.Decode(conf)
		if !conf.Enable {
			return nil
		}
		return &database{}
	})
}

func (ldb *database) Open() error {
	p := path.Join("data", "leveldb-v3")
	d, err := leveldb.OpenFile(p, &opt.Options{
		WriteBuffer: 32 * opt.KiB,
	})
	if err != nil {
		return errors.Wrap(err, "open leveldb error")
	}
	ldb.db = d
	return nil
}

func (ldb *database) GetMessageByGlobalID(id int32) (_ db.StoredMessage, err error) {
	v, err := ldb.db.Get(binary.ToBytes(id), nil)
	if err != nil || len(v) == 0 {
		return nil, errors.Wrap(err, "get value error")
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("%v", r)
		}
	}()
	r := newReader(utils.B2S(v))
	switch r.uvarint() {
	case group:
		return r.readStoredGroupMessage(), nil
	case private:
		return r.readStoredPrivateMessage(), nil
	default:
		return nil, errors.New("unknown message flag")
	}
}

func (ldb *database) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
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

func (ldb *database) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
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

func (ldb *database) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	v, err := ldb.db.Get([]byte(id), nil)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("%v", r)
		}
	}()
	r := newReader(utils.B2S(v))
	switch r.uvarint() {
	case guildChannel:
		return r.readStoredGuildChannelMessage(), nil
	default:
		return nil, errors.New("unknown message flag")
	}
}

func (ldb *database) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	w := writer{
		stringIndex: map[string]uint64{},
	}
	w.uvarint(group)
	w.writeStoredGroupMessage(msg)
	err := ldb.db.Put(binary.ToBytes(msg.GlobalID), w.bytes(), nil)
	return errors.Wrap(err, "put data error")
}

func (ldb *database) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	w := writer{
		stringIndex: map[string]uint64{},
	}
	w.uvarint(private)
	w.writeStoredPrivateMessage(msg)
	err := ldb.db.Put(binary.ToBytes(msg.GlobalID), w.bytes(), nil)
	return errors.Wrap(err, "put data error")
}

func (ldb *database) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	w := writer{
		stringIndex: map[string]uint64{},
	}
	w.uvarint(guildChannel)
	w.writeStoredGuildChannelMessage(msg)
	err := ldb.db.Put(utils.S2B(msg.ID), w.bytes(), nil)
	return errors.Wrap(err, "put data error")
}
