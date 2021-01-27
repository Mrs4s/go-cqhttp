package global

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

//MSG 消息Map
type MSG map[string]interface{}

//Get 尝试从消息Map中取出key为s的值,若不存在则返回MSG{}
//
//若所给key对应的值的类型是global.MSG,则返回此值
//
//若所给key对应值的类型不是global.MSG,则返回MSG{"__str__": Val}
func (m MSG) Get(s string) MSG {
	if v, ok := m[s]; ok {
		if msg, ok := v.(MSG); ok {
			return msg
		}
		return MSG{"__str__": v} // 用这个名字应该没问题吧
	}
	return nil // 不存在为空
}

//String 将消息Map转化为String。若Map存在key "__str__",则返回此key对应的值,否则将输出整张消息Map对应的JSON字符串
func (m MSG) String() string {
	if m == nil {
		return "" // 空 JSON
	}
	if str, ok := m["__str__"]; ok {
		if str == nil {
			return "" // 空 JSON
		}
		return fmt.Sprint(str)
	}
	str, _ := json.MarshalToString(m)
	return str
}

//Filter 定义了一个消息上报过滤接口
type Filter interface {
	Eval(payload MSG) bool
}

type operationNode struct {
	key    string
	filter Filter
}

//NotOperator 定义了过滤器中Not操作符
type NotOperator struct {
	operand Filter
}

func notOperatorConstruct(argument gjson.Result) *NotOperator {
	if !argument.IsObject() {
		panic("the argument of 'not' operator must be an object")
	}
	op := new(NotOperator)
	op.operand = Generate("and", argument)
	return op
}

//Eval 对payload执行Not过滤
func (op *NotOperator) Eval(payload MSG) bool {
	return !op.operand.Eval(payload)
}

//AndOperator 定义了过滤器中And操作符
type AndOperator struct {
	operands []operationNode
}

func andOperatorConstruct(argument gjson.Result) *AndOperator {
	if !argument.IsObject() {
		panic("the argument of 'and' operator must be an object")
	}
	op := new(AndOperator)
	argument.ForEach(func(key, value gjson.Result) bool {
		if key.Str[0] == '.' {
			// is an operator
			//   ".foo": {
			//       "bar": "baz"
			//   }
			opKey := key.Str[1:]
			op.operands = append(op.operands, operationNode{"", Generate(opKey, value)})
		} else if value.IsObject() {
			// is an normal key with an object as the value
			//   "foo": {
			//       ".bar": "baz"
			//   }
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("and", value)})
		} else {
			// is an normal key with a non-object as the value
			//   "foo": "bar"
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("eq", value)})
		}
		return true
	})
	return op
}

//Eval 对payload执行And过滤
func (andOperator *AndOperator) Eval(payload MSG) bool {
	res := true
	for _, operand := range andOperator.operands {

		if len(operand.key) == 0 {
			// is an operator
			res = res && operand.filter.Eval(payload)
		} else {
			// is an normal key
			val := payload.Get(operand.key)
			res = res && operand.filter.Eval(val)
		}

		if !res {
			break
		}
	}
	return res
}

//OrOperator 定义了过滤器中Or操作符
type OrOperator struct {
	operands []Filter
}

func orOperatorConstruct(argument gjson.Result) *OrOperator {
	if !argument.IsArray() {
		panic("the argument of 'or' operator must be an array")
	}
	op := new(OrOperator)
	argument.ForEach(func(_, value gjson.Result) bool {
		op.operands = append(op.operands, Generate("and", value))
		return true
	})
	return op
}

//Eval 对payload执行Or过滤
func (op *OrOperator) Eval(payload MSG) bool {
	res := false
	for _, operand := range op.operands {
		res = res || operand.Eval(payload)
		if res {
			break
		}
	}
	return res
}

//EqualOperator 定义了过滤器中Equal操作符
type EqualOperator struct {
	operand string
}

func equalOperatorConstruct(argument gjson.Result) *EqualOperator {
	op := new(EqualOperator)
	op.operand = argument.String()
	return op
}

