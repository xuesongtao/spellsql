package dialect

import (
	"context"
	"database/sql"
	"strings"
)

// DBer
type DBer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Dialect 数据库方言接口, 适配不同数据库, 不变的部分
type Dialect interface {
	GetWarpFieldSymbol() string           // 获取字段包裹符号
	GetWarpValueStrSymbol() string        // 获取值为字符串的包裹符号
	GetValueEscapeMap() map[byte][]byte   // 获取值转义规则
	GetLimitSql(limit, offset int) string // 获取 limit sql 语句
}

func Placeholders(n ...int) string {
	nn := 1
	if len(n) > 0 {
		nn = n[0]
	}
	return strings.Repeat("?, ", nn-1) + "?"
}

func WarpField(d Dialect, field string) string {
	if strings.HasPrefix(field, d.GetWarpFieldSymbol()) {
		return field
	}
	return d.GetWarpFieldSymbol() + field + d.GetWarpFieldSymbol()
}

func WarpValue(d Dialect, value string) string {
	if strings.HasPrefix(value, d.GetWarpValueStrSymbol()) {
		return value
	}
	return d.GetWarpValueStrSymbol() + value + d.GetWarpValueStrSymbol()
}

func WarpJoinFields(d Dialect, fields ...string) string {
	result := make([]string, len(fields))
	for i, field := range fields {
		result[i] = WarpField(d, field)
	}
	return strings.Join(result, ", ")
}

// TableMeter 表元信息, 为了适配不同数据库
type TableMeter interface {
	SetTableName(tableName string)                                                                     // 方便框架调用设置 tableName 参数
	GetField2ColInfoMap(ctx context.Context, db DBer, printLog bool) (map[string]*TableColInfo, error) // key: field
}
