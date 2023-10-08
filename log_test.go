package spellsql

import (
	"context"
	"testing"
)

func TestCallFile(t *testing.T) {
	l := NewLogger()
	t.Log(l.callInfo(1))
	t.Log(l.getPrefix(2))
}

func TestDemo(t *testing.T) {
	sLog.Info(context.Background(), "hello info")

	sLog.Warning(context.Background(), "hello warning")

	sLog.Error(context.Background(), "hello error")
}
