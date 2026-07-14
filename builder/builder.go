package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type SQLBuilder interface {
	InitSql2Args(s string, args ...interface{}) *Builder   // InitSql2Args 初始化 SQL 语句和参数, 用于拼接 SQL 语句
	AppendSql2Args(s string, args ...interface{}) *Builder // AppendSql2Args 追加 SQL 语句和参数, 用于拼接 SQL 语句
	GetNoParseSql2Args() (string, []interface{})           // GetNoParseSql2Args 保留输入的占位符 SQL 语句和参数, spellsql 内部使用
	GetSqlStr() string                                     // GetSqlStr 解析输入占位符后的 SQL 语句, 建议用于打印日志
	GetSql2Args() (string, []interface{})                  // GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
}

type Builder struct {
	dbType    dialect.DbType
	finalSql  strings.Builder
	finalArgs []interface{}
	genFinal  func(b *Builder)

	// 用于 AppendSql2Args
	extSql  strings.Builder
	extArgs []interface{}
}

func NewBuilder(dt ...dialect.DbType) *Builder {
	obj := new(Builder)
	obj.init(dt...)
	return obj
}

func (b *Builder) init(dt ...dialect.DbType) {
	b.dbType = getDbType(dt...)
	b.finalSql.Reset()
	b.finalArgs = nil
	b.genFinal = nil

	b.extSql.Reset()
	b.extArgs = nil
}

func (b *Builder) setGenFinal(f func(b *Builder)) {
	b.genFinal = f
}

func (b *Builder) appendSql2Args(s string, args ...interface{}) {
	b.appendSql(s)
	b.appendArgs(args...)
}

func (b *Builder) len() int {
	return b.finalSql.Len()
}

func (b *Builder) empty() bool {
	return b.len() == 0
}

func (b *Builder) appendSql(s string) {
	b.finalSql.WriteString(s)
}

func (b *Builder) appendArgs(args ...interface{}) {
	if b.finalArgs == nil {
		b.finalArgs = make([]interface{}, 0, len(args)*2)
	}
	b.finalArgs = append(b.finalArgs, args...)
}

func (b *Builder) getFinalNoPraseSql2Args() (string, []interface{}) {
	if b.genFinal != nil {
		b.genFinal(b)
		b.genFinal = nil
	}

	if b.extSql.Len() > 0 {
		b.finalSql.WriteString(b.extSql.String())
		b.extSql.Reset()
	}

	if len(b.extArgs) > 0 {
		b.finalArgs = append(b.finalArgs, b.extArgs...)
		b.extArgs = nil
	}
	return b.finalSql.String(), b.finalArgs
}

// GetNoParseSql 获取保留输入的占位符 SQL 语句
func (b *Builder) GetNoParseSql() string {
	sqlStr, _ := b.getFinalNoPraseSql2Args()
	return sqlStr
}

// GetNoParseArgs 获取保留输入的占位符 SQL 参数
func (b *Builder) GetNoParseArgs() []interface{} {
	_, args := b.getFinalNoPraseSql2Args()
	return args
}

func (b *Builder) HaveStr(field string) bool {
	return utils.Index(strings.ToUpper(b.finalSql.String()), strings.ToUpper(field), false) > -1
}

func (b *Builder) whereStrIndex() int {
	return utils.Index(strings.ToUpper(b.finalSql.String()), " WHERE", false)
}

func (b *Builder) initWhere(sqlStr string, args ...interface{}) {
	if i := b.whereStrIndex(); i == -1 {
		b.appendSql(" WHERE ")
	} else if i+5 < b.len()-1 { // "WHERE" 后面还有内容, 需要加上 AND
		b.appendSql(" AND ")
	} else if i+5 == b.len()-1 { // "WHERE" 后面没有内容, 直接追加
		b.appendSql(" ")
	}
	b.appendSql2Args(sqlStr, args...)
}

func (b *Builder) InitSql2Args(sqlStr string, args ...interface{}) *Builder {
	b.appendSql2Args(sqlStr, args...)
	return b
}

func (b *Builder) AppendSql2Args(sqlStr string, args ...interface{}) *Builder {
	if b.extArgs == nil {
		b.extArgs = make([]interface{}, 0, len(args))
	}
	b.extSql.WriteString(" ")
	b.extSql.WriteString(sqlStr)
	b.extArgs = append(b.extArgs, args...)
	return b
}

// GetNoParseSql2Args 保留输入的占位符 SQL 语句和参数, spellsql 内部使用
func (b *Builder) GetNoParseSql2Args() (string, []interface{}) {
	return b.getFinalNoPraseSql2Args()
}

// GetSqlStr 解析输入占位符后的 SQL 语句, 建议用于打印日志
func (b *Builder) GetSqlStr() string {
	sqlStr, sqlArgs := b.getFinalNoPraseSql2Args()
	// fmt.Println(sqlStr, sqlArgs)
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Parse().Result()
}

// GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
func (b *Builder) GetSql2Args() (string, []interface{}) {
	sqlStr, sqlArgs := b.getFinalNoPraseSql2Args()
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Replace().Result(), sqlArgs
}

func getDbType(dt ...dialect.DbType) dialect.DbType {
	if len(dt) > 0 {
		return dt[0]
	}
	return dialect.DefaultDbType
}
