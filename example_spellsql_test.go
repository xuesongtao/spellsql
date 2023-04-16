package spellsql

import (
	"fmt"
	"math"
)

func ExampleSpellSqlList() {
	s := NewCacheSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?d", "test", "123").SetPrintLog(false)
	// s.SetPrintLog(false)
	if true {
		s.SetWhere("username", "test")
	}

	if true {
		s.SetOrWhere("password", "123456")
	}

	if true { // 占位符 ?
		s.SetWhereArgs("age IN (?)", []int{80, 100})
	}

	if true { // 占位符 ?d
		s.SetWhereArgs("id IN (?d)", []string{"1", "2", "3"})
	}

	if true { // 占位符 ?v
		s.SetWhereArgs("cls_id IN (?v)", FmtSqlStr("SELECT id FROM class WHERE cls_name=?", "社大"))
	}

	totalSqlStr := s.GetTotalSqlStr("selectUserTotal")
	sqlStr := s.SetOrderByStr("create_time DESC").SetLimit(1, 10).GetSqlStr("selectUser")

	fmt.Println(totalSqlStr)
	fmt.Println(sqlStr)

	// Output:
	// SELECT COUNT(*) FROM sys_user WHERE username = "test" AND password = 123 AND username = "test" OR password = "123456" AND age IN (80,100) AND id IN (1,2,3) AND cls_id IN (SELECT id FROM class WHERE cls_name="社大");
	// SELECT username, password FROM sys_user WHERE username = "test" AND password = 123 AND username = "test" OR password = "123456" AND age IN (80,100) AND id IN (1,2,3) AND cls_id IN (SELECT id FROM class WHERE cls_name="社大") ORDER BY create_time DESC LIMIT 10 OFFSET 0;
}

func ExampleSpellSqlInsert() {
	s := NewCacheSql("INSERT INTO sys_user (username, password, name)")
	// s.SetPrintLog(false)
	s.SetInsertValues("xuesongtao", "123456", "阿桃")
	s.SetInsertValues("xuesongtao1", "123456", "阿桃")
	fmt.Println(s.GetSqlStr())

	// Output:
	// INSERT INTO sys_user (username, password, name) VALUES ("xuesongtao", "123456", "阿桃"), ("xuesongtao1", "123456", "阿桃");
}

func ExampleSpellSqlUpdate() {
	s := NewCacheSql("UPDATE sys_user SET")
	// s.SetPrintLog(false)
	if true {
		s.SetUpdateValue("username", "test")
	}

	if true {
		s.SetUpdateValueArgs("age=?", 10)
	}

	s.SetWhere("id", 1)
	fmt.Println(s.GetSqlStr())

	// Output:
	// UPDATE sys_user SET username = "test", age=10 WHERE id = 1;
}

func ExampleSpellSqlDelete() {
	s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123)
	// s.SetPrintLog(false)
	fmt.Println(s.GetSqlStr())

	// Output:
	// DELETE FROM sys_user WHERE id = 123;
}

// NewCacheSql 分页处理场景
func ExampleNewCacheSqlPageHandle() {
	sqlObj := NewCacheSql("SELECT * FROM user_info WHERE status = 1")
	handleFn := func(obj *SqlStrObj, page, size int32) {
		// 业务代码
		fmt.Println(obj.SetLimit(page, size).SetPrintLog(false).GetSqlStr())
	}

	// 每次同步大小
	var (
		totalNum  int32 = 30
		page      int32 = 1
		size      int32 = 10
		totalPage int32 = int32(math.Ceil(float64(totalNum / size)))
	)
	sqlStr := sqlObj.SetPrintLog(false).GetSqlStr("", "")
	for page <= totalPage {
		handleFn(NewCacheSql(sqlStr), page, size)
		page++
	}

	// Output:
	// SELECT * FROM user_info WHERE status = 1 LIMIT 10 OFFSET 0;
	// SELECT * FROM user_info WHERE status = 1 LIMIT 10 OFFSET 10;
	// SELECT * FROM user_info WHERE status = 1 LIMIT 10 OFFSET 20;
}

// NewSql 分页处理场景
func ExampleNewSqlPageHandle() {
	sqlObj := NewSql("SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1")
	handleFn := func(obj *SqlStrObj, page, size int32) {
		// 业务代码
		fmt.Println(obj.SetLimit(page, size).SetPrintLog(false).GetSqlStr())
	}

	// 每次同步大小
	var (
		totalNum  int32 = 30
		page      int32 = 1
		size      int32 = 10
		totalPage int32 = int32(math.Ceil(float64(totalNum / size)))
	)
	for page <= totalPage {
		handleFn(sqlObj.Clone(), page, size)
		page++
	}

	// Output:
	// SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1 LIMIT 10 OFFSET 0;
	// SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1 LIMIT 10 OFFSET 10;
	// SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1 LIMIT 10 OFFSET 20;
}

func ExampleFmtSqlStr() {
	sqlStr := FmtSqlStr("SELECT * FROM ?v WHERE id IN (?d) AND name=?", "user_info", []string{"1", "2"}, "测试")
	fmt.Println(sqlStr)

	// Output:
	// SELECT * FROM user_info WHERE id IN (1,2) AND name="测试"
}
