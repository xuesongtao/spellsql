package spellsql

import (
	"strings"
)

type SelectBuilder struct {
	buf        strings.Builder // 记录骨架或者最终 sqlStr
	whereBuf   strings.Builder // 记录带有占位符 (?) 的 WHERE 条件
	whereArgs  []interface{}   // 记录 WHERE 条件的所有参数
	limitStr   string
	orderByStr string
	groupByStr string
	havingStr  string        // 记录 HAVING 条件
	havingArgs []interface{} // 记录 HAVING 的所有参数

	hasWhereStr    bool // 是否有 where
	needAddJoinStr bool // 是否需要添加连接词
}

func NewSelectBuilder() *SelectBuilder {
	obj := &SelectBuilder{
		buf:        strings.Builder{},
		whereBuf:   strings.Builder{},
		whereArgs:  make([]interface{}, 0, 4),
		havingArgs: make([]interface{}, 0),
		limitStr:   "",
		orderByStr: "",
		groupByStr: "",
	}
	return obj
}

func (s *SelectBuilder) Select(col ...string) *SelectBuilder {
	s.buf.WriteString("SELECT ")
	if len(col) > 0 {
		s.buf.WriteString(strings.Join(col, ", "))
	} else {
		s.buf.WriteString("*")
	}
	return s
}

// Join 设置 join
func (s *SelectBuilder) Join(tableName string, on string, joinType ...uint8) *SelectBuilder {
	deferJoinStr := "JOIN"
	if len(joinType) > 0 {
		switch joinType[0] {
		case LJI:
			deferJoinStr = "LEFT JOIN"
		case RJI:
			deferJoinStr = "RIGHT JOIN"
		}
	}
	s.buf.WriteString(" ")
	s.buf.WriteString(deferJoinStr)
	s.buf.WriteString(" ")
	s.buf.WriteString(tableName)
	s.buf.WriteString(" ON ")
	s.buf.WriteString(on)
	return s
}

// LeftJoin 设置 left join
func (s *SelectBuilder) LeftJoin(tableName string, on string) *SelectBuilder {
	return s.Join(tableName, on, LJI)
}

// RightJoin 设置 right join
func (s *SelectBuilder) RightJoin(tableName string, on string) *SelectBuilder {
	return s.Join(tableName, on, RJI)
}

// Where 设置过滤条件, 连接符为 AND
// 如果 len = 1 的时候, 会拼接成: filed = arg
// 如果 len = 2 的时候, 会拼接成: filed arg[0] arg[1]
func (s *SelectBuilder) Where(fieldName string, args ...interface{}) *SelectBuilder {
	s.initWhere()
	return s.setWhere(fieldName, args...)
}

// OrWhere 设置过滤条件, 连接符为 OR
// 如果 len = 1 的时候, 会拼接成: filed = arg
// 如果 len = 2 的时候, 会拼接成: filed arg[0] arg[1]
func (s *SelectBuilder) OrWhere(fieldName string, args ...interface{}) *SelectBuilder {
	s.initWhere("OR")
	return s.setWhere(fieldName, args...)
}

// initWhere 初始化where, joinStr 为 AND/OR 连接符, 默认时 AND
func (s *SelectBuilder) initWhere(joinStr ...string) {
	defaultJoinStr := " AND"
	if len(joinStr) > 0 {
		// 这里为 " OR"
		defaultJoinStr = " " + joinStr[0]
	}

	if s.WhereStrLen() > 0 {
		s.whereBuf.WriteString(defaultJoinStr)
	}
	// return

	isNeedAddJoinStr := true // 本次默认添加
	if !s.hasWhereStr {
		s.whereBuf.WriteString(" WHERE")
		s.hasWhereStr = true
		isNeedAddJoinStr = false
	}

	// 这里是第一次拼接 WHERE, 需要判断下是否要加 AND/OR
	if s.needAddJoinStr {
		s.whereBuf.WriteString(defaultJoinStr)
		s.needAddJoinStr = false
		isNeedAddJoinStr = false
	}

	if isNeedAddJoinStr {
		// 如果 where 里有值的话就添加 AND
		if s.WhereStrLen() > 0 {
			s.whereBuf.WriteString(defaultJoinStr)
			return
		}

		// 如果初始化时或已经merge后 sqlStr已经这样了: xxx WHERE xxx, 我们通过判断 WHERE 的下标是否为最后几个字符, 如果是的话就
		// 不处理, 反之加 AND
		whereIndex := getTargetIndex(s.buf.String(), "WHERE")
		if whereIndex == -1 {
			return
		}
		lastIndex := s.buf.Len() - 1

		// 需要跳过本身长度
		if lastIndex-(whereIndex+5) > 5 {
			s.whereBuf.WriteString(defaultJoinStr)
		}
	}
}

