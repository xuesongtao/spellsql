package spellsql

import (
	"encoding/json"
	"fmt"
)

func myPrint(v interface{}, isStruct bool) {
	if !isStruct {
		fmt.Println(v)
		return
	}
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

func ExampleOrmList() {
	sqlObj := NewCacheSql("SELECT id,name,age FROM man")
	if true {
		sqlObj.SetWhereArgs("id IN (?)", []int{1, 2})
	}

	table := NewTable(db).Raw(sqlObj)
	var (
		total int
		res   = make([]*Man, 0, 10)
	)
	_ = table.Count(&total)
	_ = table.FindAll(&res, func(_row interface{}) error {
		v := _row.(*Man)
		if v.Id == 1 {
			v.Name = "被修改为 test"
		}
		return nil
	})
	myPrint(total, false)
	myPrint(res, true)

	// Output:
	// 4
	// [{"id":1,"name":"被修改为 test","age":20,"nickname":""},{"id":2,"name":"xue1","age":18,"nickname":""}]
}

func ExampleOrmInsert() {
	m := Man{
		Name: "xue1234",
		Age:  18,
		Addr: "成都市",
	}
	_, _ = NewTable(db).Insert(m).Exec()

	// Output:
}

func ExampleOrmUpdate() {
	m := Man{
		Name: "xuesongtao",
		Age:  20,
		Addr: "测试",
	}
	_, _ = NewTable(db).Update(m).Where("id=?", 7).Exec()

	// Output:
}

func ExampleOrmDelete() {
	m := Man{
		Id: 9,
	}
	_, _ = NewTable(db).Delete(m).Exec()

	// Output:
}
