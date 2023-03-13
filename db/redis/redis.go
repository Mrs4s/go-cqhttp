package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	GocqhttpGroupMsgKeyPrefix        = "GOCQHTTP_GROUP_MSG:"
	GocqhttpPrivateMsgKeyPrefix      = "GOCQHTTP_PRIVATE_MSG:"
	GocqhttpGuildChannelMsgKeyPrefix = "GOCQHTTP_GUILD_CHANNEL_MSG:"
)

type database struct {
	uri     string
	timeout time.Duration
	rdb     *redis.Client
}

type config struct {
	Enable  bool          `yaml:"enable"`
	URI     string        `yaml:"uri"`
	Timeout time.Duration `yaml:"timeout"`

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
			if conf.Host == "" {
				conf.Host = "127.0.0.1"
			}
			if conf.Port == "" {
				conf.Port = "6379"
			}
			if conf.Database == "" {
				conf.Database = "0"
			}
			conf.URI = fmt.Sprintf("redis://%s:%s/%s", conf.Host, conf.Port, conf.Database)
		}
		log.Debugf("redis registration successful, uri: %s", conf.URI)
		return &database{uri: conf.URI, timeout: conf.Timeout}
	})
}

func (r *database) Open() error {
	opt, err := redis.ParseURL(r.uri)
	if err != nil {
		return errors.Wrap(err, "open redis error")
	}

	rdb := redis.NewClient(opt)
	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return errors.Wrap(err, "ping redis error")
	}

	r.rdb = rdb

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

	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	result, err := r.rdb.Get(ctx, GocqhttpGroupMsgKeyPrefix+strconv.Itoa(int(id))).Result()
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

	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	result, err := r.rdb.Get(ctx, GocqhttpPrivateMsgKeyPrefix+strconv.Itoa(int(id))).Result()
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

	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	result, err := r.rdb.Get(ctx, GocqhttpGuildChannelMsgKeyPrefix+id).Result()
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

	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	err = r.rdb.Set(ctx, GocqhttpGroupMsgKeyPrefix+strconv.Itoa(int(msg.GlobalID)), utils.B2S(jsonData), 0).Err()
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
	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	err = r.rdb.Set(ctx, GocqhttpPrivateMsgKeyPrefix+strconv.Itoa(int(msg.GlobalID)), utils.B2S(jsonData), 0).Err()
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

	ctx, cancelFunc := buildCtx(r.timeout)
	defer cancelFunc()
	err = r.rdb.Set(ctx, GocqhttpGuildChannelMsgKeyPrefix+msg.ID, utils.B2S(jsonData), 0).Err()
	if err != nil {
		return errors.Wrap(err, "set value error")
	}
	return err
}

func buildCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	var ctx context.Context
	if timeout != 0 {
		return context.WithTimeout(context.Background(), timeout)
	} else {
		ctx = context.Background()
		return ctx, func() {}
	}
}