// setWhere 转换参数
func (s *SelectBuilder) setWhere(fieldName string, args ...interface{}) *SelectBuilder {
	argsLen := len(args)
	if argsLen == 0 {
		args = []interface{}{"NULL"}
	}

	// 默认操作符为 "="
	opSymbol := "="
	arg := args[0]
	// 如果参数个数大于等于 2 的话, 就会包含操作符, 所以这里需要替换下
	if argsLen >= 2 {
		tmpOpSymbol, ok := args[0].(string)
		if ok {
			opSymbol = tmpOpSymbol
		}
		arg = args[1]
	}

	// 处理字段, 如: fieldName = "test"
	sqlStr := fieldName + " " + opSymbol
	needAdd := true // 标记是否需要添加占位符
	switch opSymbol {
	case "IN", "in":
		sqlStr += " ("
		if v, ok := arg.(string); ok {
			// 子查询就原样输入
			v = strings.TrimPrefix(v, "") // 去掉空
			if len(v) > 6 && toUpper(v[:6]) == "SELECT" {
				sqlStr += "?v"
				needAdd = false
			}
		}

		if needAdd {
			sqlStr += "?"
			needAdd = false
		}
		sqlStr += ")"
	}

	if needAdd {
		sqlStr += " ?"
	}
	s.whereBuf.WriteString(" " + sqlStr)
	s.whereArgs = append(s.whereArgs, arg)
	return s
}

// SetRightLike 设置右模糊查询, 如: xxx LIKE "test%"
func (s *SelectBuilder) RightLike(fieldName string, val string) *SelectBuilder {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", EscapeLike(val)+"%")
	return s
}

// LeftLike 设置左模糊查询, 如: xxx LIKE "%test"
func (s *SelectBuilder) LeftLike(fieldName string, val string) *SelectBuilder {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", "%"+EscapeLike(val))
	return s
}

// AllLike 设置全模糊, 如: xxx LIKE "%test%"
func (s *SelectBuilder) AllLike(fieldName string, val string) *SelectBuilder {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", "%"+EscapeLike(val)+"%")
	return s
}

// Between 设置 BETWEEN ? AND ?
func (s *SelectBuilder) Between(fieldName string, leftVal, rightVal interface{}) *SelectBuilder {
	return s.WhereArgs("(?v BETWEEN ? AND ?)", fieldName, leftVal, rightVal)
}

// WhereArgs 支持占位符
// 如: WhereArgs("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (s *SelectBuilder) WhereArgs(sqlStr string, args ...interface{}) *SelectBuilder {
	s.initWhere()
	s.whereBuf.WriteString(" " + sqlStr)
	s.whereArgs = append(s.whereArgs, args...)
	return s
}

// OrWhereArgs 支持占位符
// 如: OrWhereArgs("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (s *SelectBuilder) OrWhereArgs(sqlStr string, args ...interface{}) *SelectBuilder {
	s.initWhere("OR")
	s.whereBuf.WriteString(" " + sqlStr)
	s.whereArgs = append(s.whereArgs, args...)
	return s
}

// WhereIsEmpty 判断where条件是否为空
func (s *SelectBuilder) WhereIsEmpty() bool {
	return s.WhereStrLen() == 0
}

