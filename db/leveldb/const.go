package leveldb

const dataVersion = 1

const (
	group        = 0x0
	private      = 0x1
	guildChannel = 0x2
)

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
