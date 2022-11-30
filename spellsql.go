package spellsql

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	// "github.com/gogf/gf/os/glog"
)

const (
	// sql 操作数字
	_ uint8 = iota
	INSERT
	DELETE
	SELECT
	UPDATE

	// sql LIKE 语句
	ALK // 全模糊 如: xxx LIKE "%xxx%"
	RLK // 右模糊 如: xxx LIKE "xxx%"
	LLK // 左模糊 如: xxx LIKE "%xxx"

	// sql join 语句
	LJI // 左连接
	RJI // 右连接
)

var (
	sqlSyncPool = sync.Pool{New: func() interface{} { return new(SqlStrObj) }} // 考虑到性能问题, 这里用 pool
)

// SqlStrObj 拼接 sql 对象
type SqlStrObj struct {
	hasWhereStr     bool  // 标记 SELECT/UPDATE/DELETE 是否添加已添加 WHERE
	hasValuesStr    bool  // 标记 INSERT 是否添加已添加 VALUES
	hasSetStr       bool  // 标记 UPDATE 是否添加 SET
	isPutPooled     bool  // 标记是否已被回收了
	needAddJoinStr  bool  // 标记初始化后, WHERE 后再新加的值时是否需要添加 AND/OR
	needAddComma    bool  // 标记初始化后, UPDATE/INSERT 再添加的值是是否需要添加 ","
	needAddBracket  bool  // 标记 INSERT 时, 是否添加括号
	isPrintSqlLog   bool  // 标记是否打印 生成的 sqlStr log
	isCallCacheInit bool  // 标记是否为 NewCacheSql 初始化生产的对象
	actionNum       uint8 // INSERT/DELETE/SELECT/UPDATE
	callerSkip      uint8 // 跳过调用栈的数
	limitStr        string
	orderByStr      string
	groupByStr      string
	buf             strings.Builder // 记录最终 sqlStr
	whereBuf        strings.Builder // 记录 WHERE 条件
	valuesBuf       strings.Builder // 记录 INSERT/UPDATE 设置的值
	extBuf          strings.Builder // 追加到最后, 辅助字段
}

// NewCacheSql 初始化, 支持占位符, 此函数比 NewSql 更加高效(有缓存)
//
// 1. 注意:
// 		a. 此函数只支持调用一次 GetSqlStr 方法, 如果要调用多次需要使用 NewSql
// 		b. 此函数不支持 Clone 方法, 如果要使用 Clone 需要调用 NewSql
//      说明: 是防止同一对象被两个协程共同使用
//
// 2. 占位符为: ?, 直接根据 args 中类型来自动推动 arg 的类型
//      第一种用法: 根据 args 中类型来自动推动 arg 的类型
//      如: NewCacheSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?", "test", 123)
//      => SELECT username, password FROM sys_user WHERE username = "test" AND password = 123
//
// 		第二种用法: 当 arg 为 []int8/int 等
// 		如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?)", []int{1, 2, 3})
// 		=> SELECT username, password FROM sys_user WHERE id IN (1,2,3)
//
// 3. 占位符为: ?d, 只会把数字型的字符串转为数字型, 如果是字母的话会被转义为 0, 如: "123" => 123; []string{"1", "2", "3"} => 1,2,3
// 		第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个
// 		如: NewCacheSql("SELECT username, password FROM sys_user WHERE id = ?d", "123")
// 		=> SELECT username, password FROM sys_user WHERE id = 123
//
//      第二种用法: 当 arg 为 []string, 又想把解析后的单个元素不加引号
// 		如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?d)", []string{"1", "2", "3"})
// 		=> SELECT username, password FROM sys_user WHERE id IN (1,2,3)
//
// 4. 占位符为: ?v, 这样会让字符串类型不加引号, 原样输出, 如: "test" => test;
// 		第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个, 注: 只支持 arg 为字符串类型
// 		如: NewCacheSql("SELECT username, password FROM ?v WHERE id = ?d", "sys_user", "123")
// 		=> SELECT username, password FROM sys_user WHERE id = 123
func NewCacheSql(sqlStr string, args ...interface{}) *SqlStrObj {
	obj := sqlSyncPool.Get().(*SqlStrObj)
	obj.initSql(sqlStr, args...)
	obj.isCallCacheInit = true
	return obj
}

