package coolq

import (
	"bufio"
	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"
	"math"
	"strconv"
	"strings"
)

type CQCodeConverter struct {
	message []rune
	bot     *CQBot
	group   bool

	index    int
	stat     state
	elements []message.IMessageElement
	tempText []rune
	cqCode   []rune
}

type state uint8

const (
	S0 state = iota
	S1
)

func newCQCodeConverter(message string, bot *CQBot, group bool) *CQCodeConverter {
	return &CQCodeConverter{message: []rune(message), bot: bot, group: group, stat: S0}
}

func (c *CQCodeConverter) hasNext() bool {
	return c.index < len(c.message)
}

func (c *CQCodeConverter) next() rune {
	r := c.message[c.index]
	c.index++
	return r
}

func (c *CQCodeConverter) move(steps int) {
	c.index += steps
}

func (c *CQCodeConverter) peek(idx int) rune {
	return c.message[idx]
}

func (c *CQCodeConverter) peekN(count int) string {
	lastIdx := int(math.Min(float64(c.index+count), float64(len(c.message)-1)))
	return string(c.message[c.index:lastIdx])
}

func (c *CQCodeConverter) isCQCodeBegin(r rune) bool {
	return r == '[' && c.peekN(3) == "CQ:"
}

func (c *CQCodeConverter) saveTempText() {
	if len(c.tempText) != 0 {
		c.elements = append(c.elements, message.NewText(CQCodeUnescapeValue(string(c.tempText))))
	}
	c.tempText = []rune{}
	c.cqCode = []rune{}
}

func (c *CQCodeConverter) saveCQCode() {
	defer func() {
		c.cqCode = []rune{}
		c.tempText = []rune{}
	}()
	reader := strings.NewReader(string(c.cqCode))
	buf := bufio.NewReader(reader)
	t, _ := buf.ReadString(',')
	t = t[0 : len(t)-1]
	params := make(map[string]string)
	for buf.Buffered() > 0 {
		p, _ := buf.ReadString(',')
		if strings.HasSuffix(p, ",") {
			p = p[0 : len(p)-1]
		}
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		data := strings.SplitN(p, "=", 2)
		if len(data) == 2 {
			params[data[0]] = data[1]
		} else {
			params[p] = ""
		}
	}
	if t == "reply" { // reply 特殊处理
		if len(c.elements) > 0 {
			if _, ok := c.elements[0].(*message.ReplyElement); ok {
				log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
				return
			}
		}
		mid, err := strconv.Atoi(params["id"])
		if err == nil {
			org := c.bot.GetMessage(int32(mid))
			if org != nil {
				c.elements = append([]message.IMessageElement{
					&message.ReplyElement{
						ReplySeq: org["message-id"].(int32),
						Sender:   org["sender"].(message.Sender).Uin,
						Time:     org["time"].(int32),
						Elements: c.bot.ConvertStringMessage(org["message"].(string), c.group),
					},
				}, c.elements...)
				return
			}
		}
	}
	elem, err := c.bot.ToElement(t, params, c.group)
	if err != nil {
		org := "[" + string(c.cqCode) + "]"
		if !IgnoreInvalidCQCode {
			log.Warnf("转换CQ码 %v 时出现错误: %v 将原样发送.", org, err)
			c.elements = append(c.elements, message.NewText(org))
		} else {
			log.Warnf("转换CQ码 %v 时出现错误: %v 将忽略.", org, err)
		}
		return
	}
	switch i := elem.(type) {
	case message.IMessageElement:
		c.elements = append(c.elements, i)
	case []message.IMessageElement:
		c.elements = append(c.elements, i...)
	}
}

func (c *CQCodeConverter) convert() []message.IMessageElement {
	for c.hasNext() {
		ch := c.next()
		switch c.stat {
		case S0:
			if c.isCQCodeBegin(ch) {
				c.saveTempText()
				c.tempText = append(c.tempText, []rune("[CQ:")...)
				c.move(3)
				c.stat = S1
			} else {
				c.tempText = append(c.tempText, ch)
			}
		case S1:
			if c.isCQCodeBegin(ch) {
				c.move(-1)
				c.stat = S0
			} else if ch == ']' {
				c.saveCQCode()
				c.stat = S0
			} else {
				c.cqCode = append(c.cqCode, ch)
				c.tempText = append(c.tempText, ch)
			}
		}
	}
	c.saveTempText()
	return c.elements
}
