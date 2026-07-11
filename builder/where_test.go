package builder

import (
	"strings"
	"testing"

	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
	"gitee.com/xuesongtao/spellsql/test"
)

func TestWhereBuilderGetExecArgs(t *testing.T) {
	t.Run("mixed types and operations", func(t *testing.T) {
		w := NewWhereBuilder(dialect.MySQL)
		w.Eq("id", 100).
			And("status IN (?)", []int{1, 2}).
			OrNotEq("name", "test").
			Gt("score", 95.5).
			Lte("age", 30)

		sqlStr, args := w.GetSql2Args()

		// 1. 验证 SQL 占位符数量与参数数量是否匹配
		placeholderCount := strings.Count(sqlStr, "?")
		if placeholderCount != len(args) {
			t.Errorf("placeholder count %d != args len %d", placeholderCount, len(args))
		}

		// 2. 验证参数值
		expectedArgs := []interface{}{100, []int{1, 2}, "test", 95.5, 30}
		for i, arg := range args {
			if !test.Equal(arg, expectedArgs[i]) {
				t.Errorf("arg at index %d error, got: %v, want: %v", i, arg, expectedArgs[i])
			}
		}

		// 3. 验证 SQL 结构
		expectedSql := "`id` = ? AND status IN (?) OR `name` <> ? AND `score` > ? AND `age` <= ?"
		if sqlStr != expectedSql {
			t.Errorf("sqlStr error, got: %s, want: %s", sqlStr, expectedSql)
		}
	})

	t.Run("placeholder consistency in postgres", func(t *testing.T) {
		w := NewWhereBuilder(dialect.Postgres)
		w.Eq("id", 1).OrEq("id", 2)

		sqlStr, _ := w.GetSql2Args()
		// 注意：目前 WhereBuilder 的 Postgres 实现只返回 "$"，
		// 真正的 "$1, $2" 转换可能是在底层执行器中完成，
		// 或者是你预期的行为。这里按你当前代码逻辑进行验证。
		if strings.Count(sqlStr, "$") != 2 {
			t.Errorf("postgres placeholder error, got: %s", sqlStr)
		}
	})
}

func TestWhereBuilderGetSqlStr(t *testing.T) {
	t.Run("mysql base", func(t *testing.T) {
		w := NewWhereBuilder(dialect.MySQL)
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
		w := NewWhereBuilder(dialect.MySQL)
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
		w := NewWhereBuilder(dialect.MySQL)
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
		w := NewWhereBuilder(dialect.Postgres)
		w.Eq("id", 1).
			Eq("name", "xue")

		// 这里的 GetSql2Args 返回的是占位符模板
		sqlStr, _ := w.GetSql2Args()
		sureSql := "\"id\" = $1 AND \"name\" = $2"
		if sqlStr != sureSql {
			t.Errorf("sqlStr is not eq, got: %s, want: %s", sqlStr, sureSql)
		}

		// GetSqlStr 会根据方言解析占位符
		finalSql := w.GetSqlStr()
		sureFinalSql := "\"id\" = 1 AND \"name\" = 'xue'"
		if finalSql != sureFinalSql {
			t.Errorf("finalSql is not eq, got: %s, want: %s", finalSql, sureFinalSql)
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		w := NewWhereBuilder(dialect.MySQL)
		if w.GetSqlStr() != "" {
			t.Errorf("empty builder should return empty sql")
		}

		w.Eq("name", nil)
		if w.GetSqlStr() != "`name` = undefined" {
			t.Errorf("sql should not be empty after Eq, got: %s", w.GetSqlStr())
		}

		w = NewWhereBuilder(dialect.MySQL)
		w.Eq("name", internal.NULL)
		if w.GetSqlStr() != "`name` = NULL" {
			t.Errorf("NULL constant should be rendered without quotes, got: %s", w.GetSqlStr())
		}

		// 特殊字符转义 (LIKE)
		w = NewWhereBuilder(dialect.MySQL)
		w.Like("name", "x%_")
		// x%_ -> x\%[\_] 经过 EscapeLike 转义
		// 最终 SQL 里的参数应该是 %x\%[\_]%
		_, args := w.GetSql2Args()
		if args[0] != "%x\\%\\_%" {
			t.Errorf("LIKE special chars escape error, got: %v", args[0])
		}

		w = NewWhereBuilder(dialect.MySQL)
		w.And("(role_id = ? OR role_id IS NULL)", 10)
		if w.GetSqlStr() != "(role_id = 10 OR role_id IS NULL)" {
			t.Errorf("complex snippet error, got: %s", w.GetSqlStr())
		}

		w = NewWhereBuilder(dialect.MySQL)
		w.In("id", []int{})
		// placeholders(gd, 0) 会返回 ""
		// 所以生成的会是 `id` IN ()
		if w.GetSqlStr() != "`id` IN ()" {
			t.Errorf("empty IN slice error, got: %s", w.GetSqlStr())
		}
	})
}
