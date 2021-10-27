// Package filter implements an event filter for go-cqhttp
package filter

import (
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// Filter 定义了一个消息上报过滤接口
type Filter interface {
	Eval(payload gjson.Result) bool
}

type operationNode struct {
	key    string
	filter Filter
}

// notOperator 定义了过滤器中Not操作符
type notOperator struct {
	operand Filter
}

func newNotOp(argument gjson.Result) Filter {
	if !argument.IsObject() {
		panic("the argument of 'not' operator must be an object")
	}
	return &notOperator{operand: Generate("and", argument)}
}

// Eval 对payload执行Not过滤
func (op *notOperator) Eval(payload gjson.Result) bool {
	return !op.operand.Eval(payload)
}

// andOperator 定义了过滤器中And操作符
type andOperator struct {
	operands []operationNode
}

func newAndOp(argument gjson.Result) Filter {
	if !argument.IsObject() {
		panic("the argument of 'and' operator must be an object")
	}
	op := new(andOperator)
	argument.ForEach(func(key, value gjson.Result) bool {
		switch {
		case key.Str[0] == '.':
			// is an operator
			//   ".foo": {
			//       "bar": "baz"
			//   }
			opKey := key.Str[1:]
			op.operands = append(op.operands, operationNode{"", Generate(opKey, value)})
		case value.IsObject():
			// is a normal key with an object as the value
			//   "foo": {
			//       ".bar": "baz"
			//   }
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("and", value)})
		default:
			// is a normal key with a non-object as the value
			//   "foo": "bar"
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("eq", value)})
		}
		return true
	})
	return op
}

// Eval 对payload执行And过滤
func (op *andOperator) Eval(payload gjson.Result) bool {
	res := true
	for _, operand := range op.operands {
		if len(operand.key) == 0 {
			// is an operator
			res = res && operand.filter.Eval(payload)
		} else {
			// is a normal key
			val := payload.Get(operand.key)
			res = res && operand.filter.Eval(val)
		}

		if !res {
			break
		}
	}
	return res
}

// orOperator 定义了过滤器中Or操作符
type orOperator struct {
	operands []Filter
}

func newOrOp(argument gjson.Result) Filter {
	if !argument.IsArray() {
		panic("the argument of 'or' operator must be an array")
	}
	op := new(orOperator)
	argument.ForEach(func(_, value gjson.Result) bool {
		op.operands = append(op.operands, Generate("and", value))
		return true
	})
	return op
}

// Eval 对payload执行Or过滤
func (op *orOperator) Eval(payload gjson.Result) bool {
	res := false
	for _, operand := range op.operands {
		res = res || operand.Eval(payload)
		if res {
			break
		}
	}
	return res
}

// eqOperator 定义了过滤器中Equal操作符
type eqOperator struct {
	operand string
}

func newEqOp(argument gjson.Result) Filter {
	return &eqOperator{operand: argument.String()}
}

// Eval 对payload执行Equal过滤
func (op *eqOperator) Eval(payload gjson.Result) bool {
	return payload.String() == op.operand
}

// neqOperator 定义了过滤器中NotEqual操作符
type neqOperator struct {
	operand string
}

func newNeqOp(argument gjson.Result) Filter {
	return &neqOperator{operand: argument.String()}
}

// Eval 对payload执行NotEqual过滤
func (op *neqOperator) Eval(payload gjson.Result) bool {
	return !(payload.String() == op.operand)
}

// inOperator 定义了过滤器中In操作符
type inOperator struct {
	operandString string
	operandArray  []string
}

func newInOp(argument gjson.Result) Filter {
	if argument.IsObject() {
		panic("the argument of 'in' operator must be an array or a string")
	}
	op := new(inOperator)
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

// Eval 对payload执行In过滤
func (op *inOperator) Eval(payload gjson.Result) bool {
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

// containsOperator 定义了过滤器中Contains操作符
type containsOperator struct {
	operand string
}

func newContainOp(argument gjson.Result) Filter {
	if argument.IsArray() || argument.IsObject() {
		panic("the argument of 'contains' operator must be a string")
	}
	return &containsOperator{operand: argument.String()}
}

// Eval 对payload执行Contains过滤
func (op *containsOperator) Eval(payload gjson.Result) bool {
	return strings.Contains(payload.String(), op.operand)
}

// regexOperator 定义了过滤器中Regex操作符
type regexOperator struct {
	regex *regexp.Regexp
}

func newRegexOp(argument gjson.Result) Filter {
	if argument.IsArray() || argument.IsObject() {
		panic("the argument of 'regex' operator must be a string")
	}
	return &regexOperator{regex: regexp.MustCompile(argument.String())}
}

// Eval 对payload执行RegexO过滤
func (op *regexOperator) Eval(payload gjson.Result) bool {
	return op.regex.MatchString(payload.String())
}

// Generate 根据给定操作符名opName及操作符参数argument创建一个过滤器实例
func Generate(opName string, argument gjson.Result) Filter {
	switch opName {
	case "not":
		return newNotOp(argument)
	case "and":
		return newAndOp(argument)
	case "or":
		return newOrOp(argument)
	case "eq":
		return newEqOp(argument)
	case "neq":
		return newNeqOp(argument)
	case "in":
		return newInOp(argument)
	case "contains":
		return newContainOp(argument)
	case "regex":
		return newRegexOp(argument)
	default:
		panic("the operator " + opName + " is not supported")
	}
}
