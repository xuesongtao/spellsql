package spellsql

import (
	"context"
	"fmt"
	"testing"

	"gitee.com/xuesongtao/spellsql/v2/builder"
)

func TestSearchAfter(t *testing.T) {
	for i := 1; i <= 105; i++ {
		InitTestMain(t)
	}
	var totalDst int32
	_ = Count(db, "man", &totalDst, "1")

	obj := &SearchAfter{
		SqlStr:   "select id,name from man",
		Table:    "man",
		Names:    []string{"id"},
		Values:   []interface{}{0},
		OrderBys: []string{},
		Size:     20,
		Dest:     &ManCopy{},
	}
	// 求总数
	total := 0
	obj.RowFn = func(_row interface{}) error {
		total++
		return nil
	}
	err := obj.Search(context.TODO(), db)
	if err != nil {
		t.Fatal(err)
	}

	if totalDst != int32(total) {
		t.Error("it is no ok")
	}
	t.Logf("total: %d, totalDst: %d", total, totalDst)
}

func TestSearchAfterOfSelectBuilder(t *testing.T) {
	for i := 1; i <= 105; i++ {
		InitTestMain(t)
	}
	var totalDst int32
	_ = Count(db, "man", &totalDst, "1")

	obj := &SearchAfter{
		SqlStr: builder.NewSelect().Select("name").From("man"),
		// Table:  "", //  可以忽略
		// 默认按 id asc 进行查询
		// Names:    []string{},
		// Values:   []interface{}{},
		// OrderBys: []string{},
		Size: 20,
		Dest: &ManCopy{},
	}
	// 求总数
	total := 0
	obj.RowFn = func(_row interface{}) error {
		total++
		return nil
	}
	err := obj.Search(context.TODO(), db)
	if err != nil {
		t.Fatal(err)
	}

	if totalDst != int32(total) {
		t.Error("it is no ok")
	}
	t.Logf("total: %d, totalDst: %d", total, totalDst)
}

func TestSearchAfter2ResultDemo(t *testing.T) {
	InitTestMain(t, 50)
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
	results := NewSearchResults(10)
	obj.RowFn = func(_row interface{}) error {
		v := _row.(*ManCopy)
		total++
		obj.Values[0] = v.Id
		return results.Append(v).LenGte2Do(10, func(res []interface{}) error {
			fmt.Println("handle res:", res)
			return nil
		})
	}
	defer results.End2Do(func(res []interface{}) error {
		fmt.Println("final res:", res)
		return nil
	})

	err := obj.Search(context.TODO(), db)
	if err != nil {
		t.Fatal(err)
	}
}
