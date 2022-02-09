package config

import (
	"strings"
	"testing"
)

func Test_expand(t *testing.T) {
	nullStringMapping := func(_ string) string {
		return ""
	}
	tests := []struct {
		src      string
		mapping  func(string) string
		expected string
	}{
		{
			src:      "foo: ${bar}",
			mapping:  strings.ToUpper,
			expected: "foo: BAR",
		},
		{
			src:      "$123",
			mapping:  strings.ToUpper,
			expected: "$123",
		},
		{
			src:      "foo: ${bar:123456}",
			mapping:  nullStringMapping,
			expected: "foo: 123456",
		},
		{
			src:      "foo: ${bar:127.0.0.1:5700}",
			mapping:  nullStringMapping,
			expected: "foo: 127.0.0.1:5700",
		},
		{
			src:      "foo: ${bar:ws//localhost:9999/ws}",
			mapping:  nullStringMapping,
			expected: "foo: ws//localhost:9999/ws",
		},
	}
	for i, tt := range tests {
		if got := expand(tt.src, tt.mapping); got != tt.expected {
			t.Errorf("testcase %d failed, expected %v but got %v", i, tt.expected, got)
		}
	}
}
