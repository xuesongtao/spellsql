package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/dialect"
)

type Builder interface {
	AppendSql2Args(s string, args ...interface{}) // AppendSql2Args 追加 SQL 语句和参数, 用于拼接 SQL 语句
	GetNoParseSql2Args() (string, []interface{})  // GetNoParseSql2Args 保留输入的占位符 SQL 语句和参数, spellsql 内部使用
	GetSqlStr() string                            // GetSqlStr 解析输入占位符后的 SQL 语句, 用于打印日志
	GetSql2Args() (string, []interface{})         // GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
}

type builder struct {
	dbType    dialect.DbType
	finalSql  strings.Builder
	finalArgs []interface{}
	genFinal  func()

	// 用于 AppendSql2Args
	extSql  []string
	extArgs []interface{}
}

func newBuilder(dt dialect.DbType) *builder {
	return &builder{
		dbType: dt,
	}
}

func (b *builder) setGenFinal(f func()) {
	b.genFinal = f
}

func (b *builder) appendSql2Args(s string, args ...interface{}) {
	b.appendSql(s)
	b.appendArgs(args...)
}

func (b *builder) len() int {
	return b.finalSql.Len()
}

func (b *builder) empty() bool {
	return b.len() == 0
}

func (b *builder) appendSql(s string) {
	b.finalSql.WriteString(s)
}

func (b *builder) appendArgs(args ...interface{}) {
	if b.finalArgs == nil {
		b.finalArgs = make([]interface{}, 0, len(args)*2)
	}
	b.finalArgs = append(b.finalArgs, args...)
}

func (b *builder) getFinalSql2Args() (string, []interface{}) {
	if b.genFinal != nil {
		b.genFinal()
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

func (b *builder) AppendSql2Args(sqlStr string, args ...interface{}) {
	if b.extArgs == nil {
		b.extArgs = make([]interface{}, 0, len(args))
	}
	b.extSql = append(b.extSql, " "+sqlStr)
	b.extArgs = append(b.extArgs, args...)
}

func (b *builder) GetNoParseSql2Args() (string, []interface{}) {
	return b.getFinalSql2Args()
}

func (b *builder) GetSqlStr() string {
	sqlStr, sqlArgs := b.getFinalSql2Args()
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Parse().Result()
}

func (b *builder) GetSql2Args() (string, []interface{}) {
	sqlStr, sqlArgs := b.getFinalSql2Args()
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Replace().Result(), sqlArgs
}
