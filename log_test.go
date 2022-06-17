package spellsql

import "testing"

func TestCallFile(t *testing.T) {
	l := NewCjLogger()
	t.Log(l.callInfo(1))
	t.Log(l.getPrefix(2))
}

func TestDemo(t *testing.T) {
	cjLog.Info("hello info")
	cjLog.Infof("hello infof: %v", 1)

	cjLog.Warning("hello warning")
	cjLog.Warningf("hello warningf: %v", 2)

	cjLog.Error("hello error")
	cjLog.Errorf("hello errorf: %v", 3)
}

func TestFatal(t *testing.T) {
	t.Skip()
	l := NewCjLogger()
	// cjLog.Fatal("hello fatal")
	l.Fatalf("hello fatalf: %v", 1)
}