// NewSql 此函数与 NewCacheSql 功能一样, 此函数的使用场景: 1. 需要调用多次 GetSqlStr; 2. 需要调用 Clone
func NewSql(sqlStr string, args ...interface{}) *SqlStrObj {
	obj := new(SqlStrObj)
	obj.initSql(sqlStr, args...)
	return obj
}

// initSql 初始化需要的 buf
func (s *SqlStrObj) initSql(sqlStr string, args ...interface{}) {
	sqlLen := len(sqlStr)

	// INSERT, DELETE, SELECT, UPDATE
	if sqlLen > 6 { // 判断是什么操作
		actionStr := sqlStr[:6]
		upperStr := toUpper(actionStr)
		switch upperStr {
		case "INSERT", "REPLAC":
			s.actionNum = INSERT
		case "DELETE":
			s.actionNum = DELETE
		case "SELECT":
			s.actionNum = SELECT
		case "UPDATE":
			s.actionNum = UPDATE
		}
	}

	s.init()
	if sqlLen < 2<<8 {
		s.buf.Grow(sqlLen * 2)
		s.whereBuf.Grow(sqlLen)
		if s.actionNum == INSERT || s.actionNum == UPDATE {
			s.valuesBuf.Grow(sqlLen)
		}
	} else {
		s.buf.Grow(sqlLen)
		s.whereBuf.Grow(sqlLen / 2)
		if s.actionNum == INSERT || s.actionNum == UPDATE {
			s.valuesBuf.Grow(sqlLen / 2)
		}
	}

	initWhereFn := func() {
		whereIndex := getTargetIndex(sqlStr, "WHERE")
		if whereIndex > -1 {
			s.hasWhereStr = true

			// 判断下是否需要自动添加 AND, 判断条件:
			// 1. 获取到 WHERE 位置 index, 通过计算: sqlLen - (index + 5) > 3 来判断是否后面有值, 3个字符假定为: 字段名, 操作符, 值
			// 2. 如果表达式为 true, 标记 needAddJoinStr = true, 反之 false
			s.needAddJoinStr = sqlLen-whereIndex > 3+5
		}
	}

	if s.actionNum == SELECT {
		initWhereFn()
	}

	if s.actionNum == UPDATE {
		setIndex := getTargetIndex(sqlStr, "SET")
		if setIndex > -1 {
			s.hasSetStr = true
			s.needAddComma = sqlLen-setIndex > 3+3

			// 如果有 SET 需要判断下是否包含 WHERE
			initWhereFn()
		}
	}

	if s.actionNum == DELETE {
		initWhereFn()
	}

	if s.actionNum == INSERT {
		s.hasValuesStr = getTargetIndex(sqlStr, "VALUE") > -1
	}
	s.writeSqlStr2Buf(&s.buf, sqlStr, args...)
}

// init 初始化标记, 防止从 pool 里申请的标记已有内容
func (s *SqlStrObj) init() {
	s.hasWhereStr = false
	s.hasValuesStr = false
	s.hasSetStr = false
	s.isPutPooled = false
	s.needAddJoinStr = false
	s.needAddComma = false
	s.isCallCacheInit = false
	s.needAddBracket = false
	s.callerSkip = 1

	// 默认打印 log
	s.isPrintSqlLog = true
}

