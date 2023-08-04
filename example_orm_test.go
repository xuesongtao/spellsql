package spellsql

import (
	"encoding/json"
	"fmt"

	"gitee.com/xuesongtao/spellsql/test"
	// "gitlab.cd.anpro/go/common/spellsql/test"
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
		sqlObj.SetWhereArgs("id < ?", 5)
	}

	table := NewTable(db).Raw(sqlObj)
	var (
		total int
		res   = make([]*ManCopy, 0, 10)
	)
	_ = table.Count(&total)
	_ = table.FindAll(&res, func(_row interface{}) error {
		v := _row.(*ManCopy)
		if v.Id == 1 {
			v.Name = "被修改为 test"
		}
		return nil
	})
	myPrint(total, false)
	myPrint(res, true)

	// Output:
	// 4
	// [{"id":1,"name":"被修改为 test","age":20},{"id":2,"name":"xue1","age":18},{"id":3,"name":"xue12","age":18},{"id":4,"name":"xue123","age":18}]
}

func ExampleExecForSql() {
	// 新增
	insertSql := NewCacheSql("INSERT INTO man (name,age,addr) VALUES")
	insertSql.SetInsertValues("test1", 18, "四川成都")
	if _, err := ExecForSql(db, insertSql); err != nil {
		myPrint(err, false)
		return
	}

	// 修改
	updateSql := NewCacheSql("UPDATE man SET")
	updateSql.SetUpdateValue("name", "test12")
	updateSql.SetWhere("id", 8)
	if _, err := ExecForSql(db, updateSql); err != nil {
		myPrint(err, false)
		return
	}

	// 删除
	deleteSql := NewCacheSql("DELETE FROM man WHERE id=100")
	if _, err := ExecForSql(db, deleteSql); err != nil {
		myPrint(err, false)
		return
	}

	// Output:
}

func ExampleCount() {
	var count int
	_ = Count(db, "man", &count, "id<?", 10)

	myPrint(count, false)

	// Output:
	// 8
}

func ExampleInsertForObj() {
	type Tmp struct {
		Id   int32  `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Age  int32  `json:"age,omitempty"`
		Addr string `json:"addr,omitempty"`
	}

	m := Tmp{
		Name: "xue1234",
		Age:  18,
		Addr: "成都市",
	}

	r, _ := InsertForObj(db, "man", m)
	rr, _ := r.RowsAffected()
	myPrint(rr, false)

	// Output:
	// 1
}

func ExampleInsertHasDefaultForObj() {
	type Tmp struct {
		Id   int32  `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Age  int32  `json:"age,omitempty"`
		Addr string `json:"addr,omitempty"`
	}

	m := Tmp{
		Name: "xue1234",
		Addr: "成都市",
	}

	r, err := InsertHasDefaultForObj(db, "man", nil, m)
	if err != nil {
		fmt.Println(`field "age" should't null, you can first call TagDefault`)
		return
	}
	rr, _ := r.RowsAffected()
	myPrint(rr, false)

	// Output:
	// field "age" should't null, you can first call TagDefault
}

func ExampleUpdateForObj() {
	type Tmp struct {
		Id   int32  `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Age  int32  `json:"age,omitempty"`
		Addr string `json:"addr,omitempty"`
	}

	m := Tmp{
		Name: "xue1234",
		Age:  18,
		Addr: "成都市",
	}

	_, _ = UpdateForObj(db, "man", m, "id=7")
	// rr, _ := r.RowsAffected()
	// myPrint(rr, false)

	var b Tmp
	_ = SelectFindOne(db, "name,age,addr", "man", "id=7", &b)
	myPrint(b, true)
	// Output:
	// {"name":"xue1234","age":18,"addr":"成都市"}
}

func ExampleDeleteWhere() {
	_, _ = DeleteWhere(db, "man", "id=100")

	// Output:
}

func ExampleFindWhere() {
	var m test.Man
	_ = FindWhere(db, "man", &m, "id=?", 1)

	myPrint(m, true)

	// Output:
	// {"id":1,"name":"测试1","age":20,"json_txt":{},"xml_txt":{},"json1_txt":{}}
}

func ExampleSelectFindWhere() {
	var m test.Man
	_ = SelectFindWhere(db, "name,addr", "man", &m, "id=?", 1)

	myPrint(m, true)

	// Output:
	// {"name":"测试1","json_txt":{},"xml_txt":{},"json1_txt":{}}
}

func ExampleSelectFindOne() {
	var m test.Man
	_ = SelectFindOne(db, "name,addr", "man", "id=1", &m)

	myPrint(m, true)

	// Output:
	// {"name":"测试1","json_txt":{},"xml_txt":{},"json1_txt":{}}
}

func ExampleSelectFindOneFn() {
	var m test.Man
	_ = SelectFindOneFn(db, "name,age", "man", "id=1", &m, func(_row interface{}) error {
		v := _row.(*test.Man)
		v.Name = "被修改了哦"
		return nil
	})

	myPrint(m, true)

	// Output:
	// {"name":"被修改了哦","age":20,"json_txt":{},"xml_txt":{},"json1_txt":{}}
}

func ExampleSelectFindOneIgnoreResult() {
	var m test.Man
	var idMap = make(map[int32]string, 10)
	_ = SelectFindOneIgnoreResult(db, "id,name", "man", "id<10", &m, func(_row interface{}) error {
		v := _row.(*test.Man)
		idMap[v.Id] = v.Name
		return nil
	})

	myPrint(idMap, true)

	// Output:
	// {"1":"测试1","2":"xue1","3":"xue12","4":"xue123","5":"xue1234","6":"xue1234","7":"xue1234","8":"test12"}
}

func ExampleSelectFindAll() {
	var m []test.Man
	_ = SelectFindAll(db, "id,name", "man", "id<3", &m)

	myPrint(m, true)

	// Output:
	// [{"id":1,"name":"测试1","json_txt":{},"xml_txt":{},"json1_txt":{}},{"id":2,"name":"xue1","json_txt":{},"xml_txt":{},"json1_txt":{}}]
}

func ExampleFindOne() {
	var m test.Man
	sqlObj := NewCacheSql("SELECT name,age,addr FROM man WHERE id=?", 1)
	_ = FindOne(db, sqlObj, &m)

	myPrint(m, true)

	// Output:
	// {"name":"测试1","age":20,"json_txt":{},"xml_txt":{},"json1_txt":{}}
}
