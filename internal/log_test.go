package internal

import (
	"testing"
)

func TestCallFile(t *testing.T) {
	l := NewLogger()
	t.Log(l.callInfo(1))
	t.Log(l.getPrefix(2))
}
