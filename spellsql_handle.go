package spellsql

import (
	"gitee.com/xuesongtao/spellsql/v2/builder"
)

// SetUpdateValue update 语句中, 设置字段值
func (s *SqlStrObj) SetUpdateValue(fieldName string, arg interface{}) *SqlStrObj {
	s.builder.(*builder.Update).Set(fieldName, arg)
	return s
}

// SetInsertValues 批量插入拼接, 如: xxx VALUES (xxx, xxx), (xxx, xxx)
func (s *SqlStrObj) SetInsertValues(args ...interface{}) *SqlStrObj {
	s.builder.(*builder.Insert).Values(args...)
	return s
}
