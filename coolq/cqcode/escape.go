// Package cqcode provides CQCode util functions.
package cqcode

import "strings"

// EscapeText 将字符串raw中部分字符转义
//
//   - & -> &amp;
//   - [ -> &#91;
//   - ] -> &#93;
func EscapeText(s string) string {
	count := strings.Count(s, "&")
	count += strings.Count(s, "[")
	count += strings.Count(s, "]")
	if count == 0 {
		return s
	}

	// Apply replacements to buffer.
	var b strings.Builder
	b.Grow(len(s) + count*4)
	start := 0
	for i := 0; i < count; i++ {
		j := start
		for index, r := range s[start:] {
			if r == '&' || r == '[' || r == ']' {
				j += index
				break
			}
		}
		b.WriteString(s[start:j])
		switch s[j] {
		case '&':
			b.WriteString("&amp;")
		case '[':
			b.WriteString("&#91;")
		case ']':
			b.WriteString("&#93;")
		}
		start = j + 1
	}
	b.WriteString(s[start:])
	return b.String()
}

// EscapeValue 将字符串value中部分字符转义
//
//   - , -> &#44;
//   - & -> &amp;
//   - [ -> &#91;
//   - ] -> &#93;
func EscapeValue(value string) string {
	ret := EscapeText(value)
	return strings.ReplaceAll(ret, ",", "&#44;")
}

// UnescapeText 将字符串content中部分字符反转义
//
//   - &amp; -> &
//   - &#91; -> [
//   - &#93; -> ]
func UnescapeText(content string) string {
	ret := content
	ret = strings.ReplaceAll(ret, "&#91;", "[")
	ret = strings.ReplaceAll(ret, "&#93;", "]")
	ret = strings.ReplaceAll(ret, "&amp;", "&")
	return ret
}

// UnescapeValue 将字符串content中部分字符反转义
//
//   - &#44; -> ,
//   - &amp; -> &
//   - &#91; -> [
//   - &#93; -> ]
func UnescapeValue(content string) string {
	ret := strings.ReplaceAll(content, "&#44;", ",")
	return UnescapeText(ret)
}
