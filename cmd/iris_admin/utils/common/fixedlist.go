// Package common 通用工具包
package common

import (
	"container/list"
	"sync"
)

// FixedList 定长列表。初始化时定义好长度，add()负责向尾部添加数据,当数据达到指定长度时，fixedlist会自动删除头部数据。
type FixedList interface {
	Add(interface{})
	Len() int
	Data() []interface{}
}

type fixedList struct {
	sync.RWMutex
	length int
	data   *list.List
}

// NewFixedList 创建定长列表
func NewFixedList(lenth int) FixedList {
	f := &fixedList{}
	f.length = lenth
	f.data = list.New()
	return f
}

// Add 添加一条记录
func (f *fixedList) Add(val interface{}) {
	f.Lock()
	defer f.Unlock()
	f.data.PushBack(val)
	if f.data.Len() > f.length {
		for i := 0; i <= f.data.Len()-f.length; i++ {
			f.data.Remove(f.data.Front())
		}
	}
}

// Len 获取数据长度
func (f *fixedList) Len() int {
	return f.data.Len()
}

// Data 获取数据
func (f *fixedList) Data() []interface{} {
	f.RLock()
	defer f.RUnlock()
	var data []interface{}
	for i := f.data.Front(); i != nil; i = i.Next() {
		data = append(data, i.Value)
	}
	return data
}
