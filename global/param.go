package global

import (
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// MSG 消息Map
type MSG map[string]interface{}

// VersionNameCompare 检查版本名是否需要更新, 仅适用于 go-cqhttp 的版本命名规则
//
// 例: v0.9.29-fix2 == v0.9.29-fix2 -> false
//
// v0.9.29-fix1 < v0.9.29-fix2 -> true
//
// v0.9.29-fix2 > v0.9.29-fix1 -> false
//
// v0.9.29-fix2 < v0.9.30 -> true
//
// v1.0.0-alpha2 < v1.0.0-beta1 -> true
//
// v1.0.0 > v1.0.0-beta1 -> false
func VersionNameCompare(current, remote string) bool {
	defer func() { // 应该不会panic， 为了保险还是加个
		if err := recover(); err != nil {
			log.Warn("检查更新失败！")
		}
	}()
	sp := regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)-?(.+)?`)
	cur := sp.FindStringSubmatch(current)
	re := sp.FindStringSubmatch(remote)
	for i := 1; i <= 3; i++ {
		curSub, _ := strconv.Atoi(cur[i])
		reSub, _ := strconv.Atoi(re[i])
		if curSub != reSub {
			return curSub < reSub
		}
	}
	if cur[4] == "" || re[4] == "" {
		return re[4] == "" && cur[4] != re[4]
	}
	return cur[4] < re[4]
}
