package global

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guonaihong/gout"
	"github.com/pkg/errors"

	"github.com/tidwall/gjson"
)

var (
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: func(request *http.Request) (u *url.URL, e error) {
				if Proxy == "" {
					return http.ProxyFromEnvironment(request)
				}
				return url.Parse(Proxy)
			},
			ForceAttemptHTTP2:   true,
			MaxConnsPerHost:     0,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 999,
		},
	}

	//Proxy 存储Config.proxy_rewrite,用于设置代理
	Proxy string

	//ErrOverSize 响应主体过大时返回此错误
	ErrOverSize = errors.New("oversize")

	//UserAgent HTTP请求时使用的UA
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36 Edg/87.0.664.66"
)

//GetBytes 对给定URL发送Get请求，返回响应主体
func GetBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header["User-Agent"] = []string{UserAgent}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		buffer := bytes.NewBuffer(body)
		r, _ := gzip.NewReader(buffer)
		defer r.Close()
		unCom, err := ioutil.ReadAll(r)
		return unCom, err
	}
	return body, nil
}

//DownloadFile 将给定URL对应的文件下载至给定Path
func DownloadFile(url, path string, limit int64, headers map[string]string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if _, ok := headers["User-Agent"]; !ok {
		req.Header["User-Agent"] = []string{UserAgent}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if limit > 0 && resp.ContentLength > limit {
		return ErrOverSize
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

//DownloadFileMultiThreading 使用threadCount个线程将给定URL对应的文件下载至给定Path
func DownloadFileMultiThreading(url, path string, limit int64, threadCount int, headers map[string]string) error {
	if threadCount < 2 {
		return DownloadFile(url, path, limit, headers)
	}
	type BlockMetaData struct {
		BeginOffset    int64
		EndOffset      int64
		DownloadedSize int64
	}
	var blocks []*BlockMetaData
	var contentLength int64
	errUnsupportedMultiThreading := errors.New("unsupported multi-threading")
	// 初始化分块或直接下载
	initOrDownload := func() error {
		copyStream := func(s io.ReadCloser) error {
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err = io.Copy(file, s); err != nil {
				return err
			}
			return errUnsupportedMultiThreading
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		for k, v := range headers {
			req.Header.Set(k, v)

		}
		if _, ok := headers["User-Agent"]; !ok {
			req.Header["User-Agent"] = []string{UserAgent}
		}
		req.Header.Set("range", "bytes=0-")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New("response status unsuccessful: " + strconv.FormatInt(int64(resp.StatusCode), 10))
		}
		if resp.StatusCode == 200 {
			if limit > 0 && resp.ContentLength > limit {
				return ErrOverSize
			}
			return copyStream(resp.Body)
		}
		if resp.StatusCode == 206 {
			contentLength = resp.ContentLength
			if limit > 0 && resp.ContentLength > limit {
				return ErrOverSize
			}
			blockSize := func() int64 {
				if contentLength > 1024*1024 {
					return (contentLength / int64(threadCount)) - 10
				}
				return contentLength

			}()
			if blockSize == contentLength {
				return copyStream(resp.Body)
			}
			var tmp int64
			for tmp+blockSize < contentLength {
				blocks = append(blocks, &BlockMetaData{
					BeginOffset: tmp,
					EndOffset:   tmp + blockSize - 1,
				})
				tmp += blockSize
			}
			blocks = append(blocks, &BlockMetaData{
				BeginOffset: tmp,
				EndOffset:   contentLength - 1,
			})
			return nil
		}
		return errors.New("unknown status code")
	}
	// 下载分块
	downloadBlock := func(block *BlockMetaData) error {
		req, _ := http.NewRequest("GET", url, nil)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer file.Close()
		_, _ = file.Seek(block.BeginOffset, io.SeekStart)
		writer := bufio.NewWriter(file)
		defer writer.Flush()

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		if _, ok := headers["User-Agent"]; ok {
			req.Header["User-Agent"] = []string{UserAgent}
		}
		req.Header.Set("range", "bytes="+strconv.FormatInt(block.BeginOffset, 10)+"-"+strconv.FormatInt(block.EndOffset, 10))
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New("response status unsuccessful: " + strconv.FormatInt(int64(resp.StatusCode), 10))
		}
		var buffer = make([]byte, 1024)
		i, err := resp.Body.Read(buffer)
		for {
			if err != nil && err != io.EOF {
				return err
			}
			i64 := int64(len(buffer[:i]))
			needSize := block.EndOffset + 1 - block.BeginOffset
			if i64 > needSize {
				i64 = needSize
				err = io.EOF
			}
			_, e := writer.Write(buffer[:i64])
			if e != nil {
				return e
			}
			block.BeginOffset += i64
			block.DownloadedSize += i64
			if err == io.EOF || block.BeginOffset > block.EndOffset {
				break
			}
			i, err = resp.Body.Read(buffer)
		}
		return nil
	}

	if err := initOrDownload(); err != nil {
		if err == errUnsupportedMultiThreading {
			return nil
		}
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(blocks))
	var lastErr error
	for i := range blocks {
		go func(b *BlockMetaData) {
			defer wg.Done()
			if err := downloadBlock(b); err != nil {
				lastErr = err
			}
		}(blocks[i])
	}
	wg.Wait()
	return lastErr
}

//GetSliderTicket 通过给定的验证链接raw和id,获取验证结果Ticket
func GetSliderTicket(raw, id string) (string, error) {
	var rsp string
	if err := gout.POST("https://api.shkong.com/gocqhttpapi/task").SetJSON(gout.H{
		"id":  id,
		"url": raw,
	}).SetTimeout(time.Second * 35).BindBody(&rsp).Do(); err != nil {
		return "", err
	}
	g := gjson.Parse(rsp)
	if g.Get("error").Str != "" {
		return "", errors.New(g.Get("error").Str)
	}
	return g.Get("ticket").Str, nil
}

//QQMusicSongInfo 通过给定id在QQ音乐上查找曲目信息
func QQMusicSongInfo(id string) (gjson.Result, error) {
	d, err := GetBytes(`https://u.y.qq.com/cgi-bin/musicu.fcg?format=json&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0&data={%22comm%22:{%22ct%22:24,%22cv%22:0},%22songinfo%22:{%22method%22:%22get_song_detail_yqq%22,%22param%22:{%22song_type%22:0,%22song_mid%22:%22%22,%22song_id%22:` + id + `},%22module%22:%22music.pf_song_detail_svr%22}}`)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(d).Get("songinfo.data"), nil
}

//NeteaseMusicSongInfo 通过给定id在wdd音乐上查找曲目信息
func NeteaseMusicSongInfo(id string) (gjson.Result, error) {
	d, err := GetBytes(fmt.Sprintf("http://music.163.com/api/song/detail/?id=%s&ids=%%5B%s%%5D", id, id))
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(d).Get("songs.0"), nil
}
