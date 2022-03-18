package db

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// backends 多数据库支持， 后端支持
// 写入会对所有 Backend 进行写入
// 读取只会读取第一个库
var backends []Database

// drivers 多数据库启动
var drivers = make(map[string]func(node yaml.Node) Database)

// DatabaseDisabledError 没有可用的db
var DatabaseDisabledError = errors.New("database disabled")

// Register 添加数据库后端
func Register(name string, init func(yaml.Node) Database) {
	if _, ok := drivers[name]; ok {
		panic("database driver conflict: " + name)
	}
	drivers[name] = init
}

// Init 加载所有后端配置文件
func Init() {
	backends = make([]Database, 0, len(drivers))
	for name, init := range drivers {
		if n, ok := base.Database[name]; ok {
			db := init(n)
			if db != nil {
				backends = append(backends, db)
			}
		}
	}
}

func Open() error {
	for _, b := range backends {
		if err := b.Open(); err != nil {
			return errors.Wrap(err, "open backend error")
		}
	}
	base.Database = nil
	return nil
}

func GetMessageByGlobalID(id int32) (StoredMessage, error) {
	if len(backends) == 0 {
		return nil, DatabaseDisabledError
	}
	return backends[0].GetMessageByGlobalID(id)
}

func GetGroupMessageByGlobalID(id int32) (*StoredGroupMessage, error) {
	if len(backends) == 0 {
		return nil, DatabaseDisabledError
	}
	return backends[0].GetGroupMessageByGlobalID(id)
}

func GetPrivateMessageByGlobalID(id int32) (*StoredPrivateMessage, error) {
	if len(backends) == 0 {
		return nil, DatabaseDisabledError
	}
	return backends[0].GetPrivateMessageByGlobalID(id)
}

func GetGuildChannelMessageByID(id string) (*StoredGuildChannelMessage, error) {
	if len(backends) == 0 {
		return nil, DatabaseDisabledError
	}
	return backends[0].GetGuildChannelMessageByID(id)
}

func InsertGroupMessage(m *StoredGroupMessage) error {
	for _, b := range backends {
		if err := b.InsertGroupMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}

func InsertPrivateMessage(m *StoredPrivateMessage) error {
	for _, b := range backends {
		if err := b.InsertPrivateMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}

func InsertGuildChannelMessage(m *StoredGuildChannelMessage) error {
	for _, b := range backends {
		if err := b.InsertGuildChannelMessage(m); err != nil {
			return errors.Wrap(err, "insert message to backend error")
		}
	}
	return nil
}
