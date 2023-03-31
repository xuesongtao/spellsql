package spellsql

// SetUpdateValue update 语句中, 设置字段值
func (s *SqlStrObj) SetUpdateValue(fieldName string, arg interface{}) *SqlStrObj {
	s.initValues()
	s.writeSqlStr2Buf(&s.valuesBuf, " "+fieldName+" = ?", arg)
	return s
}

// ValueIsEmpty insert/update 中 value 是否为空
func (s *SqlStrObj) ValueIsEmpty() bool {
	return s.ValueStrLen() == 0
}

// ValueStrLen valueBuf 长度
func (s *SqlStrObj) ValueStrLen() int {
	return s.valuesBuf.Len()
}

// SetUpdateValueArgs 支持占位符
// 如: SetUpdateValueArgs("username = ?, age = ?d", "test", "20")
// => username = "test", age = 20
func (s *SqlStrObj) SetUpdateValueArgs(sqlStr string, arg ...interface{}) *SqlStrObj {
	s.initValues()
	s.writeSqlStr2Buf(&s.valuesBuf, " "+sqlStr, arg...)
	return s
}

// SetInsertValues 批量插入拼接, 如: xxx VALUES (xxx, xxx), (xxx, xxx)
func (s *SqlStrObj) SetInsertValues(args ...interface{}) *SqlStrObj {
	s.initValues()
	l := len(args)
	sqlStr := "("
	if l > 0 {
		sqlStr += "?"
	}
	for i := 1; i < len(args); i++ {
		sqlStr += ", ?"
	}
	sqlStr += ")"
	s.writeSqlStr2Buf(&s.valuesBuf, " "+sqlStr, args...)
	return s
}

// SetInsertValuesArgs 支持占位符, 如 SetInsertValuesArg("(?, ?, ?d)", "test", "12345", "123456") 或 SetInsertValuesArg("?, ?, ?d", "test", "12345", "123456")
// => ("test", "123456", 123456)
// 批量插入拼接, 如: xxx VALUES (xxx, xxx), (xxx, xxx)
func (s *SqlStrObj) SetInsertValuesArgs(sqlStr string, args ...interface{}) *SqlStrObj {
	s.initValues()
	if !s.needAddBracket { // 防止重复处理
		if IndexForBF(true, sqlStr, "(") == -1 && IndexForBF(false, sqlStr, ")") == -1 {
			s.needAddBracket = true
		}
	}

	if s.needAddBracket {
		sqlStr = "(" + sqlStr + ")"
	}
	s.writeSqlStr2Buf(&s.valuesBuf, " "+sqlStr, args...)
	return s
}

// initValues 初始化 valueBuf
func (s *SqlStrObj) initValues() {
	isAddComma := true // 本次默认加逗号
	if s.actionNum == INSERT && !s.hasValuesStr {
		s.valuesBuf.WriteString(" VALUES")
		s.hasValuesStr = true
		isAddComma = false
	}

	if s.actionNum == UPDATE {
		if !s.hasSetStr {
			s.valuesBuf.WriteString(" SET")
			s.hasSetStr = true
			isAddComma = false
		}
		if s.needAddComma {
			// 第一次添加的时候, 判断是否需要添加逗号
			s.valuesBuf.WriteString(",")
			s.needAddComma = false
			isAddComma = false
		}
	}

	if isAddComma {
		// fast past
		// 说明已经设过值了, 这里在前面加个逗号
		if s.ValueStrLen() > 0 {
			s.valuesBuf.WriteString(",")
			return
		}

		if s.actionNum == INSERT {
			// slow path
			// 如果初始化时或已经merge后 sqlStr已经这样了: xxx VALUES (xxx), 我们通过判断 VALUE 的下标是否为最后几个字符, 如果是的话就
			// 不处理, 反之加逗号
			valueIndex := getTargetIndex(s.buf.String(), "VALUE")
			if valueIndex == -1 {
				return
			}

			lastIndex := s.SqlStrLen() - 1

			// 需要跳过本身长度
			if lastIndex-(valueIndex+5) > 5 {
				s.valuesBuf.WriteString(",")
			}
		}
	}
}