// WhereStrLen where 条件内容长度
func (s *SelectBuilder) WhereStrLen() int {
	return s.whereBuf.Len()
}

// OrderBy 设置排序
func (s *SelectBuilder) OrderBy(orderByStr string) *SelectBuilder {
	s.orderByStr = " ORDER BY " + orderByStr
	return s
}

// Limit 设置分页
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func (s *SelectBuilder) Limit(page, size interface{}) *SelectBuilder {
	sizeInt64, offsetInt64 := GetOffset(page, size)
	return s.LimitStr(Int2Str(sizeInt64) + " OFFSET " + Int2Str(offsetInt64))
}

// LimitStr 字符串来设置
func (s *SelectBuilder) LimitStr(limitStr string) *SelectBuilder {
	s.limitStr = " LIMIT " + limitStr
	return s
}

// LimitIsEmpty 是否添加 limit
func (s *SelectBuilder) LimitIsEmpty() bool {
	return null(s.limitStr)
}

// GroupBy 设置 groupBy
func (s *SelectBuilder) GroupBy(groupByStr string) *SelectBuilder {
	s.groupByStr = " GROUP BY " + groupByStr
	return s
}

// Having 设置 Having
func (s *SelectBuilder) Having(having string, args ...interface{}) *SelectBuilder {
	s.havingStr = " HAVING " + having
	s.havingArgs = append(s.havingArgs, args...)
	return s
}

// GetExecArgs 返回带有占位符的 SQL 和参数切片，适合传给 db.Query 防止注入并利用 DB 软解析
func (s *SelectBuilder) GetExecArgs() (string, []interface{}) {
	var finalSql strings.Builder
	finalSql.WriteString(s.buf.String())
	if s.WhereStrLen() > 0 {
		finalSql.WriteString(s.whereBuf.String())
	}
	if s.groupByStr != "" {
		finalSql.WriteString(s.groupByStr)
	}
	if s.havingStr != "" {
		finalSql.WriteString(s.havingStr)
	}
	if s.orderByStr != "" {
		finalSql.WriteString(s.orderByStr)
	}
	if s.limitStr != "" {
		finalSql.WriteString(s.limitStr)
	}

	args := make([]interface{}, 0, len(s.whereArgs)+len(s.havingArgs))
	args = append(args, s.whereArgs...)
	args = append(args, s.havingArgs...)

	return finalSql.String(), args
}

// GetSqlStr 获取拼接了实际参数的最终可执行 SQL (主要用于测试或打印日志)
func (s *SelectBuilder) GetSqlStr() string {
	var finalSql strings.Builder
	finalSql.WriteString(s.buf.String())

	if s.WhereStrLen() > 0 {
		finalSql.WriteString(Parse(s.whereBuf.String(), s.whereArgs...).String())
	}

	if s.groupByStr != "" {
		if len(s.havingArgs) > 0 {
			finalSql.WriteString(Parse(s.groupByStr, s.havingArgs...).String())
		} else {
			finalSql.WriteString(s.groupByStr)
		}
	}

	if s.orderByStr != "" {
		finalSql.WriteString(s.orderByStr)
	}

	if s.limitStr != "" {
		finalSql.WriteString(s.limitStr)
	}

	return finalSql.String()
}

// EscapeLike 转义 like
func EscapeLike(val string) string {
	res := Escape(
		[]byte(val),
		map[byte][]byte{
			'_': {'\\', '_'},
			'%': {'\\', '%'},
		})
	return string(res)
}

// GetOffset 根据分页获取 offset
// 注: page, size 只支持 int 系列类型
func GetOffset(page, size interface{}) (int64, int64) {
	pageInt64, sizeInt64 := Int64(page), Int64(size)
	if pageInt64 <= 0 {
		pageInt64 = 1
	}
	if sizeInt64 <= 0 {
		sizeInt64 = 10
	}
	return sizeInt64, (pageInt64 - 1) * sizeInt64
}
