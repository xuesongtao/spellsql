package spellsql

import (
	"strings"
)

// SetJoin 设置 join
func (s *SqlStrObj) SetJoin(tableName string, on string, joinType ...uint8) *SqlStrObj {
	deferJoinStr := "JOIN"
	if len(joinType) > 0 {
		switch joinType[0] {
		case LJI:
			deferJoinStr = "LEFT JOIN"
		case RJI:
			deferJoinStr = "RIGHT JOIN"
		}
	}
	s.buf.WriteString(" " + deferJoinStr + " " + tableName + " ON " + on)
	return s
}

// SetLeftJoin 设置 left join
func (s *SqlStrObj) SetLeftJoin(tableName string, on string) *SqlStrObj {
	return s.SetJoin(tableName, on, LJI)
}

// SetRightJoin 设置 right join
func (s *SqlStrObj) SetRightJoin(tableName string, on string) *SqlStrObj {
	return s.SetJoin(tableName, on, RJI)
}

// SetWhere 设置过滤条件, 连接符为 AND
// 如果 len = 1 的时候, 会拼接成: filed = arg
// 如果 len = 2 的时候, 会拼接成: filed arg[0] arg[1]
func (s *SqlStrObj) SetWhere(fieldName string, args ...interface{}) *SqlStrObj {
	s.initWhere()
	return s.setWhere(fieldName, args...)
}

// SetOrWhere 设置过滤条件, 连接符为 OR
// 如果 len = 1 的时候, 会拼接成: filed = arg
// 如果 len = 2 的时候, 会拼接成: filed arg[0] arg[1]
func (s *SqlStrObj) SetOrWhere(fieldName string, args ...interface{}) *SqlStrObj {
	s.initWhere("OR")
	return s.setWhere(fieldName, args...)
}

// initWhere 初始化where, joinStr 为 AND/OR 连接符, 默认时 AND
func (s *SqlStrObj) initWhere(joinStr ...string) {
	defaultJoinStr := " AND"
	if len(joinStr) > 0 {
		// 这里为 " OR"
		defaultJoinStr = " " + joinStr[0]
	}

	// fmtSql action 可能为 0
	if s.is(none) {
		if s.SqlStrLen() > 0 || s.WhereStrLen() > 0 {
			s.whereBuf.WriteString(defaultJoinStr)
		}
		return
	}

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
		lastIndex := s.SqlStrLen() - 1

		// 需要跳过本身长度
		if lastIndex-(whereIndex+5) > 5 {
			s.whereBuf.WriteString(defaultJoinStr)
		}
	}
}

// setWhere 转换参数
func (s *SqlStrObj) setWhere(fieldName string, args ...interface{}) *SqlStrObj {
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
	s.writeSqlStr2Buf(&s.whereBuf, " "+sqlStr, arg)
	return s
}

// SetRightLike 设置右模糊查询, 如: xxx LIKE "test%"
func (s *SqlStrObj) SetRightLike(fieldName string, val string) *SqlStrObj {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", val+"%")
	return s
}

// SetLeftLike 设置左模糊查询, 如: xxx LIKE "%test"
func (s *SqlStrObj) SetLeftLike(fieldName string, val string) *SqlStrObj {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", "%"+val)
	return s
}

// SetAllLike 设置全模糊, 如: xxx LIKE "%test%"
func (s *SqlStrObj) SetAllLike(fieldName string, val string) *SqlStrObj {
	s.initWhere()
	s.setWhere(fieldName, "LIKE", "%"+val+"%")
	return s
}

// SetBetween 设置 BETWEEN ? AND ?
func (s *SqlStrObj) SetBetween(fieldName string, leftVal, rightVal interface{}) *SqlStrObj {
	return s.SetWhereArgs("(?v BETWEEN ? AND ?)", fieldName, leftVal, rightVal)
}

// SetWhereArgs 支持占位符
// 如: SetWhereArgs("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (s *SqlStrObj) SetWhereArgs(sqlStr string, args ...interface{}) *SqlStrObj {
	s.initWhere()
	s.writeSqlStr2Buf(&s.whereBuf, " "+sqlStr, args...)
	return s
}

// SetOrWhereArgs 支持占位符
// 如: SetOrWhereArgs("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (s *SqlStrObj) SetOrWhereArgs(sqlStr string, args ...interface{}) *SqlStrObj {
	s.initWhere("OR")
	s.writeSqlStr2Buf(&s.whereBuf, " "+sqlStr, args...)
	return s
}

// WhereIsEmpty 判断where条件是否为空
func (s *SqlStrObj) WhereIsEmpty() bool {
	return s.WhereStrLen() == 0
}

// WhereStrLen where 条件内容长度
func (s *SqlStrObj) WhereStrLen() int {
	return s.whereBuf.Len()
}

// SetOrderByStr 设置排序
func (s *SqlStrObj) SetOrderByStr(orderByStr string) *SqlStrObj {
	s.orderByStr = " ORDER BY " + orderByStr
	return s
}

// GetOffset 根据分页获取 offset
// 注: page, size 只支持 int 系列类型
func (s *SqlStrObj) GetOffset(page, size interface{}) (int64, int64) {
	pageInt64, sizeInt64 := Int64(page), Int64(size)
	if pageInt64 <= 0 {
		pageInt64 = 1
	}
	if sizeInt64 <= 0 {
		sizeInt64 = 10
	}
	return sizeInt64, (pageInt64 - 1) * sizeInt64
}

// SetLimit 设置分页
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func (s *SqlStrObj) SetLimit(page, size interface{}) *SqlStrObj {
	sizeInt64, offsetInt64 := s.GetOffset(page, size)
	return s.SetLimitStr(s.Int2Str(sizeInt64) + " OFFSET " + s.Int2Str(offsetInt64))
}

// SetLimitStr 字符串来设置
func (s *SqlStrObj) SetLimitStr(limitStr string) *SqlStrObj {
	s.limitStr = " LIMIT " + limitStr
	return s
}

// LimitIsEmpty 是否添加 limit
func (s *SqlStrObj) LimitIsEmpty() bool {
	return null(s.limitStr)
}

// SetGroupByStr 设置 groupBy
func (s *SqlStrObj) SetGroupByStr(groupByStr string) *SqlStrObj {
	s.groupByStr = " GROUP BY " + groupByStr
	return s
}

// SetHaving 设置 Having
func (s *SqlStrObj) SetHaving(having string, args ...interface{}) *SqlStrObj {
	tmpBuf := getTmpBuf()
	defer putTmpBuf(tmpBuf)
	s.writeSqlStr2Buf(tmpBuf, having, args...)
	s.groupByStr += " HAVING " + tmpBuf.String()
	return s
}
