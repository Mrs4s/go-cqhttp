package global

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

// NotOperator 定义了过滤器中Not操作符
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

// Eval 对payload执行Not过滤
func (op *NotOperator) Eval(payload gjson.Result) bool {
	return !op.operand.Eval(payload)
}

// AndOperator 定义了过滤器中And操作符
type AndOperator struct {
	operands []operationNode
}

func andOperatorConstruct(argument gjson.Result) *AndOperator {
	if !argument.IsObject() {
		panic("the argument of 'and' operator must be an object")
	}
	op := new(AndOperator)
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
			// is an normal key with an object as the value
			//   "foo": {
			//       ".bar": "baz"
			//   }
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("and", value)})
		default:
			// is an normal key with a non-object as the value
			//   "foo": "bar"
			opKey := key.String()
			op.operands = append(op.operands, operationNode{opKey, Generate("eq", value)})
		}
		return true
	})
	return op
}

// Eval 对payload执行And过滤
func (op *AndOperator) Eval(payload gjson.Result) bool {
	res := true
	for _, operand := range op.operands {
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

// OrOperator 定义了过滤器中Or操作符
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

// Eval 对payload执行Or过滤
func (op *OrOperator) Eval(payload gjson.Result) bool {
	res := false
	for _, operand := range op.operands {
		res = res || operand.Eval(payload)
		if res {
			break
		}
	}
	return res
}

// EqualOperator 定义了过滤器中Equal操作符
type EqualOperator struct {
	operand string
}

func equalOperatorConstruct(argument gjson.Result) *EqualOperator {
	op := new(EqualOperator)
	op.operand = argument.String()
	return op
}

// Eval 对payload执行Equal过滤
func (op *EqualOperator) Eval(payload gjson.Result) bool {
	return payload.String() == op.operand
}

// NotEqualOperator 定义了过滤器中NotEqual操作符
type NotEqualOperator struct {
	operand string
}

func notEqualOperatorConstruct(argument gjson.Result) *NotEqualOperator {
	op := new(NotEqualOperator)
	op.operand = argument.String()
	return op
}

// Eval 对payload执行NotEqual过滤
func (op *NotEqualOperator) Eval(payload gjson.Result) bool {
	return !(payload.String() == op.operand)
}

// InOperator 定义了过滤器中In操作符
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

// Eval 对payload执行In过滤
func (op *InOperator) Eval(payload gjson.Result) bool {
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

// ContainsOperator 定义了过滤器中Contains操作符
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

// Eval 对payload执行Contains过滤
func (op *ContainsOperator) Eval(payload gjson.Result) bool {
	return strings.Contains(payload.String(), op.operand)
}

// RegexOperator 定义了过滤器中Regex操作符
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

// Eval 对payload执行RegexO过滤
func (op *RegexOperator) Eval(payload gjson.Result) bool {
	matched := op.regex.MatchString(payload.String())
	return matched
}

// Generate 根据给定操作符名opName及操作符参数argument创建一个过滤器实例
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
