package config

import (
	"strings"
	"testing"
)

func Test_expand(t *testing.T) {
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
	}
	for i, tt := range tests {
		if got := expand(tt.src, tt.mapping); got != tt.expected {
			t.Errorf("testcase %d failed, expected %v but got %v", i, tt.expected, got)
		}
	}
}
