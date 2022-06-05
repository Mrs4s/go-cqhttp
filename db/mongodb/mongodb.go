package mongodb

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/db"
)

type database struct {
	uri   string
	db    string
	mongo *mongo.Database
}

// config mongodb 相关配置
type config struct {
	Enable   bool   `yaml:"enable"`
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

const (
	MongoGroupMessageCollection        = "group-messages"
	MongoPrivateMessageCollection      = "private-messages"
	MongoGuildChannelMessageCollection = "guild-channel-messages"
)

func init() {
	db.Register("database", func(node yaml.Node) db.Database {
		conf := new(config)
		_ = node.Decode(conf)
		if conf.Database == "" {
			conf.Database = "gocq-database"
		}
		if !conf.Enable {
			return nil
		}
		return &database{uri: conf.URI, db: conf.Database}
	})
}

func (m *database) Open() error {
	cli, err := mongo.Connect(context.Background(), options.Client().ApplyURI(m.uri))
	if err != nil {
		return errors.Wrap(err, "open mongo connection error")
	}
	m.mongo = cli.Database(m.db)
	return nil
}

func (m *database) GetMessageByGlobalID(id int32) (db.StoredMessage, error) {
	if r, err := m.GetGroupMessageByGlobalID(id); err == nil {
		return r, nil
	}
	return m.GetPrivateMessageByGlobalID(id)
}

func (m *database) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
	coll := m.mongo.Collection(MongoGroupMessageCollection)
	var ret db.StoredGroupMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *database) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
	coll := m.mongo.Collection(MongoPrivateMessageCollection)
	var ret db.StoredPrivateMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *database) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	coll := m.mongo.Collection(MongoGuildChannelMessageCollection)
	var ret db.StoredGuildChannelMessage
	if err := coll.FindOne(context.Background(), bson.D{{"_id", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *database) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	coll := m.mongo.Collection(MongoGroupMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}

func (m *database) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	coll := m.mongo.Collection(MongoPrivateMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}

func (m *database) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	coll := m.mongo.Collection(MongoGuildChannelMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}
