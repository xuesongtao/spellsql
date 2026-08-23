package sqldb

import (
	"context"
	"database/sql"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
)

const (
	PriFlag     = "PRI" // 主键标识
	NotNullFlag = "NO"  // 非空标识
)

var tableMeterMap = map[dialect.DbType]func() TableMeter{
	dialect.MySQL:    func() TableMeter { return &MySQL{} },
	dialect.Postgres: func() TableMeter { return &PgSQL{} },
	dialect.SQLite:   func() TableMeter { return &SQLite{} },
}

func GetTableMeter(dbType dialect.DbType) TableMeter {
	fn, ok := tableMeterMap[dbType]
	if ok {
		return fn()
	}
	return tableMeterMap[dialect.DefaultDbType]()
}


// DBer
type DBer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// TableMeter 表元信息, 为了适配不同数据库
type TableMeter interface {
	GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) // key: col
	GetDefaultVal(col string, colInfo *TableColInfo) internal.RawSql
}

// TableColInfo 表列详情
type TableColInfo struct {
	Index   int            // 字段在表的位置
	Field   string         // 字段名(必须)
	Type    string         // 数据库类型
	Null    string         // 是否为 NULL(建议)
	Key     string         // 索引名(建议)
	Default sql.NullString // 默认值
	Extra   string         // 预留字段
}

// IsPri 是否为主键
func (t *TableColInfo) IsPri() bool {
	return t.Key == PriFlag
}

// NotNull 数据库字段非空约束, NO 不能为 NULL, YES 能为 NULL
func (t *TableColInfo) NotNull() bool {
	return t.Null == NotNullFlag
}

type SortByTableColInfo []*TableColInfo

func (a SortByTableColInfo) Len() int           { return len(a) }
func (a SortByTableColInfo) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByTableColInfo) Less(i, j int) bool { return a[i].Index < a[j].Index }



