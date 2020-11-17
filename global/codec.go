package global

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/wdvxdr1123/go-silk/silk"
)

var codec silk.Encoder
var useCodec = true
var once sync.Once

func InitCodec() {
	once.Do(func() {
		log.Info("正在加载silk编码器...")
		err := codec.Init("data/cache", "codec")
		if err != nil {
			log.Error(err)
			useCodec = false
		}
	})
}

func Encoder(data []byte) ([]byte, error) {
	if useCodec == false {
		return nil, errors.New("no silk encoder")
	}
	h := md5.New()
	h.Write(data)
	tempName := fmt.Sprintf("%x", h.Sum(nil))
	if silkPath := path.Join("data/cache", tempName+".silk"); PathExists(silkPath) {
		return ioutil.ReadFile(silkPath)
	}
	slk, err := codec.EncodeToSilk(data, tempName, true)
	if err != nil {
		return nil, err
	}
	return slk, nil
}
