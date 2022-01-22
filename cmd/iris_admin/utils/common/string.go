package common

// LimitedString 防止字符串过长
func LimitedString(str string) string {
	limited := [14]rune{10: ' ', 11: '.', 12: '.', 13: '.'}
	i := 0
	for _, r := range str {
		if i >= 10 {
			break
		}
		limited[i] = r
		i++
	}
	if i != 10 {
		return str
	}
	return string(limited[:])
}
