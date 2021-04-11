package update

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Update go-cqhttp自我更新
func Update(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Error("更新失败: ", err)
		return
	}
	defer resp.Body.Close()
	wc := WriteCounter{}
	rsp, _ := io.ReadAll(io.TeeReader(resp.Body, &wc))
	reader, _ := zip.NewReader(bytes.NewReader(rsp), resp.ContentLength)
	file, err := reader.Open("go-cqhttp.exe")
	if err != nil {
		log.Error("更新失败!", err)
		return
	}
	err, _ = FromStream(file)
	fmt.Println()
	if err != nil {
		log.Error("更新失败!", err)
		return
	}
	log.Info("更新完成！")
}
