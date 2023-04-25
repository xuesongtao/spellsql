package spellsql

import (
	"encoding/json"
	"testing"

	"gitee.com/xuesongtao/spellsql/test"
)

func TestCommonTable(t *testing.T) {
	c := &CommonTable{}
	m := test.Man{
		JsonTxt: test.Tmp{
			Name: "json",
			Data: "\n" + "\t" + "test json marshal",
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	escapeBytes := c.escapeBytes(b)
	t.Log(len(b), len(escapeBytes))

	var mm test.Man
	if err := json.Unmarshal(escapeBytes, &mm); err != nil {
		t.Fatal(err)
	}
	t.Log(mm)
}
