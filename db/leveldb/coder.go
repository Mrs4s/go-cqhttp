package leveldb

type coder byte

const (
	coderNil coder = iota
	coderInt
	coderUint
	coderInt32
	coderUint32
	coderInt64
	coderUint64
	coderString
	coderMSG      // global.MSG
	coderArrayMSG // []global.MSG
	coderStruct   // struct{}
)
