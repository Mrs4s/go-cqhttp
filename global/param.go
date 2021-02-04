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

// EnsureBool 判断给定的p是否可表示为合法Bool类型,否则返回defaultVal
//
// 支持的合法类型有
//
// type bool
//
// type gjson.True or gjson.False
//
// type string "true","yes","1" or "false","no","0" (case insensitive)
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
//
// 例: v0.9.29-fix2 == v0.9.29-fix2 -> false
//
// v0.9.29-fix1 < v0.9.29-fix2 -> true
//
// v0.9.29-fix2 > v0.9.29-fix1 -> false
//
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

// SplitURL 将给定URL字符串分割为两部分，用于URL预处理防止风控
func SplitURL(s string) []string {
	reg := regexp.MustCompile(`(?i)[a-z\d][-a-z\d]{0,62}(\.[a-z\d][-a-z\d]{0,62})+\.?`)
	idx := reg.FindAllStringIndex(s, -1)
	if len(idx) == 0 {
		return []string{s}
	}
	var result []string
	last := 0
	for i := 0; i < len(idx); i++ {
		if len(idx[i]) != 2 {
			continue
		}
		m := int(math.Abs(float64(idx[i][0]-idx[i][1]))/1.5) + idx[i][0]
		result = append(result, s[last:m])
		last = m
	}
	result = append(result, s[last:])
	return result
}