// initWhere 初始化where, joinStr 为 AND/OR 连接符, 默认时 AND
func (s *SqlStrObj) initWhere(joinStr ...string) {
	defaultJoinStr := " AND"
	if len(joinStr) > 0 {
		// 这里为 " OR"
		defaultJoinStr = " " + joinStr[0]
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

// SetPrintLog 设置是否打印 sqlStr log
func (s *SqlStrObj) SetPrintLog(isPrint bool) *SqlStrObj {
	s.isPrintSqlLog = isPrint
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

// SetOrderByStr 设置排序
func (s *SqlStrObj) SetOrderByStr(orderByStr string) *SqlStrObj {
	s.orderByStr = " ORDER BY " + orderByStr
	return s
}

// SetLimit 设置分页
// page 从 1 开始
func (s *SqlStrObj) SetLimit(page, size int32) *SqlStrObj {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size
	s.limitStr = " LIMIT " + s.Int2Str(int64(offset)) + ", " + s.Int2Str(int64(size))
	return s
}

// SetLimitStr 字符串来设置
func (s *SqlStrObj) SetLimitStr(limitStr string) *SqlStrObj {
	s.limitStr = " LIMIT " + limitStr
	return s
}

// LimitIsEmpty 是否添加 limit
func (s *SqlStrObj) LimitIsEmpty() bool {
	return s.limitStr == ""
}

// SetGroupByStr 设置 groupBy
func (s *SqlStrObj) SetGroupByStr(groupByStr string) *SqlStrObj {
	s.groupByStr = " GROUP BY " + groupByStr
	return s
}

// Having 设置 Having
func (s *SqlStrObj) SetHaving(having string, args ...interface{}) *SqlStrObj {
	tmpBuf := new(strings.Builder)
	s.writeSqlStr2Buf(tmpBuf, having, args...)
	s.groupByStr += " HAVING " + tmpBuf.String()
	return s
}

// Append 将类型追加在最后
func (s *SqlStrObj) Append(sqlStr string, args ...interface{}) *SqlStrObj {
	s.writeSqlStr2Buf(&s.extBuf, " "+sqlStr, args...)
	return s
}

// Clone 克隆对象. 注意: 如果是 NewCacheSql 初始化将返回 nil, 需要采用 NewSql 进行初始化
func (s *SqlStrObj) Clone() *SqlStrObj {
	if s.isCallCacheInit {
		return nil
	}
	return NewSql(s.buf.String())
}

// writeSqlStr2Buf 处理输入为 sqlStr 的
func (s *SqlStrObj) writeSqlStr2Buf(buf *strings.Builder, sqlStr string, args ...interface{}) {
	argLen := len(args)
	if argLen == 0 {
		buf.WriteString(sqlStr)
		return
	}

	sqlLen := len(sqlStr)
	argIndex := -1
	for i := 0; i < sqlLen; i++ {
		v := sqlStr[i]
		if v != '?' {
			buf.WriteByte(v)
			continue
		}
		argIndex++

		// 如果参数不够的话就不进行处理
		if argIndex > argLen-1 {
			buf.WriteByte(v)
			continue
		}

		switch val := args[argIndex].(type) {
		case string:
			// 如果占位符?在最后一位时, 就不往下执行了防止 panic
			if i >= sqlLen-1 {
				buf.WriteByte('"')
				buf.WriteString(s.toEscape(val, false))
				buf.WriteByte('"')
				break
			}

			// 判断下如果为 ?d 字符的话, 这里不需要加引号
			// 如果包含字母的话, 就转为 0, 防止数字型注入
			if sqlStr[i+1] == 'd' {
				buf.WriteString(s.toEscape(val, true))
				i++
			} else if sqlStr[i+1] == 'v' { // 原样输出
				buf.WriteString(val)
				i++
			} else {
				buf.WriteByte('"')
				buf.WriteString(s.toEscape(val, false))
				buf.WriteByte('"')
			}
		case []string:
			lastIndex := len(val) - 1
			// 判断下是否加引号
			isAdd := true
			// 这里必须小于最后一个最后一值才行
			if i < sqlLen-1 {
				if sqlStr[i+1] == 'd' {
					isAdd = false
					i++
				}
				for i1 := 0; i1 <= lastIndex; i1++ {
					if isAdd {
						buf.WriteByte('"')
					}
					buf.WriteString(s.toEscape(val[i1], !isAdd))
					if isAdd {
						buf.WriteByte('"')
					}
					if i1 < lastIndex {
						buf.WriteByte(',')
					}
				}
			} else {
				// 最后一个占位符
				for i1 := 0; i1 <= lastIndex; i1++ {
					buf.WriteByte('"')
					buf.WriteString(s.toEscape(val[i1], false))
					buf.WriteByte('"')
					if i1 < lastIndex {
						buf.WriteByte(',')
					}
				}
			}
		case []byte:
			buf.WriteString("\"" + s.toEscape(string(val), false) + "\"")
		case int:
			buf.WriteString(s.Int2Str(int64(val)))
		case int32:
			buf.WriteString(s.Int2Str(int64(val)))
		case uint:
			buf.WriteString(s.UInt2Str(uint64(val)))
		case uint32:
			buf.WriteString(s.UInt2Str(uint64(val)))
		case []int:
			lastIndex := len(val) - 1
			for i1 := 0; i1 <= lastIndex; i1++ {
				buf.WriteString(s.Int2Str(int64(val[i1])))
				if i1 < lastIndex {
					buf.WriteByte(',')
				}
			}
		case []int32:
			lastIndex := len(val) - 1
			for i1 := 0; i1 <= lastIndex; i1++ {
				buf.WriteString(s.Int2Str(int64(val[i1])))
				if i1 < lastIndex {
					buf.WriteByte(',')
				}
			}
		default:
			// 不常用的走慢处理
			reflectValue := reflect.ValueOf(val)
			switch reflectValue.Kind() {
			case reflect.Slice, reflect.Array: // 这里不会有 []string, 不需要处理符号, 所以直接处理即可
				lastIndex := reflectValue.Len() - 1
				for i1 := 0; i1 <= lastIndex; i1++ {
					buf.WriteString(Str(reflectValue.Index(i1).Interface()))
					if i1 < lastIndex {
						buf.WriteByte(',')
					}
				}
			case reflect.Float32, reflect.Float64:
				buf.WriteString(Str(reflectValue.Float()))
			case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
				buf.WriteString(Str(reflectValue.Int()))
			case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
				buf.WriteString(Str(reflectValue.Uint()))
			default:
				buf.WriteString("undefined")
			}
		}
	}
}

// toEscape 转义
func (s *SqlStrObj) toEscape(val string, is2Num bool) string {
	pos := 0
	vLen := len(val)

	// 有可能有中文, 所以这里用 rune
	buf := make([]rune, vLen*2)
	for _, v := range val {
		switch v {
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
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
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			// 这里需要判断下在占位符: ?d 时是否包含字母, 如果有的话就转为 0, 防止数字型注入
			if is2Num && ((v >= 'A' && v <= 'Z') || (v >= 'a' && v <= 'z')) {
				v = '0'
			}
			buf[pos] = v
			pos++
		}
	}
	return string(buf[:pos])
}

// free 释放
func (s *SqlStrObj) free(isNeedPutPool bool) {
	s.valuesBuf.Reset()
	s.whereBuf.Reset()
	s.extBuf.Reset()
	s.groupByStr = ""
	s.orderByStr = ""
	s.limitStr = ""

	// 不需要 Put 的条件如下:
	// 1. 不需要Put的
	// 2. 只有调用 NewCacheSql 获取的对象才进行 Put
	// 3. 如果已经Put过了
	if !isNeedPutPool {
		return
	}

	if !s.isCallCacheInit {
		return
	}

	if s.isPutPooled {
		return
	}

	// 重置 buf
	s.buf.Reset()
	sqlSyncPool.Put(s)
	s.isPutPooled = true
}

// mergeSql 合并 sql
func (s *SqlStrObj) mergeSql() {
	defer s.buf.WriteString(s.extBuf.String())

	if s.actionNum == INSERT {
		s.buf.WriteString(s.valuesBuf.String())
		return
	}

	if s.actionNum == UPDATE {
		s.buf.WriteString(s.valuesBuf.String())
	}

	// UPDATE, SELECT, DELETE 都会走这里
	s.buf.WriteString(s.whereBuf.String())

	if s.actionNum == SELECT {
		s.buf.WriteString(s.groupByStr)
	}

	if s.actionNum == DELETE || s.actionNum == SELECT || s.actionNum == UPDATE {
		s.buf.WriteString(s.orderByStr)
		s.buf.WriteString(s.limitStr)
	}
}

// SqlIsEmpty sql 是否为空
func (s *SqlStrObj) SqlIsEmpty() bool {
	return s.SqlStrLen() == 0
}

// SqlStrLen sql 的总长度
func (s *SqlStrObj) SqlStrLen() int {
	return s.buf.Len()
}

// SetCallerSkip 设置打印调用跳过的层数
func (s *SqlStrObj) SetCallerSkip(skip uint8) *SqlStrObj {
	s.callerSkip = skip
	return s
}

// GetSqlStr 获取最终 sqlStr, 默认打印 sqlStr, title[0] 为打印 log 的标题; title[1] 为 sqlStr 的结束符, 默认为 ";"
// 注意: 通过 NewCacheSql 初始化对象的只能调用一次此函数, 因为调用后会清空所有buf; 通过 NewSql 初始化对象的可以调用多次此函数
func (s *SqlStrObj) GetSqlStr(title ...string) (sqlStr string) {
	defer s.free(true)
	s.mergeSql()

	argsLen := len(title)
	// sqlStr 的结束符, 默认为 ";"
	endMarkStr := ";"
	if argsLen > 1 { // 第二个参数为内部使用参数, 主要用于不加结束符
		if title[1] == "" {
			endMarkStr = ""
		}
	}

	sqlStr = s.buf.String() + endMarkStr
	if s.isPrintSqlLog {
		var finalTitle string
		_, file, line, ok := runtime.Caller(int(s.callerSkip))
		if ok {
			finalTitle += "(" + parseFileName(file) + ":" + s.Int2Str(int64(line)) + ") "
		}
		sqlStrTitle := "sqlStr"
		if argsLen > 0 {
			sqlStrTitle = title[0]
		}
		finalTitle += sqlStrTitle
		cjLog.Info(finalTitle+":", sqlStr)
		// glog.Info(finalTitle+":", sqlStr)
	}
	return
}

// GetTotalSqlStr 将查询条件替换为 COUNT(*), 默认打印 sqlStr, title 为打印 log 的标题, 对外只支持一个参数, 多传没有用
func (s *SqlStrObj) GetTotalSqlStr(title ...string) (findSqlStr string) {
	if s.actionNum != SELECT {
		return
	}
	defer s.free(false)
	s.mergeSql()
	sqlStr := s.buf.String()
	bufLen := s.buf.Len()

	tmpBuf := new(strings.Builder)
	tmpBuf.Grow(bufLen)
	isAddCountStr := false // 标记是否添加 COUNT(*)
	isAppend := false      // 标记是否直接添加
	for i := 0; i < bufLen; i++ {
		v := sqlStr[i]

		// 直接添加, 如果为 true 就不向下执行了
		if isAppend {
			tmpBuf.WriteByte(v)
			continue
		}

		if i < 6 { // SELECT/select
			tmpBuf.WriteByte(v)
			continue
		}

		if !isAddCountStr {
			tmpBuf.WriteString(" COUNT(*) ")
			isAddCountStr = true
		}

		// 判断遇到第一个 FROM 就直接将后面所有 sql 追加到 tmpBuf
		if v == 'f' || v == 'F' {
			formStr := sqlStr[i : i+4]
			if formStr == "FROM" || formStr == "from" {
				tmpBuf.WriteByte(v)
				isAppend = true
			}
		}
	}
	findSqlStr = tmpBuf.String() + ";"
	if s.isPrintSqlLog {
		var finalTitle string
		_, file, line, ok := runtime.Caller(int(s.callerSkip))
		if ok {
			finalTitle += "(" + parseFileName(file) + ":" + s.Int2Str(int64(line)) + ") "
		}
		sqlStrTitle := "sqlTotalStr"
		if len(title) > 0 {
			sqlStrTitle = title[0]
		}
		finalTitle += sqlStrTitle
		cjLog.Info(finalTitle+":", findSqlStr)
		// glog.Info(finalTitle+":", findSqlStr)
	}
	return
}

// Int2Str 数字转字符串
func (s *SqlStrObj) Int2Str(num int64) string {
	return strconv.FormatInt(num, 10)
}

// UInt2Str
func (s *SqlStrObj) UInt2Str(num uint64) string {
	return strconv.FormatUint(num, 10)
}

// getTargetIndex 忽略大小写
func getTargetIndex(sqlStr, targetStr string, isFont2End ...bool) int {
	is := false // 默认后往前
	if len(isFont2End) > 0 {
		is = isFont2End[0]
	}
	tmpIndex := IndexForBF(is, sqlStr, targetStr)
	if tmpIndex == -1 {
		tmpIndex = IndexForBF(is, sqlStr, toLower(targetStr))
	}
	return tmpIndex
}

// toUpper
func toUpper(str string) string {
	strByte := []byte(str)
	l := len(strByte)
	for i := 0; i < l; i++ {
		strByte[i] &= '_'
	}
	return string(strByte)
}

// toLower
func toLower(str string) string {
	strByte := []byte(str)
	l := len(strByte)
	for i := 0; i < l; i++ {
		strByte[i] |= ' '
	}
	return string(strByte)
}

// IndexForBF 查找, 通过 BF 算法来获取匹配的 index
// isFont2End 是否从主串前向后遍历查找
// 如果匹配的内容靠前建议 isFont2End=true, 反之 false
// TODO 暂不支持中文
func IndexForBF(isFont2End bool, s, substr string) int {
	substrLen := len(substr)
	sLen := len(s)
	switch {
	case sLen == 0 || substrLen == 0:
		return 0
	case substrLen > sLen:
		return -1
	}

	if isFont2End {
		for i := 0; i <= sLen-substrLen; i++ {
			for j := 0; j < substrLen; j++ {
				mainStr := s[i+j]
				sonStr := substr[j]
				if mainStr != sonStr {
					break
				}
				// 如果 j 为最后一个值的话说明全匹配
				if j == substrLen-1 {
					return i
				}
			}
		}
		return -1
	}

	for i := sLen - 1; i >= 0; i-- {
		for j := substrLen - 1; j >= 0; j-- {
			mainStr := s[i]
			sonStr := substr[j]
			if mainStr != sonStr {
				break
			}
			// 如果 j 为最后一个值的话说明全匹配
			if j == 0 {
				return i
			}

			// 如果匹配到最开头的字符时 i=0, 如果 i--, i 为负数, s[i] 会 panic
			if i > 0 {
				i--
			}
		}
	}
	return -1
}

// parseFileName 解析文件名
func parseFileName(filePath string) string {
	sysSplit := "/"
	if runtime.GOOS == "windows" {
		sysSplit = "\\"
	}
	lastIndex := IndexForBF(false, filePath, sysSplit)
	if lastIndex == -1 {
		return ""
	}
	return filePath[lastIndex+1:]
}

// Str 将内容转为 string
func Str(src interface{}) string {
	if src == nil {
		return ""
	}

	switch value := src.(type) {
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.Itoa(int(value))
	case int16:
		return strconv.Itoa(int(value))
	case int32:
		return strconv.Itoa(int(value))
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// DistinctIdsStr 将输入拼接 id 参数按照指定字符进行去重, 如:
// DistinctIdsStr("12345,123,20,123,20,15", ",")
// => 12345,123,20,15
func DistinctIdsStr(s string, split string) string {
	strLen := len(s)
	if strLen == 0 {
		return s
	}

	distinctMap := make(map[string]string, strLen/2)
	sortSlice := make([]string, 0, strLen/2) // 用于保证输出顺序
	saveFunc := func(val string) {
		val = strings.Trim(val, " ")
		if _, ok := distinctMap[val]; !ok {
			distinctMap[val] = val
			sortSlice = append(sortSlice, val)
		}
	}

	for {
		index := IndexForBF(true, s, split)
		if index < 0 {
			// 这里需要处理最后一个字符
			saveFunc(s)
			break
		}
		saveFunc(s[:index])
		s = s[index+1:]

		// 这样可以防止最后一位为 split 字符, 到时就会出现一个空
		if s == "" {
			break
		}
	}
	buf := new(strings.Builder)
	buf.Grow(strLen / 2)
	lastIndex := len(sortSlice) - 1
	for index, val := range sortSlice {
		v := distinctMap[val]
		if index < lastIndex {
			buf.WriteString(v + split)
		} else {
			buf.WriteString(v)
		}
	}
	return buf.String()
}

// DistinctIds 去重
func DistinctIds(ids []string) []string {
	tmp := make(map[string]struct{}, len(ids))
	res := make([]string, 0, len(ids))

	for _, id := range ids {
		if _, ok := tmp[id]; !ok {
			tmp[id] = struct{}{}
			res = append(res, id)
		}
	}
	return res
}

// ========================================= 以下为常用操作的封装 ==================================

// GetSqlStr 适用直接获取 sqlStr, 每次会自动打印日志
func GetSqlStr(sqlStr string, args ...interface{}) string {
	return NewCacheSql(sqlStr, args...).SetCallerSkip(2).GetSqlStr()
}

// FmtSqlStr 适用直接获取 sqlStr, 不会打印日志
func FmtSqlStr(sqlStr string, args ...interface{}) string {
	return NewCacheSql(sqlStr, args...).SetCallerSkip(2).SetPrintLog(false).GetSqlStr("sqlStr", "")
}

// GetLikeSqlStr 针对 LIKE 语句, 只有一个条件
// 如: obj := GetLikeSqlStr(ALK, "SELECT id, username FROM sys_user", "name", "xue")
//     => SELECT id, username FROM sys_user WHERE name LIKE "%xue%"
func GetLikeSqlStr(likeType uint8, sqlStr, fieldName, value string, printLog ...bool) string {
	sqlObj := NewCacheSql(sqlStr)
	switch likeType {
	case ALK:
		sqlObj.SetAllLike(fieldName, value)
	case RLK:
		sqlObj.SetRightLike(fieldName, value)
	case LLK:
		sqlObj.SetLeftLike(fieldName, value)
	}
	isPrintLog := false
	endSymbol := ""

	// 判断下是否打印 log
	if len(printLog) > 0 {
		isPrintLog = true
		endSymbol = ";"
	}
	return sqlObj.SetPrintLog(isPrintLog).SetCallerSkip(2).GetSqlStr("sqlStr", endSymbol)
}
