package db

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBImpl struct {
	uri   string
	db    string
	mongo *mongo.Database
}

const (
	MongoGroupMessageCollection   = "group-messages"
	MongoPrivateMessageCollection = "private-messages"
)

func UseMongoDB(uri, db string) *MongoDBImpl {
	return &MongoDBImpl{uri: uri, db: db}
}

func (db *MongoDBImpl) Open() error {
	cli, err := mongo.Connect(context.Background(), options.Client().ApplyURI(db.uri))
	if err != nil {
		return errors.Wrap(err, "open mongo connection error")
	}
	db.mongo = cli.Database(db.db)
	return nil
}

func (db *MongoDBImpl) GetMessageByGlobalID(id int32) (IStoredMessage, error) {
	if r, err := db.GetGroupMessageByGlobalID(id); err == nil {
		return r, nil
	}
	return db.GetPrivateMessageByGlobalID(id)
}

func (db *MongoDBImpl) GetGroupMessageByGlobalID(id int32) (*StoredGroupMessage, error) {
	coll := db.mongo.Collection(MongoGroupMessageCollection)
	var ret StoredGroupMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (db *MongoDBImpl) GetPrivateMessageByGlobalID(id int32) (*StoredPrivateMessage, error) {
	coll := db.mongo.Collection(MongoPrivateMessageCollection)
	var ret StoredPrivateMessage
	if err := coll.FindOne(context.Background(), bson.D{{"globalId", id}}).Decode(&ret); err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &ret, nil
}

func (db *MongoDBImpl) InsertGroupMessage(msg *StoredGroupMessage) error {
	coll := db.mongo.Collection(MongoGroupMessageCollection)
	_, err := coll.InsertOne(context.Background(), msg)
	return errors.Wrap(err, "insert error")
}

func (db *MongoDBImpl) InsertPrivateMessage(msg *StoredPrivateMessage) error {
	coll := db.mongo.Collection(MongoPrivateMessageCollection)
	_, err := coll.InsertOne(context.Background(), msg)
	return errors.Wrap(err, "insert error")
}
