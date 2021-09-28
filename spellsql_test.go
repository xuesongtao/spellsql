package spellsql

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 可以重复设值
func TestNewSql_SetWheres(t *testing.T) {
	s := NewSql("SELECT username, password FROM sys_user WHERE name = ? AND money > ?", "test", 1000.00)
	s.SetWhereArgs("age > ?d", "12")
	s.SetWhere("age", "=", "18 or 1=1")
	s.SetWhere("age", "in", []string{"18 or 1=1"})
	s.GetTotalSqlStr("1")
	s.SetWhere("age", "in", []string{"20"})
	s.GetSqlStr("2")
	s.SetWhere("username", "test")
	s.GetSqlStr()

	s1 := s.Clone()
	s1.SetWhere("age", ">", 100)
	s1.GetSqlStr("3")
}

func TestGetSqlStr(t *testing.T) {
	GetSqlStr("INSERT INTO doctor_check_record (d_id, is_accept, no_accept_reasons, no_accept_img, check_id) "+
		"VALUES (?, ?, ?, ?, ?, ?d)", 1, 1, "test", "req.NoAcceptImg", 12, "1")
}

func TestFmtSqlStr(t *testing.T) {
	str := FmtSqlStr("SELECT * FROM user_info WHERE id IN (?)", []int{1, 2, 3})
	t.Log(str)

	str2 := FmtSqlStr("SELECT * FROM user_info WHERE id IN (?d)", []string{"1", "2", "3"})
	t.Log(str2)

	str3 := FmtSqlStr("SELECT account_id FROM (?v) tmp GROUP BY account_id HAVING COUNT(*)>=? ORDER BY NULL",
		"SELECT account_id FROM test1 UNION ALL SELECT account_id FROM test2", 2)
	t.Log(str3)
}

func TestFmtLikeSqlStr(t *testing.T) {
	str := GetLikeSqlStr(ALK, "SELECT id, username FROM sys_user", "name", "xue")
	t.Log(str)

	str = GetLikeSqlStr(RLK, "SELECT id, username FROM sys_user", "name", "xue")
	t.Log(str)

	str = GetLikeSqlStr(LLK, "SELECT id, username FROM sys_user", "name", "xue")
	t.Log(str)
}

// ?d 占位符单个字符串
func TestNewSql(t *testing.T) {
	s := NewSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?d", "test", "123")
	t.Log(s.SetPrintLog(false).GetTotalSqlStr("selectUserTotal"))
	t.Log(s.SetPrintLog(true).GetSqlStr("selectUser"))
}

// ?d 占位符字符串切片
func TestNewCacheSql(t *testing.T) {
	kindIds := []string{"1", "2", "3"}
	idsStr := "1,2,3,4"
	s := NewCacheSql("SELECT kind_id, kind_name FROM item_kind WHERE kind_id IN (?d) AND id IN (?d)", kindIds, idsStr)
	s.SetWhere("id", 1)
	t.Log(s.GetSqlStr())
}

func TestNewCacheSql_INSERT(t *testing.T) {
	s := NewCacheSql("INSERT INTO sys_user (username, password, name) VALUES (?, ?, ?)", "test", 123456, "阿涛")
	s.SetInsertValues("xuesongtao", "123456", "阿桃")
	s.SetInsertValues("xuesongtao", "123456", "阿桃")
	t.Log(s.GetSqlStr())
}

func TestNewCacheSql_UPDATE(t *testing.T) {
	s := NewCacheSql("UPDATE sys_user SET username = ?, password = ?, name = ? WHERE id = ?", "test", 123456, "阿涛", 12)
	t.Log(s.GetSqlStr())
}

func TestNewCacheSql_DELETE(t *testing.T) {
	s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123)
	t.Log(s.GetSqlStr())
}

// 少参数
func TestNewCacheSql_ArgNumErr(t *testing.T) {
	s := NewCacheSql("UPDATE sys_user SET username = ?, password = ?, name = ? WHERE id = ?", "test", 123456)
	t.Log(s.GetSqlStr())
}

