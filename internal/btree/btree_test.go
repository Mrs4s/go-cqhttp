package btree

import (
	"crypto/sha1"
	"os"
	"testing"

	"github.com/Mrs4s/MiraiGo/utils"
	assert2 "github.com/stretchr/testify/assert"
)

func tempfile(t *testing.T) string {
	temp, err := os.CreateTemp(".", "temp.*.db")
	assert2.NoError(t, temp.Close())
	assert2.NoError(t, err)
	return temp.Name()
}

func TestCreate(t *testing.T) {
	f := tempfile(t)
	_, err := Create(f)
	assert2.NoError(t, err)
	defer os.Remove(f)
}

func TestBtree(t *testing.T) {
	f := tempfile(t)
	defer os.Remove(f)
	bt, err := Create(f)
	assert := assert2.New(t)
	assert.NoError(err)

	tests := []string{
		"hello world",
		"123",
		"We are met on a great battle-field of that war.",
		"Abraham Lincoln, November 19, 1863, Gettysburg, Pennsylvania",
	}
	sha := make([]*byte, len(tests))
	for i, tt := range tests {
		hash := sha1.New()
		hash.Write([]byte(tt))
		sha[i] = &hash.Sum(nil)[0]
		bt.Insert(sha[i], []byte(tt))
	}
	assert.NoError(bt.Close())

	bt, err = Open(f)
	assert.NoError(err)
	var ss []string
	bt.Foreach(func(key [16]byte, value []byte) {
		ss = append(ss, string(value))
	})
	assert.ElementsMatch(tests, ss)

	for i, tt := range tests {
		assert.Equal([]byte(tt), bt.Get(sha[i]))
	}

	for i := range tests {
		assert.NoError(bt.Delete(sha[i]))
	}

	for i := range tests {
		assert.Equal([]byte(nil), bt.Get(sha[i]))
	}
	assert.NoError(bt.Close())
}

func testForeach(t *testing.T, elemSize int) {
	expected := make([]string, elemSize)
	for i := 0; i < elemSize; i++ {
		expected[i] = utils.RandomString(20)
	}
	f := tempfile(t)
	defer os.Remove(f)
	bt, err := Create(f)
	assert2.NoError(t, err)
	for _, v := range expected {
		hash := sha1.New()
		hash.Write([]byte(v))
		bt.Insert(&hash.Sum(nil)[0], []byte(v))
	}
	var got []string
	bt.Foreach(func(key [16]byte, value []byte) {
		got = append(got, string(value))
	})
	assert2.ElementsMatch(t, expected, got)
	assert2.NoError(t, bt.Close())
}

func TestDB_Foreach(t *testing.T) {
	elemSizes := []int{0, 5, 100, 200}
	for _, size := range elemSizes {
		testForeach(t, size)
	}
}
