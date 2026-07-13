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
	GetSqlStr() string                                     // GetSqlStr 解析输入占位符后的 SQL 语句, 用于打印日志
	GetSql2Args() (string, []interface{})                  // GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
}

type Builder struct {
	dbType    dialect.DbType
	finalSql  strings.Builder
	finalArgs []interface{}
	genFinal  func(b *Builder)

	// 用于 AppendSql2Args
	extSql  []string
	extArgs []interface{}
}

func NewBuilder(dt ...dialect.DbType) *Builder {
	return &Builder{
		dbType: getDbType(dt...),
	}
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

func (b *Builder) getFinalSql2Args() (string, []interface{}) {
	if b.genFinal != nil {
		b.genFinal(b)
		b.genFinal = nil
	}

	if len(b.extSql) > 0 {
		for _, s := range b.extSql {
			b.finalSql.WriteString(s)
		}
		b.extSql = nil
	}

	if len(b.extArgs) > 0 {
		b.finalArgs = append(b.finalArgs, b.extArgs...)
		b.extArgs = nil
	}
	return b.finalSql.String(), b.finalArgs
}

func (b *Builder) copy() *Builder {
	obj := &Builder{
		dbType:    b.dbType,
		finalSql:  strings.Builder{},
		finalArgs: make([]interface{}, len(b.finalArgs)),
		genFinal:  b.genFinal,
		extSql:    make([]string, len(b.extSql)),
		extArgs:   make([]interface{}, len(b.extArgs)),
	}
	obj.finalSql.WriteString(b.finalSql.String())
	copy(obj.finalArgs, b.finalArgs)
	copy(obj.extSql, b.extSql)
	copy(obj.extArgs, b.extArgs)
	return obj
}

func (b *Builder) haveStr(field string) bool {
	return utils.Index(strings.ToUpper(b.finalSql.String()), strings.ToUpper(field), false) >= 0
}

func (b *Builder) haveWhereStr() bool {
	return b.haveStr("WHERE")
}

func (b *Builder) InitSql2Args(sqlStr string, args ...interface{}) *Builder {
	b.finalSql.Reset()
	b.finalArgs = nil
	b.extSql = nil
	b.extArgs = nil
	b.appendSql2Args(sqlStr, args...)
	return b
}

func (b *Builder) AppendSql2Args(sqlStr string, args ...interface{}) *Builder {
	if b.extArgs == nil {
		b.extArgs = make([]interface{}, 0, len(args))
	}
	b.extSql = append(b.extSql, " "+sqlStr)
	b.extArgs = append(b.extArgs, args...)
	return b
}

func (b *Builder) GetNoParseSql2Args() (string, []interface{}) {
	return b.getFinalSql2Args()
}

func (b *Builder) GetSqlStr() string {
	sqlStr, sqlArgs := b.getFinalSql2Args()
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Parse().Result()
}

func (b *Builder) GetSql2Args() (string, []interface{}) {
	sqlStr, sqlArgs := b.getFinalSql2Args()
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Replace().Result(), sqlArgs
}

func getDbType(dt ...dialect.DbType) dialect.DbType {
	if len(dt) > 0 {
		return dt[0]
	}
	return dialect.DefaultDbType
}