// 普通查询
func TestSqlStr_SELECT(t *testing.T) {
	s := NewCacheSql("SELECT username, password FROM sys_user")
	s.SetWhere("username", "test")
	s.SetWhere("password", "test OR 1=1#")
	// like sql 注入测试
	s.SetWhere("name", "LIKE", "%"+"test\" or 1=1#"+"%")
	s.GetTotalSqlStr()
	s.SetLimit(0, 10)
	s.GetSqlStr()
}

// 连接 or/and
func TestMySql_SetOrWhere(t *testing.T) {
	s := NewCacheSql("SELECT * FROM user u LEFT JOIN role r ON u.id = r.user_id")
	// s.SetWhere("u.gender", 1)
	s.SetOrWhere("u.name", "xue")
	s.SetOrWhereArgs("(r.id IN (?d))", []string{"1", "2"})
	s.SetWhere("u.age", ">", 20)
	s.SetWhereArgs("u.addr = ?", "南部")
	s.GetSqlStr()
}

func TestSqlStr_InWhere(t *testing.T) {
	s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
	s.SetLimit(0, 10)
	s.SetGroupByStr("u.username")
	s.GetTotalSqlStr()
	s.GetSqlStr()
}

func TestMySql_SetWhereArgs(t *testing.T) {
	idsStr := []string{"1", "2", "3", "4", "5"}
	s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
	s.SetWhere("u.name", "test")
	s.SetWhereArgs("u.id IN (?d) AND su.name = ?", idsStr, "xuesongtao")
	s.SetWhereArgs("update_time between ? and ?", "2021-07-13", "2021-08-30")
	s.SetWhere("u.name", "test1")
	s.SetWhereArgs("(temp_name LIKE '%?d%' OR temp_content LIKE '%?d%')", "test", "1")
	s.SetLimit(0, 10).SetGroupByStr("u.username").SetOrderByStr("id DESC")
	s.GetTotalSqlStr()
	s.GetSqlStr()
}

func TestMySql_SetOrderByStr(t *testing.T) {
	idsStr := []string{"1", "2", "3", "4", "5"}
	s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
	s.SetWhereArgs("(temp_name LIKE '%?d%' OR temp_content LIKE '%?%')", "112", 1)
	s.SetOrWhereArgs("(id IN (?d) AND name= ?)", idsStr, "xue")
	s.SetLimit(0, 10).SetGroupByStr("u.username").SetOrderByStr("id DESC")
	s.GetTotalSqlStr()
	s.GetSqlStr()
}

func TestMySql_SetUpdateValueArgs(t *testing.T) {
	idsStr := []string{"1", "2", "3", "4", "5"}
	s := NewCacheSql("UPDATE sys_user SET")
	s.SetUpdateValue("name", "xue")
	s.SetUpdateValueArgs("age = ?, score = ?", 18, 90.5)
	s.SetWhereArgs("id IN (?d) AND name = ?", idsStr, "xuesongtao")
	t.Log(s.GetSqlStr())
}

// 连接查询
func TestSqlStr_SELECTGetSql1(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5}
	idsStr := []string{"1", "2", "3", "4", "5"}
	s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
	s.SetWhere("u.username", "test")
	s.SetWhere("u.password", "test")
	s.SetWhere("u.password", "IN", "SELECT id FROM t WHERE id = 10")
	s.SetWhere("u.id", "IN", ids)
	s.SetWhere("su.id", "IN", idsStr)
	s.SetLimit(0, 10)
	s.SetGroupByStr("u.username")
	s.GetTotalSqlStr()
	s.GetSqlStr()
}

// 子查询
func TestSqlStr_SELECTGetSql2(t *testing.T) {
	sqlStr := fmt.Sprintf("SELECT ea.account_id,ea.hospital_id, ei.user_name, ea.flag,ea.create_time,ea.account,ei.user_gender,ei.user_phone "+
		"FROM employee_account_info ea, employee_info ei"+
		" WHERE ea.hospital_id = %v AND ea.account_id = ei.account_id AND ea.account <> 'admin' AND ea.account <> 'syncuseraccount'", 11)
	s := NewCacheSql(sqlStr)
	s.SetWhere("ei.user_dep", "12")
	s.SetWhere("ea.account_id", "in", "1, 2, 3, 4, 5")
	s.SetWhere("ei.user_name", "LIKE", "%test%")
	s.SetOrderByStr("id DESC")
	s.SetLimit(0, 10)
	s.SetGroupByStr("ea.account")
	s.GetTotalSqlStr()
	s.GetSqlStr()
}

