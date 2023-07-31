package msg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseString(_ *testing.T) {
	// TODO: add more text
	for _, v := range ParseString(`[CQ:face,id=115,text=111][CQ:face,id=217]] [CQ:text,text=123] [`) {
		fmt.Println(v)
	}
}

var (
	bench      = `asdfqwerqwerqwer[CQ:face,id=115,text=111]asdfasdfasdfasdfasdfasdfasd[CQ:face,id=217]&#93; 123 &#91;`
	benchArray = gjson.Parse(`[{"type":"text","data":{"text":"asdfqwerqwerqwer"}},{"type":"face","data":{"id":"115","text":"111"}},{"type":"text","data":{"text":"asdfasdfasdfasdfasdfasdfasd"}},{"type":"face","data":{"id":"217"}},{"type":"text","data":{"text":"] "}},{"type":"text","data":{"text":"123"}},{"type":"text","data":{"text":" ["}}]`)
)

func BenchmarkParseString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseString(bench)
	}
	b.SetBytes(int64(len(bench)))
}

func BenchmarkParseObject(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseObject(benchArray)
	}
	b.SetBytes(int64(len(benchArray.Raw)))
}

const bText = `123456789[]&987654321[]&987654321[]&987654321[]&987654321[]&987654321[]&`

func BenchmarkCQCodeEscapeText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ret := bText
		EscapeText(ret)
	}
}

func TestCQCodeEscapeText(t *testing.T) {
	for i := 0; i < 200; i++ {
		rs := utils.RandomStringRange(3000, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890[]&")
		ret := rs
		ret = strings.ReplaceAll(ret, "&", "&amp;")
		ret = strings.ReplaceAll(ret, "[", "&#91;")
		ret = strings.ReplaceAll(ret, "]", "&#93;")
		assert.Equal(t, ret, EscapeText(rs))
	}
}
