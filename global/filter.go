package global

import (
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"regexp"
	"strings"
)

type Filter interface {
	Eval(payload gjson.Result) bool
}

type OperationNode struct {
	key    string
	filter Filter
}

type NotOperator struct {
	operand_ Filter
}

func notOperatorConstruct(argument gjson.Result) *NotOperator {
	if !argument.IsObject() {
		panic("the argument of 'not' operator must be an object")
	}
	op := new(NotOperator)
	op.operand_ = Generate("and", argument)
	return op
}

func (notOperator NotOperator) Eval(payload gjson.Result) bool {
	return !(notOperator.operand_).Eval(payload)
}

type AndOperator struct {
	operands []OperationNode
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
			op.operands = append(op.operands, OperationNode{"", Generate(opKey, value)})
		} else if value.IsObject() {
			// is an normal key with an object as the value
			//   "foo": {
			//       ".bar": "baz"
			//   }
			opKey := key.String()
			op.operands = append(op.operands, OperationNode{opKey, Generate("and", value)})
		} else {
			// is an normal key with a non-object as the value
			//   "foo": "bar"
			opKey := key.String()
			op.operands = append(op.operands, OperationNode{opKey, Generate("eq", value)})
		}
		return true
	})
	return op
}

func (andOperator *AndOperator) Eval(payload gjson.Result) bool {
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

		if res == false {
			break
		}
	}
	return res
}

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

func (orOperator OrOperator) Eval(payload gjson.Result) bool {
	res := false
	for _, operand := range orOperator.operands {
		res = res || operand.Eval(payload)

		if res == true {
			break
		}
	}
	return res
}

type EqualOperator struct {
	value gjson.Result
}

func equalOperatorConstruct(argument gjson.Result) *EqualOperator {
	op := new(EqualOperator)
	op.value = argument
	return op
}

func (equalOperator EqualOperator) Eval(payload gjson.Result) bool {
	return payload.String() == equalOperator.value.String()
}

type NotEqualOperator struct {
	value gjson.Result
}

func notEqualOperatorConstruct(argument gjson.Result) *NotEqualOperator {
	op := new(NotEqualOperator)
	op.value = argument
	return op
}

func (notEqualOperator NotEqualOperator) Eval(payload gjson.Result) bool {
	return !(payload.String() == notEqualOperator.value.String())
}

type InOperator struct {
	operand gjson.Result
}

func inOperatorConstruct(argument gjson.Result) *InOperator {
	if argument.IsObject() {
		panic("the argument of 'in' operator must be an array or a string")
	}
	op := new(InOperator)
	op.operand = argument
	return op
}

func (inOperator InOperator) Eval(payload gjson.Result) bool {
	if inOperator.operand.IsArray() {
		res := false
		inOperator.operand.ForEach(func(key, value gjson.Result) bool {
			res = res || value.String() == payload.String()
			return true
		})
		return res
	}
	return strings.Contains(inOperator.operand.String(), payload.String())
}

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

func (containsOperator ContainsOperator) Eval(payload gjson.Result) bool {
	if payload.IsObject() || payload.IsArray() {
		return false
	}
	return strings.Contains(payload.String(), containsOperator.operand)
}

type RegexOperator struct {
	regex string
}

func regexOperatorConstruct(argument gjson.Result) *RegexOperator {
	if argument.IsArray() || argument.IsObject() {
		panic("the argument of 'regex' operator must be a string")
	}
	op := new(RegexOperator)
	op.regex = argument.String()
	return op
}

func (containsOperator RegexOperator) Eval(payload gjson.Result) bool {
	matched, _ := regexp.MatchString(containsOperator.regex, payload.String())
	return matched
}

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

var EventFilter = new(Filter)

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
		*EventFilter = Generate("and", gjson.ParseBytes(f))
	}
}