// 单个插入
func TestSqlStr_INSERTGetSql(t *testing.T) {
	s := NewCacheSql("INSERT INTO sys_user (username, password)")
	s.SetInsertValues("xue", 12)
	s.GetSqlStr()
}

// 多个插入
func TestSqlStr_INSERTGetSql2(t *testing.T) {
	s := NewCacheSql("INSERT INTO sys_user (username, password)")
	for i := 0; i < 10; i++ {
		s.SetInsertValues("xuesongtao", "123")
	}
	s.GetSqlStr()
}

func TestSqlStr_UPATEGetSql(t *testing.T) {
	s := NewCacheSql("UPDATE sys_user SET")
	s.SetUpdateValue("name", "xuesongtao")
	s.SetUpdateValue("password", 123)
	s.SetWhere("id", "=", 1)
	s.GetSqlStr()
}

func TestMySql_CloneForCacheSql(t *testing.T) {
	sqlObj := NewCacheSql("SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1")
	totalSqlStr := sqlObj.GetTotalSqlStr()
	t.Log(totalSqlStr)
	notifyFunc := func(obj *SqlStrObj, page, size int32) {
		sqlStr := obj.SetOrderByStr("id ASC").SetLimit(page, size).SetPrintLog(false).GetSqlStr()
		t.Log(sqlStr)
	}

	// 调用并归还
	sqlObj.GetSqlStr()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {

		// 去获取上面归还的对象
		obj := NewCacheSql("select * from user")
		obj.GetSqlStr()
		wg.Done()
	}()
	time.Sleep(time.Second)
	for page := 1; page < 20; page++ {
		wg.Add(1)
		go func(page int) {
			tmpObj := sqlObj.Clone()
			notifyFunc(tmpObj, int32(page), 100)
			wg.Done()
		}(page)
	}
	wg.Wait()
}

func TestMySql_CloneForNewSql(t *testing.T) {
	sqlObj := NewSql("SELECT u_name, phone, account_id FROM user_info WHERE u_status = 1")
	totalSqlStr := sqlObj.GetTotalSqlStr()
	t.Log(totalSqlStr)
	notifyFunc := func(obj *SqlStrObj, page, size int32) {
		sqlStr := obj.SetOrderByStr("id ASC").SetLimit(page, size).SetPrintLog(false).GetSqlStr()
		t.Log(sqlStr)
	}
	sqlObj.GetSqlStr()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		obj := NewSql("select * from user")
		obj.GetSqlStr()
		wg.Done()
	}()
	time.Sleep(time.Second)
	for page := 1; page < 20; page++ {
		wg.Add(1)
		go func(page int) {
			tmpObj := sqlObj.Clone()
			notifyFunc(tmpObj, int32(page), 100)
			wg.Done()
		}(page)
	}
	wg.Wait()
}

func TestGetSqlStrAndArgs(t *testing.T) {
	sqlStr, args, sql := GetSqlStrAndArgs("SELECT * FROM sys_user WHERE age = ?d AND name = ?", "20", "test")
	t.Log("sqlStr: ", sqlStr)
	t.Log("args: ", args)
	t.Log("sql: ", sql)
}

func TestIndexForBF(t *testing.T) {
	str := "SELECT kind_id, kind_name FROM item_kind WHERE"
	i := IndexForBF(true, str, "WHEREb")
	t.Log(i)

	// str = "SELECT kind_id, kind_name FROM item_kind WHERE"
	str = "SELECT kind_id, kind_name FROM item_kind WHERE"
	i = IndexForBF(false, str, "aSELECT")
	t.Log(i)

}