//Eval 对payload执行Equal过滤
func (op *EqualOperator) Eval(payload MSG) bool {
	return payload.String() == op.operand
}

//NotEqualOperator 定义了过滤器中NotEqual操作符
type NotEqualOperator struct {
	operand string
}

func notEqualOperatorConstruct(argument gjson.Result) *NotEqualOperator {
	op := new(NotEqualOperator)
	op.operand = argument.String()
	return op
}

//Eval 对payload执行NotEqual过滤
func (op *NotEqualOperator) Eval(payload MSG) bool {
	return !(payload.String() == op.operand)
}

//InOperator 定义了过滤器中In操作符
type InOperator struct {
	operandString string
	operandArray  []string
}

func inOperatorConstruct(argument gjson.Result) *InOperator {
	if argument.IsObject() {
		panic("the argument of 'in' operator must be an array or a string")
	}
	op := new(InOperator)
	if argument.IsArray() {
		op.operandArray = []string{}
		argument.ForEach(func(_, value gjson.Result) bool {
			op.operandArray = append(op.operandArray, value.String())
			return true
		})
	} else {
		op.operandString = argument.String()
	}
	return op
}

//Eval 对payload执行In过滤
func (op *InOperator) Eval(payload MSG) bool {
	payloadStr := payload.String()
	if op.operandArray != nil {
		for _, value := range op.operandArray {
			if value == payloadStr {
				return true
			}
		}
		return false
	}
	return strings.Contains(op.operandString, payloadStr)
}

//ContainsOperator 定义了过滤器中Contains操作符
type ContainsOperator struct {
	operand string
}

func containsOperatorConstruct(argument gjson.Result) *ContainsOperator {
	if argument.IsArray() || argument.IsObject() {
		panic("the argument of 'contains' operator must be a string")
	}
	op := new(ContainsOperator)
	op.operand = argument.String()
	return op
}

//Eval 对payload执行Contains过滤
func (op *ContainsOperator) Eval(payload MSG) bool {
	return strings.Contains(payload.String(), op.operand)
}

//RegexOperator 定义了过滤器中Regex操作符
type RegexOperator struct {
	regex *regexp.Regexp
}

func regexOperatorConstruct(argument gjson.Result) *RegexOperator {
	if argument.IsArray() || argument.IsObject() {
		panic("the argument of 'regex' operator must be a string")
	}
	op := new(RegexOperator)
	op.regex = regexp.MustCompile(argument.String())
	return op
}

//Eval 对payload执行RegexO过滤
func (op *RegexOperator) Eval(payload MSG) bool {
	matched := op.regex.MatchString(payload.String())
	return matched
}

//Generate 根据给定操作符名opName及操作符参数argument创建一个过滤器实例
func Generate(opName string, argument gjson.Result) Filter {
	switch opName {
	case "not":
		return notOperatorConstruct(argument)
	case "and":
		return andOperatorConstruct(argument)
	case "or":
		return orOperatorConstruct(argument)
	case "neq":
		return notEqualOperatorConstruct(argument)
	case "eq":
		return equalOperatorConstruct(argument)
	case "in":
		return inOperatorConstruct(argument)
	case "contains":
		return containsOperatorConstruct(argument)
	case "regex":
		return regexOperatorConstruct(argument)
	default:
		panic("the operator " + opName + " is not supported")
	}
}

//EventFilter 初始化一个nil过滤器
var EventFilter Filter = nil

//BootFilter 启动事件过滤器
func BootFilter() {
	defer func() {
		if e := recover(); e != nil {
			log.Warnf("事件过滤器启动失败: %v", e)
			EventFilter = nil
		} else {
			log.Info("事件过滤器启动成功.")
		}
	}()
	f, err := ioutil.ReadFile("filter.json")
	if err != nil {
		panic(err)
	} else {
		EventFilter = Generate("and", gjson.ParseBytes(f))
	}
}
