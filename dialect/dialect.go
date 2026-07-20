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
	GetWarpColSymbol() string             // 获取字段包裹符号
	GetWarpValueStrSymbol() string        // 获取值为字符串的包裹符号
	GetValueEscapeMap() map[byte][]byte   // 获取值转义规则
	GetLimitSql(limit, offset int) string // 获取 limit sql 语句
}

// TableMeter 表元信息, 为了适配不同数据库
type TableMeter interface {
	GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) // key: col
	GetDefaultVal(col string, colInfo *TableColInfo) interface{}
}

var (
	dialectMap = map[DbType]Dialect{
		MySQL:    Mysql(),
		Postgres: Pg(),
	}
	tableMeterMap = map[DbType]func() TableMeter{
		MySQL:    func() TableMeter { return Mysql() },
		Postgres: func() TableMeter { return Pg() },
	}
)

func WarpValue(d Dialect, value string) string {
	if strings.HasPrefix(value, d.GetWarpValueStrSymbol()) {
		return value
	}
	return d.GetWarpValueStrSymbol() + value + d.GetWarpValueStrSymbol()
}

func GetTableMeter(dbType DbType) TableMeter {
	fn, ok := tableMeterMap[dbType]
	if ok {
		return fn()
	}
	return tableMeterMap[DefaultDbType]()
}

func GetDialect(dbType DbType) Dialect {
	dialect, ok := dialectMap[dbType]
	if ok {
		return dialect
	}
	return dialectMap[DefaultDbType]
}
