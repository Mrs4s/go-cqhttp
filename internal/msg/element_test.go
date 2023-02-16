package msg

import (
	"encoding/json"
	"testing"
)

func jsonMarshal(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestQuoteJSON(t *testing.T) {
	testcase := []string{
		"\u0005", // issue 1773
		"\v",
	}

	for _, input := range testcase {
		got := QuoteJSON(input)
		expected := jsonMarshal(input)
		if got != expected {
			t.Errorf("want %v but got %v", expected, got)
		}
	}
}
