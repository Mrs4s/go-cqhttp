// +build !windows

package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
	data, err := io.ReadAll(io.TeeReader(resp.Body, &wc))
	if err != nil {
		log.Error("更新失败: ", err)
		return
	}
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		log.Error("更新失败: ", err)
		return
	}
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if header.Name == "go-cqhttp" {
			err, _ := FromStream(tr)
			fmt.Println()
			if err != nil {
				log.Error("更新失败!", err)
				return
			}
			log.Info("更新完成！")
		}
	}
}
