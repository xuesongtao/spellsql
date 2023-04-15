package spellsql

import (
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

const (
	// sql 操作数字
	none uint8 = iota
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
	strSymbol       byte  // 记录解析字符串值的符号, 默认: ""
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
	s.init()

	// INSERT, DELETE, SELECT, UPDATE
	sqlLen := len(sqlStr)
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

	if sqlLen < 2<<8 {
		s.buf.Grow(sqlLen * 2)
		s.whereBuf.Grow(sqlLen)
		if s.is(INSERT) || s.is(UPDATE) {
			s.valuesBuf.Grow(sqlLen)
		}
	} else {
		s.buf.Grow(sqlLen)
		s.whereBuf.Grow(sqlLen / 2)
		if s.is(INSERT) || s.is(UPDATE) {
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

	if s.is(SELECT) {
		initWhereFn()
	}

	if s.is(UPDATE) {
		setIndex := getTargetIndex(sqlStr, "SET")
		if setIndex > -1 {
			s.hasSetStr = true
			s.needAddComma = sqlLen-setIndex > 3+3

			// 如果有 SET 需要判断下是否包含 WHERE
			initWhereFn()
		}
	}

	if s.is(DELETE) {
		initWhereFn()
	}

	if s.is(INSERT) {
		s.hasValuesStr = getTargetIndex(sqlStr, "VALUE") > -1
	}
	s.writeSqlStr2Buf(&s.buf, sqlStr, args...)
}

// is
func (s *SqlStrObj) is(op uint8, target ...uint8) bool {
	defaultNum := s.actionNum
	if len(target) > 0 {
		defaultNum = target[0]
	}
	return defaultNum == op
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
	s.actionNum = none
	s.strSymbol = '"'

	// 默认打印 log
	s.isPrintSqlLog = true
}

// SetStrSymbol 设置在解析值时字符串符号, 不同的数据库符号不同
// 如: mysql 字符串值可以用 ""或''; pg 字符串值只能用 ''
func (s *SqlStrObj) SetStrSymbol(strSymbol byte) *SqlStrObj {
	if strSymbol != '"' && strSymbol != '\'' {
		return s
	}
	s.strSymbol = strSymbol
	return s
}

// SetPrintLog 设置是否打印 sqlStr log
func (s *SqlStrObj) SetPrintLog(isPrint bool) *SqlStrObj {
	s.isPrintSqlLog = isPrint
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
				buf.WriteByte(s.strSymbol)
				buf.WriteString(s.toEscape(val, false))
				buf.WriteByte(s.strSymbol)
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
				buf.WriteByte(s.strSymbol)
				buf.WriteString(s.toEscape(val, false))
				buf.WriteByte(s.strSymbol)
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
						buf.WriteByte(s.strSymbol)
					}
					buf.WriteString(s.toEscape(val[i1], !isAdd))
					if isAdd {
						buf.WriteByte(s.strSymbol)
					}
					if i1 < lastIndex {
						buf.WriteByte(',')
					}
				}
			} else {
				// 最后一个占位符
				for i1 := 0; i1 <= lastIndex; i1++ {
					buf.WriteByte(s.strSymbol)
					buf.WriteString(s.toEscape(val[i1], false))
					buf.WriteByte(s.strSymbol)
					if i1 < lastIndex {
						buf.WriteByte(',')
					}
				}
			}
		case []byte:
			buf.WriteByte('\'')
			buf.Write(val)
			buf.WriteByte('\'')
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
		case '\t':
			buf[pos] = '\\'
			buf[pos+1] = 't'
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

	if s.is(INSERT) {
		s.buf.WriteString(s.valuesBuf.String())
		return
	}

	if s.is(UPDATE) {
		s.buf.WriteString(s.valuesBuf.String())
	}

	// UPDATE, SELECT, DELETE 都会走这里
	s.buf.WriteString(s.whereBuf.String())

	if s.is(SELECT) {
		s.buf.WriteString(s.groupByStr)
	}

	if s.is(DELETE) || s.is(SELECT) || s.is(UPDATE) {
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

// FmtSql 获取格式化后的 sql
func (s *SqlStrObj) FmtSql() string {
	return s.SetPrintLog(false).GetSqlStr("", "")
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
		if null(title[1]) {
			endMarkStr = ""
		}
	}

	sqlStr = s.buf.String() + endMarkStr
	if s.isPrintSqlLog {
		defTitle := "sqlStr"
		if argsLen > 0 {
			defTitle = title[0]
		}
		sLog.Info(s.getLogTitle(defTitle), sqlStr)
	}
	return
}

// GetTotalSqlStr 将查询条件替换为 COUNT(*), 默认打印 sqlStr, title[0] 为打印 log 的标题; title[1] 为 sqlStr 的结束符, 默认为 ";"
func (s *SqlStrObj) GetTotalSqlStr(title ...string) (findSqlStr string) {
	if !s.is(SELECT) {
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
	// sqlStr 的结束符, 默认为 ";"
	endMarkStr := ";"
	argsLen := len(title)
	if argsLen > 1 { // 第二个参数为内部使用参数, 主要用于不加结束符
		if null(title[1]) {
			endMarkStr = ""
		}
	}
	findSqlStr = tmpBuf.String() + endMarkStr
	if s.isPrintSqlLog {
		defTitle := "sqlTotalStr"
		if argsLen > 0 {
			defTitle = title[0]
		}
		sLog.Info(s.getLogTitle(defTitle), findSqlStr)
	}
	return
}

// getLogTitle 获取 log title
func (s *SqlStrObj) getLogTitle(title string) (finalTitle string) {
	// 跳过当前
	_, file, line, ok := runtime.Caller(int(s.callerSkip) + 1)
	if ok {
		finalTitle += "(" + parseFileName(file) + ":" + s.Int2Str(int64(line)) + ") "
	}
	finalTitle += title + ":"
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
