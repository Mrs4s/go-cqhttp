package coolq

import (
	"fmt"
	"testing"
)

var bot = CQBot{}

func TestCQBot_ConvertStringMessage(t *testing.T) {
	for _, v := range bot.ConvertStringMessage(`[CQ:face,id=115,text=111][CQ:face,id=217]] [CQ:text,text=123] [`, false) {
		fmt.Println(v)
	}
}

var bench = `asdfqwerqwerqwer[CQ:face,id=115,text=111]asdfasdfasdfasdfasdfasdfasd[CQ:face,id=217]] [CQ:text,text=123] [`

func BenchmarkCQBot_ConvertStringMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bot.ConvertStringMessage(bench, false)
	}
}
