package spellsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DbType int // db 类型

func (d DbType) Is(dt DbType) bool {
	return d == dt
}

type getTableTerFn func() TableMeter

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
	tableMeterMap = map[DbType]getTableTerFn{
		MySQL:    func() TableMeter { return Mysql() },
		Postgres: func() TableMeter { return Pg() },
	}
)

func getTableMeter(dbType DbType) TableMeter {
	fn, ok := tableMeterMap[dbType]
	if ok {
		return fn()
	}
	return tableMeterMap[defaultDbType]()
}

func getDialect(dbType DbType) Dialect {
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

// Tmer 设置不同数据库表初始化表方式, 调用的时候应该首先调用
// 说明: 此方法为局部方法, 如果要全局设置可以 GlobalTmer
// 如: NewTable(db).Tmer(Pg("man")).xxx
func (t *Table) Tmer(obj TableMeter) *Table {
	if obj != nil { // 出现了修改, 打印下 log
		// var old TableMetaer
		// if t.tmer != nil {
		// 	old = t.tmer
		// }
		// if old != nil && old.GetAdapterName() != obj.GetAdapterName() {
		// 	sLog.Warning(t.ctx, fmt.Sprintf("Tmer old %q to new %q", old.GetAdapterName(), obj.GetAdapterName()))
		// }
		t.tmer = obj
	}
	return t
}

// *******************************************************************************
// *                             mysql                                           *
// *******************************************************************************

type MysqlTable struct {
	initArgs []string
}

// Mysql
func Mysql() *MysqlTable {
	return &MysqlTable{
		initArgs: []string{},
	}
}

func (m *MysqlTable) GetWarpFieldSymbol() string {
	return "`"
}

func (m *MysqlTable) GetWarpValueStrSymbol() string {
	return "\""
}

func (m *MysqlTable) GetValueEscapeMap() map[byte][]byte {
	return GetValueEscapeMap()
}

// GetLimitSql implements [Dialect].
func (m *MysqlTable) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + Int2Str(int64(limit)) + " OFFSET " + Int2Str(int64(offset))
}

func (m *MysqlTable) GetAdapterName() string {
	return "mysql"
}

func (m *MysqlTable) SetTableName(name string) {
	m.initArgs = []string{name}
}

func (m *MysqlTable) GetField2ColInfoMap(ctx context.Context, db DBer, printLog bool) (map[string]*TableColInfo, error) {
	if len(m.initArgs) != 1 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, m.GetAdapterName())
	}
	st := time.Now()
	sqlStr := NewCacheSql("SHOW COLUMNS FROM ?v", m.initArgs[0]).SetCtx(ctx).SetPrintLog(false).GetSqlStr("")
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer printCostTimeLog(ctx, st, sqlStr, printLog)
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	var index int
	for rows.Next() {
		var info TableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return nil, fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		info.Index = index
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}

// *******************************************************************************
// *                             pg                                              *
// *******************************************************************************

type PgTable struct {
	initArgs []string
}

// Pg, 默认模式: public
// initArgs 允许自定义两个参数
// initArgs[0] 为 schema
// initArgs[1] 为 table name (此参数可以忽略, 因为 orm 内部会处理该值)
func Pg(initArgs ...string) *PgTable {
	obj := &PgTable{initArgs: make([]string, 2)}
	l := len(initArgs)
	switch l {
	case 1:
		obj.initArgs[0] = initArgs[0]
	case 2:
		obj.initArgs[0] = initArgs[0]
		obj.initArgs[1] = initArgs[1]
	}
	if l == 0 {
		obj.initArgs[0] = "public"
	}
	return obj
}

// GetWarpFieldSymbol implements [Dialect].
func (p *PgTable) GetWarpFieldSymbol() string {
	return `"`
}

// GetWarpValueStrSymbol implements [Dialect].
func (p *PgTable) GetWarpValueStrSymbol() string {
	return `'`
}

func (p *PgTable) GetAdapterName() string {
	return "pg"
}

// GetLimitSql implements [Dialect].
func (p *PgTable) GetLimitSql(limit int, offset int) string {
	return "LIMIT " + Int2Str(int64(limit)) + " OFFSET " + Int2Str(int64(offset))
}

func (p *PgTable) SetTableName(name string) {
	p.initArgs[1] = name
}

func (p *PgTable) GetValueEscapeMap() map[byte][]byte {
	escapeMap := GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

func (p *PgTable) GetField2ColInfoMap(ctx context.Context, db DBer, printLog bool) (map[string]*TableColInfo, error) {
	if len(p.initArgs) != 2 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, p.GetAdapterName())
	}
	st := time.Now()
	sqlStr := NewCacheSql(
		"SELECT c.column_name,c.data_type,c.is_nullable,tc.constraint_type,c.column_default FROM information_schema.columns AS c "+
			"LEFT JOIN information_schema.constraint_column_usage AS ccu USING (column_name,table_name) "+
			"LEFT JOIN information_schema.table_constraints tc ON tc.constraint_name=ccu.constraint_name "+
			"WHERE c.table_schema='?v' AND c.table_name='?v'", p.initArgs[0], p.initArgs[1]).
		SetCtx(ctx).SetPrintLog(false).GetSqlStr("")
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("pg query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer printCostTimeLog(ctx, st, sqlStr, printLog)
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	var index int
	for rows.Next() {
		var (
			info TableColInfo
			key  sql.NullString
		)
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &key, &info.Default)
		if err != nil {
			return nil, fmt.Errorf("pg scan is failed, err: %v", err)
		}
		if key.String == "PRIMARY KEY" {
			info.Key = PriFlag
		}
		info.Index = index
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}
