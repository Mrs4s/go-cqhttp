package coolq

import (
	"fmt"
	"testing"

	"github.com/tidwall/gjson"
)

var bot = CQBot{}

func TestCQBot_ConvertStringMessage(t *testing.T) {
	for _, v := range bot.ConvertStringMessage(`[CQ:face,id=115,text=111][CQ:face,id=217]] [CQ:text,text=123] [`, false) {
		fmt.Println(v)
	}
}

var bench = `asdfqwerqwerqwer[CQ:face,id=115,text=111]asdfasdfasdfasdfasdfasdfasd[CQ:face,id=217]&#93; 123 &#91;`
var benchArray = gjson.Parse(`[{"type":"text","data":{"text":"asdfqwerqwerqwer"}},{"type":"face","data":{"id":"115","text":"111"}},{"type":"text","data":{"text":"asdfasdfasdfasdfasdfasdfasd"}},{"type":"face","data":{"id":"217"}},{"type":"text","data":{"text":"] "}},{"type":"text","data":{"text":"123"}},{"type":"text","data":{"text":" ["}}]`)

func BenchmarkCQBot_ConvertStringMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bot.ConvertStringMessage(bench, false)
	}
	b.SetBytes(int64(len(bench)))
}

func BenchmarkCQBot_ConvertObjectMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bot.ConvertObjectMessage(benchArray, false)
	}
}
