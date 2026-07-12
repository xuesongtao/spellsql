package spellsql

import (
	"context"
	"path/filepath"
	"runtime"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

// SqlStrObj 拼接 sql 对象
// Deprecated: 该对象已被废弃, 请使用 builder.SQLBuilder 对象
type SqlStrObj struct {
	ctx             context.Context
	isPutPooled     bool            // 标记是否已被回收了
	isPrintSqlLog   bool            // 标记是否打印 生成的 sqlStr log
	isCallCacheInit bool            // 标记是否为 NewCacheSql 初始化生产的对象
	actionNum       internal.OpType // INSERT/DELETE/SELECT/UPDATE
	callerSkip      uint8           // 跳过调用栈的数
	dbType          dialect.DbType
	builder         builder.SQLBuilder // builder 对象, 用于拼接 sql
}

// NewCacheSql 初始化, 支持占位符, 此函数比 NewSql 更加高效(有缓存)
//
//  1. 注意:
//     a. 此函数只支持调用一次 GetSqlStr 方法, 如果要调用多次需要使用 NewSql
//     b. 此函数不支持 Clone 方法, 如果要使用 Clone 需要调用 NewSql
//     说明: 是防止同一对象被两个协程共同使用
//
//  2. 占位符为: ?, 直接根据 args 中类型来自动推动 arg 的类型
//     第一种用法: 根据 args 中类型来自动推动 arg 的类型
//     如: NewCacheSql("SELECT username, password FROM sys_user WHERE username = ? AND password = ?", "test", 123)
//     => SELECT username, password FROM sys_user WHERE username = "test" AND password = 123
//
//     第二种用法: 当 arg 为 []int8/int 等
//     如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?)", []int{1, 2, 3})
//     => SELECT username, password FROM sys_user WHERE id IN (1,2,3)
//
//  3. 占位符为: ?d, 只会把数字型的字符串转为数字型, 如果是字母的话会被转义为 0, 如: "123" => 123; []string{"1", "2", "3"} => 1,2,3
//     第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个
//     如: NewCacheSql("SELECT username, password FROM sys_user WHERE id = ?d", "123")
//     => SELECT username, password FROM sys_user WHERE id = 123
//
//     第二种用法: 当 arg 为 []string, 又想把解析后的单个元素不加引号
//     如: NewCacheSql("SELECT username, password FROM sys_user WHERE id IN (?d)", []string{"1", "2", "3"})
//     => SELECT username, password FROM sys_user WHERE id IN (1,2,3)
//
//  4. 占位符为: ?v, 这样会让字符串类型不加引号, 原样输出, 如: "test" => test;
//     第一种用法: 当 arg 为字符串时, 又想不加双引号就用这个, 注: 只支持 arg 为字符串类型
//     如: NewCacheSql("SELECT username, password FROM ?v WHERE id = ?d", "sys_user", "123")
//     => SELECT username, password FROM sys_user WHERE id = 123
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

// SetCtx 设置 context
func (s *SqlStrObj) SetCtx(ctx context.Context) *SqlStrObj {
	if ctx == nil {
		return s
	}
	s.ctx = ctx
	return s
}

// initSql 初始化需要的 buf
func (s *SqlStrObj) initSql(sqlStr string, args ...interface{}) {
	s.init()

	// INSERT, DELETE, SELECT, UPDATE
	sqlLen := len(sqlStr)
	if sqlLen > 6 { // 判断是什么操作
		actionStr := sqlStr[:6]
		upperStr := internal.ToUpper(actionStr)
		switch upperStr {
		case "INSERT", "REPLAC":
			s.actionNum = internal.INSERT
			s.builder = builder.NewInsert(s.dbType)
		case "DELETE":
			s.actionNum = internal.DELETE
			s.builder = builder.NewDelete(s.dbType)
		case "SELECT":
			s.actionNum = internal.SELECT
			s.builder = builder.NewSelect(s.dbType)
		case "UPDATE":
			s.actionNum = internal.UPDATE
			s.builder = builder.NewUpdate(s.dbType)
		default:
			s.actionNum = internal.None
			s.builder = builder.NewBuilder(s.dbType)
		}
	}
	s.builder.InitSql2Args(sqlStr, args...)
}

// is
func (s *SqlStrObj) is(op uint8, target ...uint8) bool {
	defaultNum := s.actionNum
	if len(target) > 0 {
		defaultNum = target[0]
	}
	return internal.Equal(op, defaultNum)
}

// init 初始化标记, 防止从 pool 里申请的标记已有内容
func (s *SqlStrObj) init() {
	s.ctx = context.Background()
	s.isPutPooled = false
	s.isCallCacheInit = false
	s.callerSkip = 1
	s.actionNum = internal.None

	// 默认打印 log
	s.isPrintSqlLog = true
}

// free 释放
func (s *SqlStrObj) free(isNeedPutPool bool) {
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
	sqlSyncPool.Put(s)
	s.isPutPooled = true
}

func (s *SqlStrObj) SetDbType(dt dialect.DbType) *SqlStrObj {
	s.dbType = dt
	return s
}

// SetStrSymbol 设置在解析值时字符串符号, 不同的数据库符号不同
// 如: mysql 字符串值可以用 ""或”; pg 字符串值只能用 ”
func (s *SqlStrObj) SetStrSymbol(strSymbol byte) *SqlStrObj {
	return s
}

// SetEscapeMap 设置对值的转义处理
func (s *SqlStrObj) SetEscapeMap(escapeMap map[byte][]byte) *SqlStrObj {
	return s
}

// SetPrintLog 设置是否打印 sqlStr log
func (s *SqlStrObj) SetPrintLog(isPrint bool) *SqlStrObj {
	s.isPrintSqlLog = isPrint
	return s
}

// Append 将类型追加在最后
func (s *SqlStrObj) Append(sqlStr string, args ...interface{}) *SqlStrObj {
	s.builder.AppendSql2Args(sqlStr, args...)
	return s
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

func (s *SqlStrObj) getSqlLogStr(title, sqlStr string) string {
	return s.getLogTitle(title) + sqlStr
}

// GetSqlStr 获取最终 sqlStr, 默认打印 sqlStr, title[0] 为打印 log 的标题; title[1] 为 sqlStr 的结束符, 默认为 ";"
// 注意: 通过 NewCacheSql 初始化对象的只能调用一次此函数, 因为调用后会清空所有buf; 通过 NewSql 初始化对象的可以调用多次此函数
func (s *SqlStrObj) GetSqlStr(title ...string) (sqlStr string) {
	defer s.free(true)

	argsLen := len(title)
	// sqlStr 的结束符, 默认为 ";"
	endMarkStr := ";"
	if argsLen > 1 { // 第二个参数为内部使用参数, 主要用于不加结束符
		if utils.Null(title[1]) {
			endMarkStr = ""
		}
	}

	sqlStr = s.builder.GetSqlStr() + endMarkStr
	if s.isPrintSqlLog {
		defTitle := "sqlStr"
		if argsLen > 0 {
			defTitle = title[0]
		}
		sLog.Info(s.ctx, s.getLogTitle(defTitle)+sqlStr)
	}
	return
}

// GetTotalSqlStr 将查询条件替换为 COUNT(*), 默认打印 sqlStr, title[0] 为打印 log 的标题; title[1] 为 sqlStr 的结束符, 默认为 ";"
func (s *SqlStrObj) GetTotalSqlStr(title ...string) (findSqlStr string) {
	if !s.is(internal.SELECT) {
		return
	}
	defer s.free(false)
	// sqlStr 的结束符, 默认为 ";"
	endMarkStr := ";"
	argsLen := len(title)
	if argsLen > 1 { // 第二个参数为内部使用参数, 主要用于不加结束符
		if utils.Null(title[1]) {
			endMarkStr = ""
		}
	}
	findSqlStr = s.getSelectBuilder().GetTotalSqlStr() + endMarkStr
	if s.isPrintSqlLog {
		defTitle := "sqlTotalStr"
		if argsLen > 0 {
			defTitle = title[0]
		}
		sLog.Info(s.ctx, s.getLogTitle(defTitle)+findSqlStr)
	}
	return
}

// getLogTitle 获取 log title
func (s *SqlStrObj) getLogTitle(title string) (finalTitle string) {
	// 跳过当前
	_, file, line, ok := runtime.Caller(int(s.callerSkip) + 1)
	if ok {
		finalTitle += "(" + filepath.Base(file) + ":" + utils.Int2Str(int64(line)) + ") "
	}
	finalTitle += title + ": "
	return
}

// getTargetIndex 忽略大小写
func getTargetIndex(sqlStr, targetStr string, isFont2End ...bool) int {
	is := false // 默认后往前
	if len(isFont2End) > 0 {
		is = isFont2End[0]
	}
	tmpIndex := utils.Index(sqlStr, targetStr, is)
	if tmpIndex == -1 {
		tmpIndex = utils.Index(sqlStr, internal.ToLower(targetStr), is)
	}
	return tmpIndex
}
