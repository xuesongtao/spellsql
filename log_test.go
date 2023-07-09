package spellsql

import "testing"

func TestCallFile(t *testing.T) {
	l := NewLogger()
	t.Log(l.callInfo(1))
	t.Log(l.getPrefix(2))
}

func TestDemo(t *testing.T) {
	sLog.Info("hello info")
	sLog.Infof("hello infof: %v", 1)

	sLog.Warning("hello warning")
	sLog.Warningf("hello warningf: %v", 2)

	sLog.Error("hello error")
	sLog.Errorf("hello errorf: %v", 3)
}

func TestFatal(t *testing.T) {
	t.Skip()
	l := NewLogger()
	// sLog.Fatal("hello fatal")
	l.Fatalf("hello fatalf: %v", 1)
}
