package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

type SQLBuilder interface {
	InitSql2Args(s string, args ...any) *Builder   // InitSql2Args 初始化 SQL 语句和参数, 用于拼接 SQL 语句
	AppendSql2Args(s string, args ...any) *Builder // AppendSql2Args 追加 SQL 语句和参数, 用于拼接 SQL 语句
	GetNoParseSql2Args() (string, []any)           // GetNoParseSql2Args 保留输入的占位符 SQL 语句和参数, spellsql 内部使用
	GetSqlStr() string                             // GetSqlStr 解析输入占位符后的 SQL 语句, 建议用于打印日志(sql占位符替换为对应的值)
	GetSql2Args() (string, []any)                  // GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
}

type Builder struct {
	dbType     dialect.DbType
	finalSql   strings.Builder
	finalArgs  []any
	genFinalFn func(b *Builder)

	// 用于 AppendSql2Args
	extSql  strings.Builder
	extArgs []any

	callInitSql2Args bool // 标记是否调用 InitSql2Args
}

func NewBuilder(dt ...dialect.DbType) *Builder {
	obj := new(Builder)
	obj.init(dt...)
	return obj
}

func (b *Builder) init(dt ...dialect.DbType) {
	b.dbType = dialect.DefaultDbType
	if len(dt) > 0 {
		b.dbType = dt[0]
	}
}

func (b *Builder) setGenFinal(f func(b *Builder)) {
	b.genFinalFn = f
}

func (b *Builder) writeSql2Args(s string, args ...any) {
	b.writeSql(s)
	b.writeArgs(args...)
}

func (b *Builder) len() int {
	return b.finalSql.Len()
}

func (b *Builder) empty() bool {
	return b.len() == 0
}

func (b *Builder) writeSql(s string) {
	b.finalSql.WriteString(s)
}

func (b *Builder) writeArgs(args ...any) {
	if b.finalArgs == nil {
		b.finalArgs = make([]any, 0, len(args)*2)
	}
	b.finalArgs = append(b.finalArgs, args...)
}

func (b *Builder) getFinalNoPraseSql2Args() (string, []any) {
	if b.genFinalFn != nil {
		b.genFinalFn(b)
		b.genFinalFn = nil
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
func (b *Builder) GetNoParseArgs() []any {
	_, args := b.getFinalNoPraseSql2Args()
	return args
}

func (b *Builder) HaveStr(field string) bool {
	return b.index(field) > -1
}

func (b *Builder) index(field string) int {
	return utils.Index(strings.ToUpper(b.finalSql.String()), field, false)
}

func (b *Builder) mergeWhere(where *Where) {
	sqlStr, args := where.GetNoParseSql2Args()
	if i := b.index(" WHERE"); i == -1 {
		b.writeSql(" WHERE ")
	} else if i+5+2 <= b.len()-1 { // 如: " WHERE x", 需要加 AND
		b.writeSql(" AND ")
	} else if i+5 == b.len()-1 { // " WHERE" 后面没有内容, 直接追加
		b.writeSql(" ")
	}
	b.writeSql2Args(sqlStr, args...)
}

// InitSql2Args 初始化 SQL 语句和参数, 用于拼接 SQL 语句
func (b *Builder) InitSql2Args(sqlStr string, args ...any) *Builder {
	b.callInitSql2Args = true
	b.writeSql2Args(sqlStr, args...)
	return b
}

// AppendSql2Args 追加 SQL 语句和参数, 用于拼接 SQL 语句
func (b *Builder) AppendSql2Args(sqlStr string, args ...any) *Builder {
	if b.extArgs == nil {
		b.extArgs = make([]any, 0, len(args))
	}
	b.extSql.WriteString(" ")
	b.extSql.WriteString(sqlStr)
	b.extArgs = append(b.extArgs, args...)
	return b
}

// GetNoParseSql2Args 保留输入的占位符 SQL 语句和参数, spellsql 内部使用
func (b *Builder) GetNoParseSql2Args() (string, []any) {
	return b.getFinalNoPraseSql2Args()
}

// GetSqlStr 解析输入占位符后的 SQL 语句, 建议用于打印日志
func (b *Builder) GetSqlStr() string {
	sqlStr, sqlArgs := b.getFinalNoPraseSql2Args()
	// fmt.Println(sqlStr, sqlArgs)
	return dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Parse().Result()
}

// GetSql2Args 根据不同数据库, 解析占位符后的 SQL 语句和参数, 用于执行 SQL 语句
func (b *Builder) GetSql2Args() (string, []any) {
	sqlStr, sqlArgs := b.getFinalNoPraseSql2Args()
	// fmt.Println("======> before", sqlStr, sqlArgs)
	pl := dialect.NewParsePlaceholder(b.dbType, sqlStr, sqlArgs...).Replace()
	// fmt.Println("======> after", pl.Result(), pl.Args())
	return pl.Result(), pl.Args()
}

func (b *Builder) warpCol(col string) string {
	gd := dialect.GetDialect(b.dbType)
	if strings.HasPrefix(col, gd.GetWarpColSymbol()) {
		return col
	}
	return gd.GetWarpColSymbol() + col + gd.GetWarpColSymbol()
}

func (b *Builder) warpJoinCols(fields ...string) string {
	buf := internal.GetTmpBuf()
	defer internal.PutTmpBuf(buf)

	for i, field := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(b.warpCol(field))
	}
	return buf.String()
}
