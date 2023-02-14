package global

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/internal/download"
)

// QQMusicSongInfo 通过给定id在QQ音乐上查找曲目信息
func QQMusicSongInfo(id string) (gjson.Result, error) {
	d, err := download.Request{URL: `https://u.y.qq.com/cgi-bin/musicu.fcg?format=json&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0&data={%22comm%22:{%22ct%22:24,%22cv%22:0},%22songinfo%22:{%22method%22:%22get_song_detail_yqq%22,%22param%22:{%22song_type%22:0,%22song_mid%22:%22%22,%22song_id%22:` + id + `},%22module%22:%22music.pf_song_detail_svr%22}}`}.JSON()
	if err != nil {
		return gjson.Result{}, err
	}
	return d.Get("songinfo.data"), nil
}

// NeteaseMusicSongInfo 通过给定id在wdd音乐上查找曲目信息
func NeteaseMusicSongInfo(id string) (gjson.Result, error) {
	d, err := download.Request{URL: fmt.Sprintf("http://music.163.com/api/song/detail/?id=%s&ids=%%5B%s%%5D", id, id)}.JSON()
	if err != nil {
		return gjson.Result{}, err
	}
	return d.Get("songs.0"), nil
}
