package db

import (
	"bytes"
	"encoding/gob"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"path"
)

type LevelDBImpl struct {
	db *leveldb.DB
}

const (
	group   byte = 0x0
	private byte = 0x1
)

func UseLevelDB() *LevelDBImpl {
	gob.Register(StoredMessageAttribute{})
	gob.Register(QuotedInfo{})
	gob.Register(global.MSG{})
	gob.Register(StoredGroupMessage{})
	gob.Register(StoredPrivateMessage{})
	return &LevelDBImpl{}
}

func (db *LevelDBImpl) Open() error {
	p := path.Join("data", "leveldb-v2")
	d, err := leveldb.OpenFile(p, &opt.Options{
		WriteBuffer: 128 * opt.KiB,
	})
	if err != nil {
		return errors.Wrap(err, "open level db error")
	}
	db.db = d
	return nil
}

func (db *LevelDBImpl) GetMessageByGlobalID(id int32) (IStoredMessage, error) {
	v, err := db.db.Get(binary.ToBytes(id), nil)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	r := binary.NewReader(v)
	switch r.ReadByte() {
	case group:
		g := &StoredGroupMessage{}
		if err = gob.NewDecoder(bytes.NewReader(r.ReadAvailable())).Decode(g); err != nil {
			return nil, errors.Wrap(err, "decode message error")
		}
		return g, nil
	case private:
		p := &StoredPrivateMessage{}
		if err = gob.NewDecoder(bytes.NewReader(r.ReadAvailable())).Decode(p); err != nil {
			return nil, errors.Wrap(err, "decode message error")
		}
		return p, nil
	default:
		return nil, errors.New("unknown message flag")
	}
}

func (db *LevelDBImpl) GetGroupMessageByGlobalID(id int32) (*StoredGroupMessage, error) {
	i, err := db.GetMessageByGlobalID(id)
	if err != nil {
		return nil, err
	}
	g, ok := i.(*StoredGroupMessage)
	if !ok {
		return nil, errors.New("message type error")
	}
	return g, nil
}

func (db *LevelDBImpl) GetPrivateMessageByGlobalID(id int32) (*StoredPrivateMessage, error) {
	i, err := db.GetMessageByGlobalID(id)
	if err != nil {
		return nil, err
	}
	p, ok := i.(*StoredPrivateMessage)
	if !ok {
		return nil, errors.New("message type error")
	}
	return p, nil
}

func (db *LevelDBImpl) InsertGroupMessage(msg *StoredGroupMessage) error {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return errors.Wrap(err, "encode message error")
	}
	err := db.db.Put(binary.ToBytes(msg.GlobalID), binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(group)
		w.Write(buf.Bytes())
	}), nil)
	return errors.Wrap(err, "put data error")
}

func (db *LevelDBImpl) InsertPrivateMessage(msg *StoredPrivateMessage) error {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return errors.Wrap(err, "encode message error")
	}
	err := db.db.Put(binary.ToBytes(msg.GlobalID), binary.NewWriterF(func(w *binary.Writer) {
		w.WriteByte(private)
		w.Write(buf.Bytes())
	}), nil)
	return errors.Wrap(err, "put data error")
}
