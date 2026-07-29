package internal

import (
	"strings"
	"sync"
)

var (
	tmpBuf = sync.Pool{New: func() interface{} { return new(strings.Builder) }}
)

func GetTmpBuf(size ...int) *strings.Builder {
	obj := tmpBuf.Get().(*strings.Builder)
	if len(size) > 0 {
		obj.Grow(size[0])
	}
	return obj
}

func PutTmpBuf(obj *strings.Builder) {
	obj.Reset()
	tmpBuf.Put(obj)
}

func Equal(a, b uint8) bool {
	return a == b
}

func InArray(a uint8, arr ...uint8) bool {
	for _, v := range arr {
		if a == v {
			return true
		}
	}
	return false
}

func ToUpper(str string) string {
	strByte := []byte(str)
	l := len(strByte)
	for i := 0; i < l; i++ {
		strByte[i] &= '_'
	}
	return string(strByte)
}

func ToLower(str string) string {
	strByte := []byte(str)
	l := len(strByte)
	for i := 0; i < l; i++ {
		strByte[i] |= ' '
	}
	return string(strByte)
}
