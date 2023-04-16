package spellsql

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gitee.com/xuesongtao/spellsql/test"
)

// 新增
func TestNewCacheSql_INSERT(t *testing.T) {
	t.Run("no have values", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password, name)")
		// s.SetPrintLog(false)
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password, name) VALUES ("xuesongtao", "123456", "阿桃"), ("xuesongtao", "123456", "阿桃");`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("have values", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password, name) VALUES")
		// s.SetPrintLog(false)
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password, name) VALUES ("xuesongtao", "123456", "阿桃"), ("xuesongtao", "123456", "阿桃");`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("have values", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password, name) VALUES (?, ?, ?)", "test", 123456, "阿涛")
		// s.SetPrintLog(false)
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		s.SetInsertValues("xuesongtao", "123456", "阿桃")
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password, name) VALUES ("test", 123456, "阿涛"), ("xuesongtao", "123456", "阿桃"), ("xuesongtao", "123456", "阿桃");`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("key-value", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password)")
		// s.SetPrintLog(false)
		s.SetInsertValues("xue", 12)
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password) VALUES ("xue", 12);`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("insert many", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password)")
		// s.SetPrintLog(false)
		for i := 0; i < 2; i++ {
			s.SetInsertValuesArgs("?, ?d", "xue", "123456")
			s.SetInsertValues("xue", 123456)
		}
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password) VALUES ("xue", 123456), ("xue", 123456), ("xue", 123456), ("xue", 123456);`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("duplicate update", func(t *testing.T) {
		s := NewCacheSql("INSERT INTO sys_user (username, password, age)")
		// s.SetPrintLog(false)
		s.SetInsertValuesArgs("?, ?, ?d", "xuesongtao", "123", "20")
		s.Append("ON DUPLICATE KEY UPDATE username=VALUES(?v)", "username")
		sqlStr := s.GetSqlStr()
		sureSql := `INSERT INTO sys_user (username, password, age) VALUES ("xuesongtao", "123", 20) ON DUPLICATE KEY UPDATE username=VALUES(username);`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})
}

// 删除
func TestNewCacheSql_DELETE(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123)
		// s.SetPrintLog(false)
		sqlStr := s.GetSqlStr()
		sureSql := "DELETE FROM sys_user WHERE id = 123;"
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("2", func(t *testing.T) {
		s := NewCacheSql("DELETE FROM sys_user WHERE id = ?", 123)
		// s.SetPrintLog(false)
		if true {
			s.SetWhere("age", ">", 10)
		}
		sqlStr := s.GetSqlStr()
		sureSql := "DELETE FROM sys_user WHERE id = 123 AND age > 10;"
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})
}

// 修改
func TestNewCacheSql_UPDATE(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		s := NewCacheSql("UPDATE sys_user SET username = ?, password = ?, name = ? WHERE id = ?", "test", 123456, "阿涛", 12)
		// s.SetPrintLog(false)
		sqlStr := s.GetSqlStr()
		sureSql := `UPDATE sys_user SET username = "test", password = 123456, name = "阿涛" WHERE id = 12;`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("key-value", func(t *testing.T) {
		idsStr := []string{"1", "2", "3", "4", "5"}
		s := NewCacheSql("UPDATE sys_user SET")
		// s.SetPrintLog(false)
		s.SetUpdateValue("name", "xue")
		s.SetUpdateValueArgs("age = ?, score = ?", 18, 90.5)
		s.SetWhereArgs("id IN (?d) AND age IN (?) AND name = ?", idsStr, []int{18, 20}, "xuesongtao")
		sqlStr := s.GetSqlStr()
		sureSql := `UPDATE sys_user SET name = "xue", age = 18, score = 90.5 WHERE id IN (1,2,3,4,5) AND age IN (18,20) AND name = "xuesongtao";`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("placeholder", func(t *testing.T) {
		idsStr := []string{"1", "2", "3", "4", "5"}
		s := NewCacheSql("UPDATE sys_user SET")
		// s.SetPrintLog(false)
		s.SetUpdateValue("name", "xue")
		s.SetUpdateValueArgs("age = ?, score = ?", 18, 90.5)
		s.SetWhereArgs("id IN (?d) AND name = ?", idsStr, "xuesongtao")
		sqlStr := s.GetSqlStr()
		sureSql := `UPDATE sys_user SET name = "xue", age = 18, score = 90.5 WHERE id IN (1,2,3,4,5) AND name = "xuesongtao";`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})
}

func TestNewCacheSql_Select(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		s := NewSql("SELECT username, password FROM sys_user WHERE money > ?", 1000.00)
		// s.SetPrintLog(false)
		if true {
			s.SetWhereArgs("age > ?d", "12")
		}
		if true {
			s.SetWhere("age", "=", "18 or 1=1") // 测试注入
		}
		if true {
			s.SetWhere("age", "IN", []string{"18 or 1=1"}) // 测试注入
		}
		if true {
			s.SetBetween("create_time", "2022-04-01 01:00:11", "2022-05-01 01:00:11")
		}
		if true {
			s.SetOrWhere("name", "xue")
		}
		totalSqlStr := s.GetTotalSqlStr()
		sureSql := `SELECT COUNT(*) FROM sys_user WHERE money > 1000 AND age > 12 AND age = "18 or 1=1" AND age IN ("18 or 1=1") AND (create_time BETWEEN "2022-04-01 01:00:11" AND "2022-05-01 01:00:11") OR name = "xue";`
		if !test.Equal(totalSqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}

		sqlStr := s.SetOrderByStr("create_time DESC").SetLimit(1, 10).GetSqlStr()
		sureSql = `SELECT username, password FROM sys_user WHERE money > 1000 AND age > 12 AND age = "18 or 1=1" AND age IN ("18 or 1=1") AND (create_time BETWEEN "2022-04-01 01:00:11" AND "2022-05-01 01:00:11") OR name = "xue" ORDER BY create_time DESC LIMIT 10 OFFSET 0;`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("son select", func(t *testing.T) {
		s := NewSql("SELECT username, password FROM sys_user WHERE")
		// s.SetPrintLog(false)
		if true {
			s.SetWhere("age", "IN", FmtSqlStr("SELECT age FROM user_info WHERE id=?", 10))
		}
		if true {
			s.SetWhereArgs("age IN (?v)", FmtSqlStr("SELECT age FROM user_info WHERE id=?", 10))
		}
		sqlStr := s.GetSqlStr()
		sureSql := `SELECT username, password FROM sys_user WHERE age IN (SELECT age FROM user_info WHERE id=10) AND age IN (SELECT age FROM user_info WHERE id=10);`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("list select", func(t *testing.T) {
		s := NewSql("SELECT username, password FROM sys_user WHERE")
		// s.SetPrintLog(false)
		if true {
			s.SetAllLike("name", "test")
		}
		if true {
			s.SetLeftLike("name", "test")
		}
		if true {
			s.SetRightLike("name", "test")
		}
		sqlStr := s.GetSqlStr()
		sureSql := `SELECT username, password FROM sys_user WHERE name LIKE "%test%" AND name LIKE "%test" AND name LIKE "test%";`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("list select join", func(t *testing.T) {
		s := NewSql("SELECT su.username, su.password FROM sys_user su").
			SetJoin("user_cls uc", "su.id=uc.user_id", LJI).
			SetJoin("test t", "t.user_cls_id=uc.id")
		// s.SetPrintLog(false)
		if true {
			s.SetWhere("su.name", "test")
		}
		sqlStr := s.GetSqlStr()
		sureSql := `SELECT su.username, su.password FROM sys_user su LEFT JOIN user_cls uc ON su.id=uc.user_id JOIN test t ON t.user_cls_id=uc.id WHERE su.name = "test";`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})

	t.Run("group by", func(t *testing.T) {
		s := NewSql("SELECT cls_id,COUNT(*) FROM sys_user WHERE")
		s.SetPrintLog(false)

		sqlStr := s.SetGroupByStr("cls_id").SetHaving("sum(cls_id)>10").GetSqlStr()
		sureSql := `SELECT cls_id,COUNT(*) FROM sys_user WHERE GROUP BY cls_id HAVING sum(cls_id)>10;`
		if !test.Equal(sqlStr, sureSql) {
			t.Error(test.NoEqErr)
		}
	})
}

func TestGetSqlStr(t *testing.T) {
	sqlStr := GetSqlStr("INSERT INTO doctor_check_record (d_id, is_accept, no_accept_reasons, no_accept_img, check_id) "+
		"VALUES (?, ?, ?, ?, ?, ?d)", 1, 1, "test", "req.NoAcceptImg", 12, "1")
	sureSql := `INSERT INTO doctor_check_record (d_id, is_accept, no_accept_reasons, no_accept_img, check_id) VALUES (1, 1, "test", "req.NoAcceptImg", 12, 1);`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}
}

func TestFmtSqlStr(t *testing.T) {
	sqlStr := FmtSqlStr("SELECT * FROM user_info WHERE id IN (?)", []int{1, 2, 3})
	sureSql := `SELECT * FROM user_info WHERE id IN (1,2,3)`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}

	sqlStr = FmtSqlStr("SELECT * FROM user_info WHERE id IN (?d)", []string{"1", "2", "3"})
	sureSql = `SELECT * FROM user_info WHERE id IN (1,2,3)`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}

	sqlStr = FmtSqlStr("SELECT account_id FROM (?v) tmp GROUP BY account_id HAVING COUNT(*)>=? ORDER BY NULL",
		"SELECT account_id FROM test1 UNION ALL SELECT account_id FROM test2", 2)
	sureSql = `SELECT account_id FROM (SELECT account_id FROM test1 UNION ALL SELECT account_id FROM test2) tmp GROUP BY account_id HAVING COUNT(*)>=2 ORDER BY NULL`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}

	// 组合 fmt 1
	sqlObj := NewCacheSql("id=1")
	if true {
		sqlObj.SetWhere("name", "test")
	}
	if true {
		sqlObj.SetWhereArgs("age>?", 1)
	}
	if true {
		sqlObj.SetOrWhere("name", "1")
	}
	sqlStr = sqlObj.FmtSql()
	sureSql = `id=1 AND name = "test" AND age>1 OR name = "1"`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}
}

func TestFmtLikeSqlStr(t *testing.T) {
	sqlStr := GetLikeSqlStr(ALK, "SELECT id, username FROM sys_user", "name", "xue")
	sureSql := `SELECT id, username FROM sys_user WHERE name LIKE "%xue%"`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}

	sqlStr = GetLikeSqlStr(RLK, "SELECT id, username FROM sys_user", "name", "xue")
	sureSql = `SELECT id, username FROM sys_user WHERE name LIKE "xue%"`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}

	sqlStr = GetLikeSqlStr(LLK, "SELECT id, username FROM sys_user", "name", "xue")
	sureSql = `SELECT id, username FROM sys_user WHERE name LIKE "%xue"`
	if !test.Equal(sqlStr, sureSql) {
		t.Error(test.NoEqErr)
	}
}

func TestIndexForBF(t *testing.T) {
	str := "SELECT kind_id, kind_name FROM item_kind WHERE"
	i := IndexForBF(true, str, "WHEREb")
	if i != -1 {
		t.Error(test.NoEqErr)
	}

	// str = "SELECT kind_id, kind_name FROM item_kind WHERE"
	str = "SELECT kind_id, kind_name FROM item_kind WHERE"
	i = IndexForBF(false, str, "aSELECT")
	if i != -1 {
		t.Error(test.NoEqErr)
	}
}

// 去重
func TestDistinctIdsStr(t *testing.T) {
	ids := ""
	for i := 0; i < 10; i++ {
		ids += fmt.Sprintf("%d,", i%2)
	}
	t.Log("ids: ", ids)
	res := DistinctIdsStr(ids, ",")
	if res != "0,1" {
		t.Log("ids: ", res)
		t.Error(test.NoEqErr)
	}
}

func TestDistinctIds(t *testing.T) {
	ids := []string{"0", "1", "2", "1", "0", "2"}
	res := DistinctIds(ids)
	if !test.Equal([]string{"0", "1", "2"}, res) {
		t.Error(test.NoEqErr)
	}
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

// go test -benchmem -run=^$ -bench ^BenchmarkSqlStr_GetSql gitee.com/xuesongtao/spellsql -v -count=5

func BenchmarkSqlStr_GetSql(b *testing.B) {
	b.ResetTimer()
	totalSqlStr, sqlStr := "", ""
	for i := 0; i < b.N; i++ {
		s := NewCacheSql("SELECT u.username, u.password FROM sys_user su LEFT JOIN user u ON su.id = u.id")
		s.SetWhere("u.username", "test")
		s.SetWhere("u.password", "test")
		s.SetWhere("u.password", "IN", "SELECT id FROM t WHERE id = 10")
		s.SetLimit(0, 10)
		s.SetGroupByStr("u.username, u.password")
		totalSqlStr = s.SetPrintLog(false).GetTotalSqlStr()
		sqlStr = s.GetSqlStr()
	}
	// b.Log(totalSqlStr, sqlStr)
	_ = totalSqlStr
	_ = sqlStr

	// BenchmarkSqlStr_GetSql-8          742323              1610 ns/op            1824 B/op         19 allocs/op
	// BenchmarkSqlStr_GetSql-8          721304              1610 ns/op            1824 B/op         19 allocs/op
	// BenchmarkSqlStr_GetSql-8          717390              1616 ns/op            1824 B/op         19 allocs/op
	// BenchmarkSqlStr_GetSql-8          714825              1613 ns/op            1824 B/op         19 allocs/op
	// BenchmarkSqlStr_GetSql-8          724214              1614 ns/op            1824 B/op         19 allocs/op
}

func BenchmarkSqlStr_GetSql2(b *testing.B) {
	b.ResetTimer()
	totalSqlStr, sqlStr := "", ""
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
		totalSqlStr = s
		sqlStr = s1
	}
	// b.Log(totalSqlStr, sqlStr)
	_ = totalSqlStr
	_ = sqlStr

	// BenchmarkSqlStr_GetSql2-8         714324              1643 ns/op            2320 B/op         36 allocs/op
	// BenchmarkSqlStr_GetSql2-8         678813              1637 ns/op            2320 B/op         36 allocs/op
	// BenchmarkSqlStr_GetSql2-8         705320              1640 ns/op            2320 B/op         36 allocs/op
	// BenchmarkSqlStr_GetSql2-8         704493              1642 ns/op            2320 B/op         36 allocs/op
	// BenchmarkSqlStr_GetSql2-8         697677              1665 ns/op            2320 B/op         36 allocs/op
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

func BenchmarkIntStr(b *testing.B) {
	s := int64(56)
	a := ""
	for i := 0; i < b.N; i++ {
		a = strconv.Itoa(int(s))
	}
	b.Log(a)
}

func BenchmarkIntStr1(b *testing.B) {
	s := int64(56)
	a := ""
	obj := NewCacheSql("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a = obj.Int2Str(s)
	}
	b.Log(a)
}

func BenchmarkIntStr3(b *testing.B) {
	s := int64(56)
	a := ""
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a = Str(s)
	}
	b.Log(a)
}
