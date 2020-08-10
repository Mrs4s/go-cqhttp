package global

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
)

var IMAGE_PATH = path.Join("data", "images")

var VOICE_PATH = path.Join("data", "voices")

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
	_ = ioutil.WriteFile(path, []byte(text), 0777)
}

func Check(err error) {
	if err != nil {
		log.Fatalf("遇到错误: %v", err)
	}
}

func IsAMR(b []byte) bool {
	if len(b) <= 6 {
		return false
	}
	return b[0] == 0x23 && b[1] == 0x21 && b[2] == 0x41 && b[3] == 0x4D && b[4] == 0x52 // amr file header
}
