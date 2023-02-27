// Package onebot defines onebot protocol struct and some spec info.
package onebot

import "fmt"

//go:generate go run ./../../cmd/api-generator -pkg onebot -path=./../../coolq/api.go,./../../coolq/api_v12.go -supported -o supported.go

// Spec OneBot Specification
type Spec struct {
	Version          int // must be 11 or 12
	SupportedActions []string
}

// V11 OneBot V11
var V11 = &Spec{
	Version:          11,
	SupportedActions: supportedV11,
}

// V12 OneBot V12
var V12 = &Spec{
	Version:          12,
	SupportedActions: supportedV12,
}

// ConvertID 根据版本转换ID
func (s *Spec) ConvertID(id any) any {
	if s.Version == 12 {
		return fmt.Sprint(id)
	}
	return id
}
