package global

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
)

var IMAGE_PATH = path.Join("data", "images")

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
