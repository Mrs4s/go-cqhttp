// Package onebot defines onebot protocol struct and some spec info.
package onebot

//go:generate go run github.com/Mrs4s/go-cqhttp/cmd/api-generator -pkg onebot -path=./../../coolq/api.go,./../../coolq/api_v12.go -supported -o supported.go

// Spec OneBot Specification
type Spec struct {
	Version          int // must be 11 or 12
	SupportedActions []string
}

/* // TODO: Use this variable
var V11 = &Spec{
	Version:          11,
	SupportedActions: supportedV11,
}
*/

// V12 OneBot V12
var V12 = &Spec{
	Version:          12,
	SupportedActions: supportedV12,
}
