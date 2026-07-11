package builder

import (
	"strings"
	"testing"

	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
	"gitee.com/xuesongtao/spellsql/test"
)

func TestWhereGetExecArgs(t *testing.T) {
	t.Run("mixed types and operations", func(t *testing.T) {
		w := NewWhere(dialect.MySQL)
		w.Eq("id", 100).
			And("status IN (?)", []int{1, 2}).
			OrNotEq("name", "test").
			Gt("score", 95.5).
			Lte("age", 30)

		sqlStr, args := w.GetSql2Args()

		placeholderCount := strings.Count(sqlStr, "?")
		if placeholderCount != len(args) {
			t.Errorf("placeholder count %d != args len %d", placeholderCount, len(args))
		}

		expectedArgs := []interface{}{100, []int{1, 2}, "test", 95.5, 30}
		for i, arg := range args {
			if !test.Equal(arg, expectedArgs[i]) {
				t.Errorf("arg at index %d error, got: %v, want: %v", i, arg, expectedArgs[i])
			}
		}

		expectedSql := "`id` = ? AND status IN (?) OR `name` <> ? AND `score` > ? AND `age` <= ?"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
	})

	t.Run("placeholder consistency in postgres", func(t *testing.T) {
		w := NewWhere(dialect.Postgres)
		w.Eq("id", 1).OrEq("id", 2)

		sqlStr, _ := w.GetSql2Args()
		if strings.Count(sqlStr, "$") != 2 {
			t.Errorf("postgres placeholder error, got: %s", sqlStr)
		}
	})
}

