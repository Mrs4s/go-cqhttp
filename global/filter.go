package global

import (
	"bytes"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"sync"
)

type Filter interface {
	Eval(payload []byte) bool
}

type OperationNode struct {
	key string
	filter Filter
}

type NotOperator struct {
	operand_ Filter
}

func notOperatorConstruct(argument []byte) *NotOperator {
	op := new(NotOperator)
	op.operand_ = GetOperatorFactory().Generate("and", argument)
	return op
}

func (notOperator NotOperator) Eval(payload []byte) bool {
	log.Debug("not "+string(payload))
	return !(notOperator.operand_).Eval(payload)
}

type AndOperator struct {
	operands []OperationNode
}

func andOperatorConstruct(argument []byte) *AndOperator {
	op := new(AndOperator)
	_ = jsonparser.ObjectEach(argument, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if key[0] == '.' {
			// is an operator
			//   ".foo": {
			//       "bar": "baz"
			//   }
			opKey := string(key[1:])
			op.operands = append(op.operands, OperationNode{"", GetOperatorFactory().Generate(opKey, value)})
		} else if value[0] == '{' {
			// is an normal key with an object as the value
			//   "foo": {
			//       ".bar": "baz"
			//   }
			opKey := string(key)
			op.operands = append(op.operands, OperationNode{opKey, GetOperatorFactory().Generate("and", value)})
		} else {
			// is an normal key with a non-object as the value
			//   "foo": "bar"
			opKey := string(key)
			op.operands = append(op.operands, OperationNode{opKey, GetOperatorFactory().Generate("eq", value)})
		}
		return nil
	})
	return op
}

func (andOperator *AndOperator) Eval(payload []byte) bool {
	log.Debug("and "+string(payload))
	res := true
	nodesLength := len(andOperator.operands)
	for i := 0; i < nodesLength ; i++ {

		if len(andOperator.operands[i].key) == 0 {
			// is an operator
			res = res && andOperator.operands[i].filter.Eval(payload)
		} else {
			// is an normal key
			val, _, _, _ := jsonparser.Get(payload, andOperator.operands[i].key)
			res = res && andOperator.operands[i].filter.Eval(val)
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

func orOperatorConstruct(argument []byte) *OrOperator {
	op := new(OrOperator)
	_, _ = jsonparser.ArrayEach(argument, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		op.operands = append(op.operands, GetOperatorFactory().Generate("and", value))
	})
	return op
}

func (orOperator OrOperator) Eval(payload []byte) bool {
	log.Debug("or "+string(payload))
	res:= false
	nodesLength := len(orOperator.operands)
	for i := 0; i < nodesLength ; i++ {
		res = res || orOperator.operands[i].Eval(payload)

		if res == true {
			break
		}
	}
	return res
}

type EqualOperator struct {
	value []byte
}

func equalOperatorConstruct(argument []byte) *EqualOperator {
	op := new(EqualOperator)
	op.value = argument
	return op
}

func (equalOperator EqualOperator) Eval(payload []byte) bool {
	log.Debug("eq "+string(payload))
	return bytes.Equal(payload, equalOperator.value)
}

type NotEqualOperator struct {
	value []byte
}

func notEqualOperatorConstruct(argument []byte) *NotEqualOperator {
	op := new(NotEqualOperator)
	op.value = argument
	return op
}

func (notEqualOperator NotEqualOperator) Eval(payload []byte) bool {
	log.Debug("neq "+string(payload))
	return !bytes.Equal(payload, notEqualOperator.value)
}


type InOperator struct {
	operands [][]byte
}

func inOperatorConstruct(argument []byte) *InOperator {
	op := new(InOperator)
	_, _ = jsonparser.ArrayEach(argument, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		op.operands = append(op.operands, value)
	})
	return op
}

func (inOperator InOperator) Eval(payload []byte) bool {
	log.Debug("in "+string(payload))
	res := false
	for _, v := range inOperator.operands {
		res = res || bytes.Equal(payload, v)
		if res == true {
			break
		}
	}
	return res
}

type ContainsOperator struct {
	operand []byte
}

func containsOperatorConstruct(argument []byte) *ContainsOperator {
	op := new(ContainsOperator)
	op.operand = argument
	return op
}

func (containsOperator ContainsOperator) Eval(payload []byte) bool {
	log.Debug("contains "+string(payload))
	return bytes.Contains(payload, containsOperator.operand)
}

type RegexOperator struct {
	regex string
}

func regexOperatorConstruct(argument []byte) *RegexOperator {
	op := new(RegexOperator)
	op.regex = string(argument)
	return op
}

func (containsOperator RegexOperator) Eval(payload []byte) bool {
	log.Debug("regex "+string(payload))
	matched, _ := regexp.Match(containsOperator.regex, payload)
	return matched
}
// 单例工厂
type operatorFactory struct{
}

var instance *operatorFactory = &operatorFactory{}

func GetOperatorFactory() *operatorFactory {
	return instance
}

func (o operatorFactory) Generate(opName string, argument []byte) Filter {
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
		log.Warnf("the operator '%s' is not supported", opName)
		return nil
	}
}

var filter = new(Filter)
var once sync.Once   // 过滤器单例模式

func GetFilter() *Filter {
	once.Do(func() {
		f, err := ioutil.ReadFile("filter.json")
		if err != nil {
			filter = nil
		} else {
			*filter = GetOperatorFactory().Generate("and", f)
		}
	})
	return filter
}