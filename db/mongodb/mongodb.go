package mongodb

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/modules/config"
)

type MongoDBImpl struct {
	uri   string
	db    string
	mongo *mongo.Database
}

const (
	MongoGroupMessageCollection        = "group-messages"
	MongoPrivateMessageCollection      = "private-messages"
	MongoGuildChannelMessageCollection = "guild-channel-messages"
)

func init() {
	db.Register("mongodb", func(node yaml.Node) db.Database {
		conf := new(config.MongoDBConfig)
		_ = node.Decode(conf)
		if conf.Database == "" {
			conf.Database = "gocq-database"
		}
		if !conf.Enable {
			return nil
		}
		return &MongoDBImpl{uri: conf.URI, db: conf.Database}
	})
}

func (m *MongoDBImpl) Open() error {
	cli, err := mongo.Connect(context.Background(), options.Client().ApplyURI(m.uri))
	if err != nil {
		return errors.Wrap(err, "open mongo connection error")
	}
	m.mongo = cli.Database(m.db)
	return nil
}

func (m *MongoDBImpl) GetMessageByGlobalID(id int32) (db.StoredMessage, error) {
	if r, err := m.GetGroupMessageByGlobalID(id); err == nil {
		return r, nil
	}
	return m.GetPrivateMessageByGlobalID(id)
}

func (m *MongoDBImpl) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
	coll := m.mongo.Collection(MongoGroupMessageCollection)
	var ret db.StoredGroupMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *MongoDBImpl) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
	coll := m.mongo.Collection(MongoPrivateMessageCollection)
	var ret db.StoredPrivateMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *MongoDBImpl) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	coll := m.mongo.Collection(MongoGuildChannelMessageCollection)
	var ret db.StoredGuildChannelMessage
	if err := coll.FindOne(context.Background(), bson.D{{"_id", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (m *MongoDBImpl) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	coll := m.mongo.Collection(MongoGroupMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}

func (m *MongoDBImpl) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	coll := m.mongo.Collection(MongoPrivateMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}

func (m *MongoDBImpl) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	coll := m.mongo.Collection(MongoGuildChannelMessageCollection)
	_, err := coll.UpdateOne(context.Background(), bson.D{{"_id", msg.ID}}, bson.D{{"$set", msg}}, options.Update().SetUpsert(true))
	return errors.Wrap(err, "insert error")
}
