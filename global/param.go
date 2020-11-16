package global

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var trueSet = map[string]struct{}{
	"true": {},
	"yes":  {},
	"1":    {},
}

var falseSet = map[string]struct{}{
	"false": {},
	"no":    {},
	"0":     {},
}

func EnsureBool(p interface{}, defaultVal bool) bool {
	var str string
	if b, ok := p.(bool); ok {
		return b
	}
	if j, ok := p.(gjson.Result); ok {
		if !j.Exists() {
			return defaultVal
		}
		if j.Type == gjson.True {
			return true
		}
		if j.Type == gjson.False {
			return false
		}
		if j.Type != gjson.String {
			return defaultVal
		}
		str = j.Str
	} else if s, ok := p.(string); ok {
		str = s
	}
	str = strings.ToLower(str)
	if _, ok := trueSet[str]; ok {
		return true
	}
	if _, ok := falseSet[str]; ok {
		return false
	}
	return defaultVal
}

// VersionNameCompare 检查版本名是否需要更新, 仅适用于 go-cqhttp 的版本命名规则
// 例: v0.9.29-fix2 == v0.9.29-fix2 -> false
// v0.9.29-fix1 < v0.9.29-fix2 -> true
// v0.9.29-fix2 > v0.9.29-fix1 -> false
// v0.9.29-fix2 < v0.9.30 -> true
func VersionNameCompare(current, remote string) bool {
	sp := regexp.MustCompile(`[0-9]\d*`)
	cur := sp.FindAllStringSubmatch(current, -1)
	re := sp.FindAllStringSubmatch(remote, -1)
	for i := 0; i < int(math.Min(float64(len(cur)), float64(len(re)))); i++ {
		curSub, _ := strconv.Atoi(cur[i][0])
		reSub, _ := strconv.Atoi(re[i][0])
		if curSub < reSub {
			return true
		}
	}
	return len(cur) < len(re)
}
