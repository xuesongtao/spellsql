package spellsql

import (
	"context"
	"testing"
)

func TestSearchAfter(t *testing.T) {
	obj := &SearchAfter{
		SqlStr:   "select id,name from man",
		Table:    "man",
		Names:    []string{"id"},
		Values:   []interface{}{0},
		OrderBys: []string{},
		Size:     0,
		Dest:     &ManCopy{},
	}
	// 求总数
	total := 0
	obj.RowFn = func(_row interface{}) error {
		v := _row.(*ManCopy)
		total++
		obj.Values[0] = v.Id
		return nil
	}
	err := obj.Search(context.TODO(), db)
	if err != nil {
		t.Fatal(err)
	}

	var totalDst int32
	_ = Count(db, "man", &totalDst, "1")
	if totalDst != int32(total) {
		t.Error("it is no ok")
	}
	t.Logf("total: %d, totalDst: %d", total, totalDst)
}
