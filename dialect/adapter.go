package dialect

import (
	"database/sql"
)

type DbType int // db 类型

func (d DbType) Is(dt DbType) bool {
	return d == dt
}

const (
	PriFlag     = "PRI" // 主键标识
	NotNullFlag = "NO"  // 非空标识
)

const (
	MySQL DbType = iota
	Postgres
)

const defaultDbType = MySQL

var (
	_ Dialect = &MysqlTable{}
	_ Dialect = &PgTable{}

	_ TableMeter = &MysqlTable{}
	_ TableMeter = &PgTable{}
)

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

func GetTableMeter(dbType DbType) TableMeter {
	fn, ok := tableMeterMap[dbType]
	if ok {
		return fn()
	}
	return tableMeterMap[defaultDbType]()
}

func GetDialect(dbType DbType) Dialect {
	dialect, ok := dialectMap[dbType]
	if ok {
		return dialect
	}
	return dialectMap[defaultDbType]
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
