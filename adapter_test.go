package spellsql

import (
	"encoding/json"
	"testing"

	"gitee.com/xuesongtao/spellsql/test"
	// "gitlab.cd.anpro/go/common/spellsql/test"
)

func TestEscapeBytes(t *testing.T) {
	c := Mysql()
	m := test.Man{
		JsonTxt: test.Tmp{
			Name: "<title>北京欢迎你</title>",
			Data: "\n" + "\t" + "test json marshal",
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	escapeBytes := c.EscapeBytes(b)
	t.Log(len(b), len(escapeBytes))
	t.Log(string(b))
	t.Log(string(escapeBytes))

	var mm test.Man
	if err := json.Unmarshal(escapeBytes, &mm); err != nil {
		t.Fatal(err)
	}
	t.Log(mm)
}
