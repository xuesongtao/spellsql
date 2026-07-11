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

func CallOnce(f func()) func() {
	var once sync.Once
	return func() {
		once.Do(f)
	}
}

func Equal(a, b uint8) bool {
	return a == b
}
