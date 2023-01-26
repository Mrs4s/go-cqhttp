package cqcode

import (
	"bytes"
	"testing"
)

func TestIssue1733(t *testing.T) {
	const (
		input    = "\u0005"
		expected = `"\u0005"`
	)
	var b bytes.Buffer
	writeQuote(&b, input)
	got := b.String()
	if got != expected {
		t.Errorf("want %v but got %v", expected, got)
	}
}
