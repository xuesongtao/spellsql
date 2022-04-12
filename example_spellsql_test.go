package spellsql

import (
	"fmt"
)

func ExampleSpellSqlList() {
	s := NewCacheSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?d", "test", "123").SetPrintLog(false)
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
	sqlStr := s.GetSqlStr("selectUser")

	fmt.Println(totalSqlStr)
	fmt.Println(sqlStr)

	// Output:
	// SELECT COUNT(*) FROM sys_user WHERE username = "test" AND password = 123 AND username = "test" OR password = "123456" AND age IN (80,100) AND id IN (1,2,3) AND cls_id IN (SELECT id FROM class WHERE cls_name="社大");
	// SELECT username, password FROM sys_user WHERE username = "test" AND password = 123 AND username = "test" OR password = "123456" AND age IN (80,100) AND id IN (1,2,3) AND cls_id IN (SELECT id FROM class WHERE cls_name="社大");
}

func ExampleSpellSqlInsert()  {
	s := NewCacheSql("INSERT INTO sys_user (username, password, name)").SetPrintLog(false)
	s.SetInsertValues("xuesongtao", "123456", "阿桃")
	s.SetInsertValues("xuesongtao1", "123456", "阿桃")
	fmt.Println(s.GetSqlStr())

	// Output:
	// INSERT INTO sys_user (username, password, name) VALUES ("xuesongtao", "123456", "阿桃"), ("xuesongtao1", "123456", "阿桃");
}

func ExampleSpellSqlUpdate()  {
	s := NewCacheSql("UPDATE sys_user SET").SetPrintLog(false)
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

func ExampleSpellSqlDelete()  {
	s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123).SetPrintLog(false)
	fmt.Println(s.GetSqlStr())

	// Output:
	// DELETE FROM sys_user WHERE id = 123;
}
