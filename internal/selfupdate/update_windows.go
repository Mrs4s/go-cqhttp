package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"

	"github.com/klauspost/compress/zip"
)

// update go-cqhttp自我更新
func update(url string, sum []byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	wc := writeSumCounter{
		hash: sha256.New(),
	}
	rsp, err := io.ReadAll(io.TeeReader(resp.Body, &wc))
	if err != nil {
		return err
	}
	if !bytes.Equal(wc.hash.Sum(nil), sum) {
		return errors.New("文件已损坏")
	}
	reader, _ := zip.NewReader(bytes.NewReader(rsp), resp.ContentLength)
	file, err := reader.Open("go-cqhttp.exe")
	if err != nil {
		return err
	}
	err, _ = fromStream(file)
	if err != nil {
		return err
	}
	return nil
}