// 去重
func TestDistinctIdsStr(t *testing.T) {
	ids := ""
	for i := 0; i < 1000; i++ {
		ids += fmt.Sprintf("%d,", i%10)
	}
	t.Log("ids: ", ids)
	t.Log("ids: ", DistinctIdsStr(ids, ","))
}

func BenchmarkIndexForBF1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		str := "SELECT kind_id, kind_name FROM item_kind WHERE"
		_ = IndexForBF(true, str, "WHERE")
	}
}

func BenchmarkIndexForBF12(b *testing.B) {
	for i := 0; i < b.N; i++ {
		str := "SELECT kind_id, kind_name FROM item_kind WHERE"
		_ = IndexForBF(false, str, "WHERE")
	}
}

func BenchmarkIndexForBF(b *testing.B) {
	for i := 0; i < b.N; i++ {
		str := "GROUP BY test, test1"
		_ = IndexForBF(true, str, "GROUP BY")
	}
}

func BenchmarkStringIndex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		str := "GROUP BY test, test1"
		_ = strings.Index(str, "GROUP BY")
	}
}

func TestSqlStr_DELETEGetSql(t *testing.T) {
	s := NewSql("DELETE FROM sys_user")
	s.SetWhere("username", "=", "xue")
	t.Log(s.GetSqlStr())
}

func BenchmarkFmtInt2Str(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%d", i)
	}
}

func BenchmarkSqlStr_Int2Str(b *testing.B) {
	s := NewSql("SELECT 12")
	var i int64
	for i < int64(b.N) {
		s.Int2Str(i)
		i++
	}
}

func BenchmarkSqlStr_GetSql(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
		s.SetWhere("u.username", "test")
		s.SetWhere("u.password", "test")
		s.SetWhere("u.password", "IN", "SELECT id FROM t WHERE id = 10")
		s.SetLimit(0, 10)
		s.SetGroupByStr("u.username, u.password")
		s.SetPrintLog(false).GetTotalSqlStr()
		s.GetSqlStr()
	}
}

func BenchmarkSqlStr_GetSql2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
		s.SetWhere("u.username", "test")
		s.SetWhere("u.password", "test")
		s.SetWhere("u.password", "IN", "SELECT id FROM t WHERE id = 10")
		s.SetLimit(0, 10)
		s.SetGroupByStr("u.username, u.password")
		s.SetPrintLog(false).GetTotalSqlStr()
		s.GetSqlStr()
	}
}

func BenchmarkSqlStr_GetSql3(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := "SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id WHERE"
		s1 := "SELECT count(*) FROM sys_user su LEFT JOIN user u ON su.id = u.id WHERE"

		s += fmt.Sprintf(" u.username = %q AND", testMySQLEscape("test"))
		s1 += fmt.Sprintf(" u.username = %q AND", testMySQLEscape("test"))

		s += fmt.Sprintf(" u.password = %q AND", testMySQLEscape("test"))
		s1 += fmt.Sprintf(" u.password = %q AND", testMySQLEscape("test"))

		s += fmt.Sprintf(" u.password IN (%v) AND", testMySQLEscape("SELECT id FROM t WHERE id = 10"))
		s1 += fmt.Sprintf(" u.password IN (%v) AND", testMySQLEscape("SELECT id FROM t WHERE id = 10"))

		s += fmt.Sprintf(" LIMIT %d, %d", 0, 10)
		s1 += fmt.Sprintf(" LIMIT %d, %d", 0, 10)

		s += "GROUP BY u.username, u.password"
		s1 += "GROUP BY u.username, u.password"
		// b.Log(s)
	}
}

func testMySQLEscape(v string) string {
	var pos = 0
	buf := make([]byte, 2*len(v))
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch c {
		case '\x00':
			buf[pos] = '\\'
			buf[pos+1] = '0'
			pos += 2
		case '\n':
			buf[pos] = '\\'
			buf[pos+1] = 'n'
			pos += 2
		case '\r':
			buf[pos] = '\\'
			buf[pos+1] = 'r'
			pos += 2
		case '\x1a':
			buf[pos] = '\\'
			buf[pos+1] = 'Z'
			pos += 2
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			buf[pos] = c
			pos++
		}
	}
	return string(buf[:pos])
}
