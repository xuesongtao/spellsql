package spellsql

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	PriFlag     = "PRI" // 主键标识
	NotNullFlag = "NO"  // 非空标识
)

var (
	_ TableMetaer = &CommonTable{}
	_ TableMetaer = &MysqlTable{}
	_ TableMetaer = &PgTable{}
)

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
func (t *Table) Tmer(obj TableMetaer) *Table {
	if obj != nil { // 出现了修改, 打印下 log
		var old TableMetaer
		if t.tmer != nil {
			old = t.tmer
		}
		if old != nil && old.GetAdapterName() != obj.GetAdapterName() {
			sLog.Warning(t.ctx, fmt.Sprintf("Tmer old %q to new %q", old.GetAdapterName(), obj.GetAdapterName()))
		}
		t.tmer = obj
	}
	return t
}

// 以下为适配多个不同类型的 db

// CommonTable 基类
type CommonTable struct {
}

func (c *CommonTable) GetValueStrSymbol() byte {
	return '"'
}

func (c *CommonTable) GetValueEscapeMap() map[byte][]byte {
	return GetValueEscapeMap()
}

func (c *CommonTable) GetParcelFieldSymbol() byte {
	return '`'
}

func (c *CommonTable) GetAdapterName() string {
	c.noImplement("GetAdapterName")
	return ""
}

func (c *CommonTable) SetTableName(tableName string) {
	c.noImplement("SetTableName")
}

func (c *CommonTable) SetCtx(ctx context.Context) {
	c.noImplement("SetCtx")
}

func (c *CommonTable) GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) {
	c.noImplement("GetField2ColInfoMap")
	return nil, nil
}

func (c *CommonTable) noImplement(name string) {
	sLog.Error(context.Background(), name, "no implement")
}

// *******************************************************************************
// *                             mysql                                           *
// *******************************************************************************

type MysqlTable struct {
	CommonTable
	ctx      context.Context
	initArgs []string
}

// Mysql
func Mysql() *MysqlTable {
	return &MysqlTable{}
}

func (m *MysqlTable) GetAdapterName() string {
	return "mysql"
}

func (m *MysqlTable) SetTableName(name string) {
	m.initArgs = []string{name}
}

func (m *MysqlTable) SetCtx(ctx context.Context) {
	m.ctx = ctx
}

func (m *MysqlTable) GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) {
	if len(m.initArgs) != 1 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, m.GetAdapterName())
	}
	sqlStr := NewCacheSql("SHOW COLUMNS FROM ?v", m.initArgs[0]).SetCtx(m.ctx).SetPrintLog(printLog).GetSqlStr()
	rows, err := db.QueryContext(m.ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
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
	CommonTable
	ctx      context.Context
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

func (p *PgTable) GetAdapterName() string {
	return "pg"
}

func (p *PgTable) SetTableName(name string) {
	p.initArgs[1] = name
}

func (p *PgTable) SetCtx(ctx context.Context) {
	p.ctx = ctx
}

func (p *PgTable) GetParcelFieldSymbol() byte {
	return '"'
}

func (p *PgTable) GetValueEscapeMap() map[byte][]byte {
	escapeMap := GetValueEscapeMap()
	// 将 "'" 进行转义
	escapeMap['\''] = []byte{'\'', '\''}
	return escapeMap
}

func (p *PgTable) GetValueStrSymbol() byte {
	return '\''
}

func (p *PgTable) GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) {
	if len(p.initArgs) != 2 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, p.GetAdapterName())
	}
	sqlStr := NewCacheSql(
		"SELECT c.column_name,c.data_type,c.is_nullable,tc.constraint_type,c.column_default FROM information_schema.columns AS c "+
			"LEFT JOIN information_schema.constraint_column_usage AS ccu USING (column_name,table_name) "+
			"LEFT JOIN information_schema.table_constraints tc ON tc.constraint_name=ccu.constraint_name "+
			"WHERE c.table_schema='?v' AND c.table_name='?v'", p.initArgs[0], p.initArgs[1]).
		SetCtx(p.ctx).SetPrintLog(printLog).GetSqlStr()
	rows, err := db.QueryContext(p.ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
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
			return nil, fmt.Errorf("mysql scan is failed, err: %v", err)
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
