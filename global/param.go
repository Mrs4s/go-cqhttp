package global

import (
	"github.com/tidwall/gjson"
	"strings"
)

var trueSet = map[string]struct{}{
	"true": struct{}{},
	"yes":  struct{}{},
	"1":    struct{}{},
}

var falseSet = map[string]struct{}{
	"false": struct{}{},
	"no":    struct{}{},
	"0":     struct{}{},
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
