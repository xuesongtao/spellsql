package spellsql

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

func (s *SqlStrObj) getSelectBuilder() *builder.Select {
	return s.builder.(*builder.Select)
}

// SetJoin 设置 join
func (s *SqlStrObj) SetJoin(tableName string, on string, joinType ...uint8) *SqlStrObj {
	if len(joinType) > 0 {
		switch joinType[0] {
		case LJI:
			s.getSelectBuilder().LeftJoin(tableName, on)
		case RJI:
			s.getSelectBuilder().RightJoin(tableName, on)
		}
	} else {
		s.getSelectBuilder().Join(tableName, on)
	}
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
	return s.setWhere(internal.SELECT, fieldName, args...)
}

// SetOrWhere 设置过滤条件, 连接符为 OR
// 如果 len = 1 的时候, 会拼接成: filed = arg
// 如果 len = 2 的时候, 会拼接成: filed arg[0] arg[1]
func (s *SqlStrObj) SetOrWhere(fieldName string, args ...interface{}) *SqlStrObj {
	return s.setWhere(internal.SELECT_OR, fieldName, args...)
}

// setWhere 转换参数
func (s *SqlStrObj) setWhere(opType internal.OpType, fieldName string, args ...interface{}) *SqlStrObj {
	argsLen := len(args)
	if argsLen == 0 {
		args = []interface{}{internal.NULL}
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
			if len(v) > 6 && internal.ToUpper(v[:6]) == "SELECT" {
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
	// s.getSelectBuilder().Where().AppendSql2Args(sqlStr, arg)
	builder.WhereCb(s.builder, func(wb *builder.Where) {
		if opType == internal.SELECT_OR {
			wb.Or(sqlStr, args...)
		} else {
			wb.And(sqlStr, args...)
		}
	})
	return s
}

// SetRightLike 设置右模糊查询, 如: xxx LIKE "test%"
func (s *SqlStrObj) SetRightLike(fieldName string, val string) *SqlStrObj {
	s.setWhere(internal.SELECT_AND, fieldName, "LIKE", builder.EscapeLike(val)+"%")
	return s
}

// SetLeftLike 设置左模糊查询, 如: xxx LIKE "%test"
func (s *SqlStrObj) SetLeftLike(fieldName string, val string) *SqlStrObj {
	s.setWhere(internal.SELECT_AND, fieldName, "LIKE", "%"+builder.EscapeLike(val))
	return s
}

// SetAllLike 设置全模糊, 如: xxx LIKE "%test%"
func (s *SqlStrObj) SetAllLike(fieldName string, val string) *SqlStrObj {
	s.setWhere(internal.SELECT_AND, fieldName, "LIKE", "%"+builder.EscapeLike(val)+"%")
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
	builder.WhereCb(s.builder, func(wb *builder.Where) {
		wb.And(" "+sqlStr, args...)
	})
	return s
}

// SetOrWhereArgs 支持占位符
// 如: SetOrWhereArgs("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (s *SqlStrObj) SetOrWhereArgs(sqlStr string, args ...interface{}) *SqlStrObj {
	builder.WhereCb(s.builder, func(wb *builder.Where) {
		wb.Or(" "+sqlStr, args...)
	})
	return s
}

// SetOrderByStr 设置排序
func (s *SqlStrObj) SetOrderByStr(orderByStr string) *SqlStrObj {
	if orderByStr == "" {
		return s
	}
	if strings.HasSuffix(strings.ToUpper(orderByStr), "DESC") {
		s.getSelectBuilder().OrderByDesc(orderByStr)
	} else {
		s.getSelectBuilder().OrderByAsc(orderByStr)
	}
	return s
}

// GetOffset 根据分页获取 offset
// 注: page, size 只支持 int 系列类型
func (s *SqlStrObj) GetOffset(page, size interface{}) (int64, int64) {
	return utils.GetOffset(page, size)
}

// SetLimit 设置分页
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func (s *SqlStrObj) SetLimit(page, size interface{}) *SqlStrObj {
	s.getSelectBuilder().Limit(page, size)
	return s
}

// SetGroupByStr 设置 groupBy
func (s *SqlStrObj) SetGroupByStr(groupByStr string) *SqlStrObj {
	s.getSelectBuilder().GroupBy(groupByStr)
	return s
}

// SetHaving 设置 Having
func (s *SqlStrObj) SetHaving(having string, args ...interface{}) *SqlStrObj {
	s.getSelectBuilder().Having(having, args...)
	return s
}
