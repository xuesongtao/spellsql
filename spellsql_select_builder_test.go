package spellsql

import (
	"strings"
	"testing"

	"gitee.com/xuesongtao/spellsql/test"
)

func TestSelectBuilder(t *testing.T) {
	t.Run("mysql base select", func(t *testing.T) {
		s := NewSelectBuilder(MySQL)
		s.Select("id", "username", "age").
			From("sys_user").
			Where(s.WB().Eq("id", 1))

		sql, args := s.GetSql2Args()
		expectedSql := "SELECT `id`, `username`, `age` FROM sys_user WHERE `id` = ?"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("args error, got: %v", args)
		}
	})

	t.Run("mysql complex query", func(t *testing.T) {
		s := NewSelectBuilder(MySQL)

		s.Select("status", "COUNT(*) as total").
			From("sys_user").
			LeftJoin("sys_role", "sys_user.role_id = sys_role.id").
			Where(s.WB().Eq("status", 1).And("age > ?", 18)).
			GroupBy("status").
			Having("total > ?", 10).
			OrderByDesc("total").
			Limit(1, 20)

		sql, args := s.GetSql2Args()

		// 校验结构顺序
		if !strings.Contains(sql, "SELECT `status`, `COUNT(*) as total` FROM sys_user") {
			t.Errorf("SELECT/FROM error: %s", sql)
		}
		if !strings.Contains(sql, "LEFT JOIN sys_role ON sys_user.role_id = sys_role.id") {
			t.Errorf("JOIN error: %s", sql)
		}
		if !strings.Contains(sql, "GROUP BY `status` HAVING total > ?") {
			t.Errorf("GROUP/HAVING error: %s", sql)
		}
		if !strings.Contains(sql, "ORDER BY `total` DESC") {
			t.Errorf("ORDER error: %s", sql)
		}
		if !strings.Contains(sql, "LIMIT 20 OFFSET 0") {
			t.Errorf("LIMIT error: %s", sql)
		}

		// 校验参数合并顺序 (WhereArgs -> HavingArgs)
		if len(args) != 3 || !test.Equal(args[0], 1) || !test.Equal(args[1], 18) || !test.Equal(args[2], 10) {
			t.Errorf("args sequence error: %v", args)
		}
	})

	t.Run("postgres placeholder and quote", func(t *testing.T) {
		s := NewSelectBuilder(Postgres)
		s.Select("id", "name").
			From("public.users").
			Where(s.WB().Eq("id", 100).OrEq("name", "tao")).
			Limit(2, 10)

		sql, args := s.GetSql2Args()
		// t.Log(s.GetSqlStr())

		// 1. 校验引号为双引号
		if !strings.Contains(sql, "\"id\"") {
			t.Errorf("Postgres quote error: %s", sql)
		}

		// 2. 校验占位符为 $1, $2
		if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
			t.Errorf("Postgres placeholder rebind error: %s", sql)
		}

		// 3. 校验分页语法
		if !strings.Contains(sql, "LIMIT 10 OFFSET 10") {
			t.Errorf("Postgres limit error: %s", sql)
		}

		if len(args) != 2 {
			t.Errorf("args len error: %d", len(args))
		}
	})

	t.Run("GetSqlStr validation", func(t *testing.T) {
		s := NewSelectBuilder(MySQL)
		s.Select("name").From("users").WhereCb(func(wb *WhereBuilder) {
			wb.Eq("id", 1)
		})

		finalSql := s.GetSqlStr()
		expected := "SELECT `name` FROM users WHERE `id` = 1"
		if finalSql != expected {
			t.Errorf("GetSqlStr error, got: %s, want: %s", finalSql, expected)
		}
	})

	t.Run("select all when empty", func(t *testing.T) {
		s := NewSelectBuilder(MySQL)
		s.From("users")
		sql, _ := s.GetSql2Args()
		if !strings.HasPrefix(sql, "SELECT * FROM") {
			t.Errorf("default select should be *, got: %s", sql)
		}
	})
}
