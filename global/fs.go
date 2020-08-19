package global

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

var (
	IMAGE_PATH = path.Join("data", "images")
	VOICE_PATH = path.Join("data", "voices")
	VIDEO_PATH = path.Join("data", "videos")
	CACHE_PATH = path.Join("data", "cache")

	HEADER_AMR  = []byte("#!AMR")
	HEADER_SILK = []byte("\x02#!SILK_V3")
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func ReadAllText(path string) string {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func WriteAllText(path, text string) {
	_ = ioutil.WriteFile(path, []byte(text), 0644)
}

func Check(err error) {
	if err != nil {
		log.Fatalf("遇到错误: %v", err)
	}
}

func IsAMRorSILK(b []byte) bool {
	return bytes.HasPrefix(b, HEADER_AMR) || bytes.HasPrefix(b, HEADER_SILK)
}
