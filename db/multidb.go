package db

import "github.com/pkg/errors"

// MultiDBImpl 多数据库支持
// 写入会对所有 Backend 进行写入
// 读取只会读取第一个库
type MultiDBImpl struct {
	backends []IDatabase
}

func NewMultiDatabase(backends ...IDatabase) *MultiDBImpl {
	return &MultiDBImpl{
		backends: backends,
	}
}

func (db *MultiDBImpl) UseDB(backend IDatabase) {
	db.backends = append(db.backends, backend)
}

func (db *MultiDBImpl) Open() error {
	for _, b := range db.backends {
		if err := b.Open(); err != nil {
			return errors.Wrap(err, "open backend error")
		}
	}
	return nil
}

func (db *MultiDBImpl) GetMessageByGlobalID(id int32) (IStoredMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetMessageByGlobalID(id)
}

func (db *MultiDBImpl) GetGroupMessageByGlobalID(id int32) (*StoredGroupMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetGroupMessageByGlobalID(id)
}

func (db *MultiDBImpl) GetPrivateMessageByGlobalID(id int32) (*StoredPrivateMessage, error) {
	if len(db.backends) == 0 {
		return nil, errors.New("database disabled")
	}
	return db.backends[0].GetPrivateMessageByGlobalID(id)
}

func (db *MultiDBImpl) InsertGroupMessage(m *StoredGroupMessage) error {
	for _, b := range db.backends {
		if err := b.InsertGroupMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}

func (db *MultiDBImpl) InsertPrivateMessage(m *StoredPrivateMessage) error {
	for _, b := range db.backends {
		if err := b.InsertPrivateMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}
