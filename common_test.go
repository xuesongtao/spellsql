package spellsql

import "testing"

func TestIsExported(t *testing.T) {
	t.Log(isExported("name"))
	t.Log(isExported("Name"))
}
