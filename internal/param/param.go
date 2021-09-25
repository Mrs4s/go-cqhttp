// Package param provide some util for param parse
package param

import (
	"math"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/segmentio/asm/base64"
	"github.com/tidwall/gjson"
)

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
		switch j.Type { // nolint
		case gjson.True:
			return true
		case gjson.False:
			return false
		case gjson.String:
			str = j.Str
		default:
			return defaultVal
		}
	} else if s, ok := p.(string); ok {
		str = s
	}
	str = strings.ToLower(str)
	switch str {
	case "true", "yes", "1":
		return true
	case "false", "no", "0":
		return false
	default:
		return defaultVal
	}
}

var (
	// once lazy compile the reg
	once sync.Once
	// reg is splitURL regex pattern.
	reg *regexp.Regexp
)

// SplitURL 将给定URL字符串分割为两部分，用于URL预处理防止风控
func SplitURL(s string) []string {
	once.Do(func() { // lazy init.
		reg = regexp.MustCompile(`(?i)[a-z\d][-a-z\d]{0,62}(\.[a-z\d][-a-z\d]{0,62})+\.?`)
	})
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

// Base64DecodeString decode base64 with avx2
// see https://github.com/segmentio/asm/issues/50
// avoid incorrect unsafe usage in origin library
func Base64DecodeString(s string) ([]byte, error) {
	e := base64.StdEncoding
	dst := make([]byte, e.DecodedLen(len(s)))
	n, err := e.Decode(dst, utils.S2B(s))
	return dst[:n], err
}

// SetAtDefault 在变量 variable 为默认值 defaultValue 的时候修改为 value
func SetAtDefault(variable, value, defaultValue interface{}) {
	v := reflect.ValueOf(variable)
	v2 := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Interface() != defaultValue {
		return
	}
	if v.Kind() != v2.Kind() {
		return
	}
	v.Set(v2)
}

// SetExcludeDefault 在目标值 value 不为默认值 defaultValue 时修改 variable 为 value
func SetExcludeDefault(variable, value, defaultValue interface{}) {
	v := reflect.ValueOf(variable)
	v2 := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if reflect.Indirect(v2).Interface() != defaultValue {
		return
	}
	if v.Kind() != v2.Kind() {
		return
	}
	v.Set(v2)
}
