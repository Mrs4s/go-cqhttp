package db

import "github.com/pkg/errors"

// MultiDatabase 多数据库支持
// 写入会对所有 Backend 进行写入
// 读取只会读取第一个库
type MultiDatabase struct {
	backends []IDatabase
}

func NewMultiDatabase(backends ...IDatabase) *MultiDatabase {
	return &MultiDatabase{
		backends: backends,
	}
}

func (db *MultiDatabase) UseDB(backend IDatabase) {
	db.backends = append(db.backends, backend)
}

func (db *MultiDatabase) Open() error {
	for _, b := range db.backends {
		if err := b.Open(); err != nil {
			return errors.Wrap(err, "open backend error")
		}
	}
	return nil
}

func (db *MultiDatabase) GetMessageByGlobalID(id int32) (IStoredMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetMessageByGlobalID(id)
}

func (db *MultiDatabase) GetGroupMessageByGlobalID(id int32) (*StoredGroupMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetGroupMessageByGlobalID(id)
}

func (db *MultiDatabase) GetPrivateMessageByGlobalID(id int32) (*StoredPrivateMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetPrivateMessageByGlobalID(id)
}

func (db *MultiDatabase) InsertGroupMessage(m *StoredGroupMessage) error {
	for _, b := range db.backends {
		if err := b.InsertGroupMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}

func (db *MultiDatabase) InsertPrivateMessage(m *StoredPrivateMessage) error {
	for _, b := range db.backends {
		if err := b.InsertPrivateMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}
