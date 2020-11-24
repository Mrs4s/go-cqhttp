package global

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var client = &http.Client{
	Timeout: time.Second * 15,
	Transport: &http.Transport{
		Proxy: func(request *http.Request) (u *url.URL, e error) {
			if Proxy == "" {
				return http.ProxyFromEnvironment(request)
			}
			return url.Parse(Proxy)
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

var Proxy string

func GetBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header["User-Agent"] = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36 Edg/83.0.478.61"}
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

func QQMusicSongInfo(id string) (gjson.Result, error) {
	d, err := GetBytes(`https://u.y.qq.com/cgi-bin/musicu.fcg?format=json&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0&data={%22comm%22:{%22ct%22:24,%22cv%22:0},%22songinfo%22:{%22method%22:%22get_song_detail_yqq%22,%22param%22:{%22song_type%22:0,%22song_mid%22:%22%22,%22song_id%22:` + id + `},%22module%22:%22music.pf_song_detail_svr%22}}`)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(d).Get("songinfo.data"), nil
}

func NeteaseMusicSongInfo(id string) (gjson.Result, error) {
	d, err := GetBytes(fmt.Sprintf("http://music.163.com/api/song/detail/?id=%s&ids=%%5B%s%%5D", id, id))
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(d).Get("songs.0"), nil
}
