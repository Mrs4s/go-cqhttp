package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	GocqhttpGroupMsgKey        = "GOCQHTTP_GROUP_MSG:"
	GocqhttpPrivateMsgKey      = "GOCQHTTP_PRIVATE_MSG:"
	GocqhttpGuildChannelMsgKey = "GOCQHTTP_GUILD_CHANNEL_MSG:"
)

type database struct {
	uri string
	rdb *redis.Client
	ctx *context.Context
}

type config struct {
	Enable bool   `yaml:"enable"`
	URI    string `yaml:"uri"`

	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
}

func init() {
	db.Register("redis", func(node yaml.Node) db.Database {
		log.Debug("begin registering redis")
		conf := new(config)
		_ = node.Decode(conf)
		if !conf.Enable {
			return nil
		}
		if conf.URI == "" {
			if conf.Port == "" {
				conf.Port = "6379"
			}
			if conf.Database == "" {
				conf.Database = "0"
			}
			conf.URI = fmt.Sprintf("redis://%s:%s/%s", conf.Host, conf.Port, conf.Database)
		}
		log.Debugf("redis registration successful, uri: %s", conf.URI)
		return &database{uri: conf.URI}
	})
}

func (r *database) Open() error {
	opt, err := redis.ParseURL(r.uri)
	if err != nil {
		return errors.Wrap(err, "open redis error")
	}
	rdb := redis.NewClient(opt)
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return errors.Wrap(err, "ping redis error")
	}

	r.rdb = rdb
	r.ctx = &ctx

	return nil
}

func (r *database) GetMessageByGlobalID(id int32) (db.StoredMessage, error) {
	log.Debugf("get message, id=%d", id)
	if r, err := r.GetGroupMessageByGlobalID(id); err == nil {
		return r, nil
	}
	return r.GetPrivateMessageByGlobalID(id)
}

func (r *database) GetGroupMessageByGlobalID(id int32) (*db.StoredGroupMessage, error) {
	log.Debugf("get group message, id=%d", id)
	msg := &db.StoredGroupMessage{}
	result, err := r.rdb.Get(*r.ctx, GocqhttpGroupMsgKey+strconv.Itoa(int(id))).Result()
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	err = json.Unmarshal(utils.S2B(result), msg)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	return msg, nil
}

func (r *database) GetPrivateMessageByGlobalID(id int32) (*db.StoredPrivateMessage, error) {
	log.Debugf("get private message, id=%d", id)
	msg := &db.StoredPrivateMessage{}
	result, err := r.rdb.Get(*r.ctx, GocqhttpPrivateMsgKey+strconv.Itoa(int(id))).Result()
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	err = json.Unmarshal(utils.S2B(result), msg)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	return msg, nil
}

func (r *database) GetGuildChannelMessageByID(id string) (*db.StoredGuildChannelMessage, error) {
	log.Debugf("get guild channel message, id=%s", id)
	msg := &db.StoredGuildChannelMessage{}
	result, err := r.rdb.Get(*r.ctx, GocqhttpGuildChannelMsgKey+id).Result()
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	err = json.Unmarshal(utils.S2B(result), msg)
	if err != nil {
		return nil, errors.Wrap(err, "get value error")
	}
	return msg, nil
}

func (r *database) InsertGroupMessage(msg *db.StoredGroupMessage) error {
	log.Debugf("set group message, id=%d", msg.GlobalID)
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	err = r.rdb.Set(*r.ctx, GocqhttpGroupMsgKey+strconv.Itoa(int(msg.GlobalID)), utils.B2S(jsonData), 0).Err()
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	return err
}

func (r *database) InsertPrivateMessage(msg *db.StoredPrivateMessage) error {
	log.Debugf("set private message, id=%d", msg.GlobalID)
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	err = r.rdb.Set(*r.ctx, GocqhttpPrivateMsgKey+strconv.Itoa(int(msg.GlobalID)), utils.B2S(jsonData), 0).Err()
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	return err
}

func (r *database) InsertGuildChannelMessage(msg *db.StoredGuildChannelMessage) error {
	log.Debugf("set guild channel message, id=%s", msg.ID)
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	err = r.rdb.Set(*r.ctx, GocqhttpGuildChannelMsgKey+msg.ID, utils.B2S(jsonData), 0).Err()
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	return err
}