func TestWhereGetSqlStr(t *testing.T) {
	t.Run("mysql base", func(t *testing.T) {
		w := NewWhere(dialect.MySQL)
		w.Eq("id", 1).
			OrEq("name", "xue").
			And("age > ?", 18).
			In("role", 1, 2, 3).
			Between("create_time", "2024-01-01", "2024-01-02")

		sqlStr, args := w.GetSql2Args()
		sureSql := "`id` = ? OR `name` = ? AND age > ? AND `role` IN (?, ?, ?) AND `create_time` (BETWEEN ? AND ?)"
		if sqlStr != sureSql {
			t.Errorf("sqlStr is not eq, got: %s, want: %s", sqlStr, sureSql)
		}
		if len(args) != 8 {
			t.Errorf("args len is not eq, got: %d, want: 8", len(args))
		}
	})

	t.Run("mysql getSqlStr", func(t *testing.T) {
		w := NewWhere(dialect.MySQL)
		w.Eq("id", 1).
			OrEq("name", "xue").
			In("age", []int{18, 20})

		sqlStr := w.GetSqlStr()
		sureSql := "`id` = 1 OR `name` = \"xue\" AND `age` IN (18, 20)"
		if sqlStr != sureSql {
			t.Errorf("sqlStr is not eq, got: %s, want: %s", sqlStr, sureSql)
		}
	})

	t.Run("mysql like", func(t *testing.T) {
		w := NewWhere(dialect.MySQL)
		w.LikeLeft("name", "xue").
			OrLikeRight("addr", "beijing").
			Like("nickname", "tao")

		sqlStr, args := w.GetSql2Args()
		sureSql := "`name` LIKE ? OR `addr` LIKE ? AND `nickname` LIKE ?"
		if sqlStr != sureSql {
			t.Errorf("sqlStr is not eq, got: %s, want: %s", sqlStr, sureSql)
		}

		if args[0] != "%xue" || args[1] != "beijing%" || args[2] != "%tao%" {
			t.Errorf("args is not eq, got: %v", args)
		}
	})

	t.Run("postgres", func(t *testing.T) {
		w := NewWhere(dialect.Postgres)
		w.Eq("id", 1).
			Eq("name", "xue")

		sqlStr, _ := w.GetSql2Args()
		sureSql := "\"id\" = $1 AND \"name\" = $2"
		if sqlStr != sureSql {
			t.Errorf("sqlStr is not eq, got: %s, want: %s", sqlStr, sureSql)
		}

		finalSql := w.GetSqlStr()
		sureFinalSql := "\"id\" = 1 AND \"name\" = 'xue'"
		if finalSql != sureFinalSql {
			t.Errorf("finalSql is not eq, got: %s, want: %s", finalSql, sureFinalSql)
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		w := NewWhere(dialect.MySQL)
		if w.GetSqlStr() != "" {
			t.Errorf("empty builder should return empty sql")
		}

		w.Eq("name", nil)
		if w.GetSqlStr() != "`name` = undefined" {
			t.Errorf("sql should not be empty after Eq, got: %s", w.GetSqlStr())
		}

		w = NewWhere(dialect.MySQL)
		w.Eq("name", internal.NULL)
		if w.GetSqlStr() != "`name` = NULL" {
			t.Errorf("NULL constant should be rendered without quotes, got: %s", w.GetSqlStr())
		}

		w = NewWhere(dialect.MySQL)
		w.Like("name", "x%_")
		_, args := w.GetSql2Args()
		if args[0] != "%x\\%\\_%" {
			t.Errorf("LIKE special chars escape error, got: %v", args[0])
		}

		w = NewWhere(dialect.MySQL)
		w.And("(role_id = ? OR role_id IS NULL)", 10)
		if w.GetSqlStr() != "(role_id = 10 OR role_id IS NULL)" {
			t.Errorf("complex snippet error, got: %s", w.GetSqlStr())
		}

		w = NewWhere(dialect.MySQL)
		w.In("id", []int{})
		if w.GetSqlStr() != "`id` IN ()" {
			t.Errorf("empty IN slice error, got: %s", w.GetSqlStr())
		}
	})
}

func TestInsert(t *testing.T) {
	t.Run("mysql base insert", func(t *testing.T) {
		i := NewInsert(dialect.MySQL)
		i.Into("user").Columns("name", "age").Values("foo", 18)
		sql, args := i.GetSql2Args()
		expectedSql := "INSERT INTO user(`name`, `age`) VALUES (?, ?)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
		if len(args) != 2 || !test.Equal(args[0], "foo") || !test.Equal(args[1], 18) {
			t.Errorf("args error, got: %v", args)
		}
	})

	t.Run("mysql insert replace", func(t *testing.T) {
		i := NewInsert(dialect.MySQL)
		i.IntoReplace("user").Columns("name").Values("bar")
		sql, _ := i.GetSql2Args()
		expectedSql := "INSERT REPLACE INTO user(`name`) VALUES (?)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})

	t.Run("mysql insert ignore", func(t *testing.T) {
		i := NewInsert(dialect.MySQL)
		i.IntoIgnore("user").Columns("name").Values("baz")
		sql, _ := i.GetSql2Args()
		expectedSql := "INSERT IGNORE INTO user(`name`) VALUES (?)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})

	t.Run("mysql batch insert", func(t *testing.T) {
		i := NewInsert(dialect.MySQL)
		i.Into("user").Columns("name", "age").
			Values("foo", 18).
			Values("bar", 20)
		sql, args := i.GetSql2Args()
		expectedSql := "INSERT INTO user(`name`, `age`) VALUES (?, ?), (?, ?)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
		if len(args) != 4 || !test.Equal(args[0], "foo") || !test.Equal(args[2], "bar") {
			t.Errorf("args error, got: %v", args)
		}
	})

	t.Run("postgres insert", func(t *testing.T) {
		i := NewInsert(dialect.Postgres)
		i.Into("user").Columns("name", "age").Values("foo", 18)
		sql, args := i.GetSql2Args()
		// Test postgres quote and placeholder
		expectedSql := "INSERT INTO user(\"name\", \"age\") VALUES ($1, $2)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
		if len(args) != 2 || !test.Equal(args[0], "foo") || !test.Equal(args[1], 18) {
			t.Errorf("args error, got: %v", args)
		}
	})

	t.Run("mysql insert duplicate update", func(t *testing.T) {
		i := NewInsert(dialect.MySQL)
		i.Into("user").Columns("name", "age").
			Values("foo", 18).
			DuplicateUpdate([]string{"name", "age"}, "name")

		sql, _ := i.GetSql2Args()
		expectedSql := "INSERT INTO user(`name`, `age`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `age`=VALUES(`age`)"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})

	t.Run("postgres insert duplicate update", func(t *testing.T) {
		i := NewInsert(dialect.Postgres)
		i.Into("user").Columns("name").
			Values("foo").
			DuplicateUpdate([]string{"name"}, "name")

		sql, _ := i.GetSql2Args()
		expectedSql := "INSERT INTO user(\"name\") VALUES ($1) ON CONFLICT (\"name\") DO UPDATE SET \"name\"=EXCLUDED.\"name\""
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d := NewDelete(dialect.MySQL)
		d.From("user")
		sqlStr, args := d.GetSql2Args()
		expectedSql := "DELETE FROM user"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
		if len(args) != 0 {
			t.Errorf("args length error, got: %d, want: 0", len(args))
		}
	})

	t.Run("where", func(t *testing.T) {
		d := NewDelete(dialect.MySQL)
		d.From("user").SetWhere(NewWhere(dialect.MySQL).Eq("id", 1))
		sqlStr, args := d.GetSql2Args()
		expectedSql := "DELETE FROM user WHERE `id` = ?"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
		if !test.Equal(args, []interface{}{1}) {
			t.Errorf("args error, got: %v, want: [1]", args)
		}
	})

	t.Run("whereCb", func(t *testing.T) {
		d := NewDelete(dialect.MySQL)
		d.From("user").WhereCb(func(w *Where) {
			w.Eq("id", 1).Eq("name", "test")
		})
		sqlStr, args := d.GetSql2Args()
		expectedSql := "DELETE FROM user WHERE `id` = ? AND `name` = ?"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
		if !test.Equal(args, []interface{}{1, "test"}) {
			t.Errorf("args error, got: %v, want: [1 test]", args)
		}
	})

	t.Run("postgres", func(t *testing.T) {
		d := NewDelete(dialect.Postgres)
		d.From("user").SetWhere(NewWhere(dialect.Postgres).Eq("id", 1))
		sqlStr, args := d.GetSql2Args()
		expectedSql := "DELETE FROM user WHERE \"id\" = $1"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
		if !test.Equal(args, []interface{}{1}) {
			t.Errorf("args error, got: %v, want: [1]", args)
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("mysql base update", func(t *testing.T) {
		u := NewUpdate(dialect.MySQL)
		u.Table("sys_user").
			Set("username", "xuesongtao").
			Set("age", 18).
			SetWhere(u.Where().Eq("id", 1))

		sql, args := u.GetSql2Args()
		expectedSql := "UPDATE sys_user SET `username` = ?, `age` = ? WHERE `id` = ?"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}

		if len(args) != 3 || !test.Equal(args[0], "xuesongtao") || !test.Equal(args[1], 18) || !test.Equal(args[2], 1) {
			t.Errorf("args error, got: %v", args)
		}
	})

	t.Run("mysql update whereCb", func(t *testing.T) {
		u := NewUpdate(dialect.MySQL)
		u.Table("sys_user").
			Set("status", 1).
			WhereCb(func(wb *Where) {
				wb.Eq("id", 100).OrEq("id", 200)
			})

		sql, _ := u.GetSql2Args()
		expectedSql := "UPDATE sys_user SET `status` = ? WHERE `id` = ? OR `id` = ?"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})

	t.Run("postgres base update", func(t *testing.T) {
		u := NewUpdate(dialect.Postgres)
		u.Table("sys_user").
			Set("username", "tao").
			SetWhere(u.Where().Eq("id", 1))

		sql, _ := u.GetSql2Args()
		expectedSql := "UPDATE sys_user SET \"username\" = $1 WHERE \"id\" = $2"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})

	t.Run("postgres append", func(t *testing.T) {
		u := NewUpdate(dialect.Postgres)
		u.Table("sys_user").
			Set("username", "tao").
			SetWhere(u.Where().Eq("id", 1))
		u.AppendSql2Args("ORDER BY id DESC")
		u.AppendSql2Args("LIMIT 1")
		sql, _ := u.GetSql2Args()
		expectedSql := "UPDATE sys_user SET \"username\" = $1 WHERE \"id\" = $2 ORDER BY id DESC LIMIT 1"
		if sql != expectedSql {
			t.Errorf("sql error, got: %s, want: %s", sql, expectedSql)
		}
	})
}

func TestSelect(t *testing.T) {
	t.Run("mysql base select", func(t *testing.T) {
		s := NewSelect(dialect.MySQL)
		s.Select("id", "username", "age").
			From("sys_user").
			SetWhere(s.Where().Eq("id", 1))

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
		s := NewSelect(dialect.MySQL)

		s.Select("status", "COUNT(*) as total").
			From("sys_user").
			LeftJoin("sys_role", "sys_user.role_id = sys_role.id").
			SetWhere(s.Where().Eq("status", 1).And("age > ?", 18)).
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

		if len(args) != 3 || !test.Equal(args[0], 1) || !test.Equal(args[1], 18) || !test.Equal(args[2], 10) {
			t.Errorf("args sequence error: %v", args)
		}
	})

	t.Run("postgres placeholder and quote", func(t *testing.T) {
		s := NewSelect(dialect.Postgres)
		s.Select("id", "name").
			From("public.users").
			SetWhere(s.Where().Eq("id", 100).OrEq("name", "tao")).
			Limit(2, 10)

		sql, args := s.GetSql2Args()
		// t.Log(s.GetSqlStr())

		if !strings.Contains(sql, "\"id\"") {
			t.Errorf("Postgres quote error: %s", sql)
		}

		if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
			t.Errorf("Postgres placeholder rebind error: %s", sql)
		}

		if !strings.Contains(sql, "LIMIT 10 OFFSET 10") {
			t.Errorf("Postgres limit error: %s", sql)
		}

		if len(args) != 2 {
			t.Errorf("args len error: %d", len(args))
		}
	})

	t.Run("GetSqlStr validation", func(t *testing.T) {
		s := NewSelect(dialect.MySQL)
		s.Select("name").From("users").WhereCb(func(wb *Where) {
			wb.Eq("id", 1)
		})

		finalSql := s.GetSqlStr()
		expected := "SELECT `name` FROM users WHERE `id` = 1"
		if finalSql != expected {
			t.Errorf("GetSqlStr error, got: %s, want: %s", finalSql, expected)
		}
	})

	t.Run("select all when empty", func(t *testing.T) {
		s := NewSelect(dialect.MySQL)
		s.From("users")
		sql, _ := s.GetSql2Args()
		if !strings.HasPrefix(sql, "SELECT * FROM") {
			t.Errorf("default select should be *, got: %s", sql)
		}
	})

	t.Run("select append", func(t *testing.T) {
		s := NewSelect(dialect.MySQL)
		s.Select().From("users")
		s.AppendSql2Args("WHERE id=?d", "1")
		s.AppendSql2Args("AND hobby in (?)", []string{"reading", "coding"})
		s.AppendSql2Args("AND son_where in (?v)", "select son_where from son_table where id=1")
		sql, args := s.GetNoParseSql2Args()
		if !strings.Contains(sql, "id=?d") {
			t.Errorf("AppendSql2Args error: %s", sql)
		}
		if len(args) != 3 {
			t.Errorf("args len error: %d", len(args))
		}

		sql = s.GetSqlStr()
		expected := `SELECT * FROM users WHERE id=1 AND hobby in ("reading", "coding") AND son_where in (select son_where from son_table where id=1)`
		if sql != expected {
			t.Errorf("GetSqlStr error, got: %s, want: %s", sql, expected)
		}

		sql, args = s.GetSql2Args()
		expectedSql := "SELECT * FROM users WHERE id=? AND hobby in (?) AND son_where in (?)"
		if sql != expectedSql {
			t.Errorf("GetSqlStr error, got: %s, want: %s", sql, expectedSql)
		}
		if len(args) != 3 || !test.Equal(args[0], "1") || !test.Equal(args[1], []string{"reading", "coding"}) || !test.Equal(args[2], "select son_where from son_table where id=1") {
			t.Errorf("args len error: %d", len(args))
		}
	})
}
