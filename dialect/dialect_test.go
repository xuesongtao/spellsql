package dialect

import "testing"

func TestParsePlaceholder(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlStr := "SELECT * FROM user WHERE id=? AND name=? AND age=?"
		args := []interface{}{1, "test", 18}
		// mysql
		p := NewParsePlaceholder(MySQL, sqlStr, args...)
		parseStr := p.Parse().Result()
		if parseStr != "SELECT * FROM user WHERE id=1 AND name=\"test\" AND age=18" {
			t.Error("mysql replace error, result:", parseStr)
		}

		replaceStr := p.Replace().Result()
		if replaceStr != "SELECT * FROM user WHERE id=? AND name=? AND age=?" {
			t.Error("mysql replace error, result:", replaceStr)
		}
	})

	t.Run("?v in where", func(t *testing.T) {
		sqlStr := "SELECT * FROM user WHERE id=? AND name=(?v)"
		args := []interface{}{1, "select name from user where id=1"}
		// mysql
		p := NewParsePlaceholder(MySQL, sqlStr, args...)
		parseStr := p.Parse().Result()
		if parseStr != "SELECT * FROM user WHERE id=1 AND name=(select name from user where id=1)" {
			t.Error("mysql replace error, result:", parseStr)
		}

		replaceStr := p.Replace().Result()
		if replaceStr != "SELECT * FROM user WHERE id=? AND name=(select name from user where id=1)" {
			t.Error("mysql replace error, result:", replaceStr)
		}
	})

	t.Run("?d in arr", func(t *testing.T) {
		sqlStr := "SELECT * FROM user WHERE id=? AND name=(?v) AND age IN (?d) AND age in (?)"
		args := []interface{}{1, "select name from user where id=1", []string{"18", "19", "20"}, []int{25}}
		// mysql
		p := NewParsePlaceholder(MySQL, sqlStr, args...)
		parseStr := p.Parse().Result()
		if parseStr != "SELECT * FROM user WHERE id=1 AND name=(select name from user where id=1) AND age IN (18, 19, 20) AND age in (25)" {
			t.Error("mysql replace error, result:", parseStr)
		}

		replaceStr := p.Replace().Result()
		if replaceStr != "SELECT * FROM user WHERE id=? AND name=(select name from user where id=1) AND age IN (18, 19, 20) AND age in (?)" {
			t.Error("mysql replace error, result:", replaceStr)
		}
	})
}
