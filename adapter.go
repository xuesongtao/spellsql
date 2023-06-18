package spellsql

import (
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

// Tmer 设置不同数据库表初始化表方式, 调用的时候应该首先调用
// 说明: 此方法为局部方法, 如果要全局设置可以 GlobalTmer
// 如: NewTable(db).Tmer(Pg("man")).xxx
func (t *Table) Tmer(obj TableMetaer) *Table {
	if obj != nil { // 出现了修改, 打印下 log
		old := t.tmer
		if old != nil {
			sLog.Warningf("Tmer old %q to new %q", old.GetAdapterName(), obj.GetAdapterName())
		}
		t.tmer = obj
	}
	return t
}

// 以下为适配多个不同类型的 db

// CommonTable 基类
type CommonTable struct {
}

func (c *CommonTable) EscapeBytes(b []byte) []byte {
	return b
}

func (c *CommonTable) GetStrSymbol() byte {
	return '"'
}

func (c *CommonTable) GetAdapterName() string {
	c.noImplement("GetAdapterName")
	return ""
}

func (c *CommonTable) SetTableName(tableName string) {
	c.noImplement("SetTableName")
}

func (c *CommonTable) GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) {
	c.noImplement("GetField2ColInfoMap")
	return nil, nil
}

func (c *CommonTable) noImplement(name string) {
	sLog.Error(name, "no implement")
}

// *******************************************************************************
// *                             mysql                                           *
// *******************************************************************************

type MysqlTable struct {
	CommonTable
	initArgs       []string
	escapeBytesMap map[byte]bool
}

// Mysql
func Mysql() *MysqlTable {
	return &MysqlTable{escapeBytesMap: map[byte]bool{
		// json 处理
		'n': true,
		'r': true,
		't': true,
	}}
}

func (m *MysqlTable) EscapeBytes(b []byte) []byte {
	vLen := len(b)
	buf := make([]byte, 0, vLen)
	for i := 0; i < vLen; i++ {
		v := b[i]
		buf = append(buf, v)
		switch v {
		case '\\': // json 符号转义
			vv := b[i+1]
			if m.escapeBytesMap[vv] {
				buf = append(buf, '\\', vv)
				i++
			}
		}
	}
	return buf
}

func (m *MysqlTable) GetAdapterName() string {
	return "mysql"
}

func (m *MysqlTable) SetTableName(name string) {
	m.initArgs = []string{name}
}

func (m *MysqlTable) GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) {
	if len(m.initArgs) != 1 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, m.GetAdapterName())
	}
	sqlStr := NewCacheSql("SHOW COLUMNS FROM ?v", m.initArgs[0]).SetPrintLog(printLog).GetSqlStr()
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	for rows.Next() {
		var info TableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return nil, fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		cacheCol2InfoMap[info.Field] = &info
	}
	return cacheCol2InfoMap, nil
}

// *******************************************************************************
// *                             pg                                              *
// *******************************************************************************

type PgTable struct {
	CommonTable
	initArgs []string
}

// Pg
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
	return obj
}

func (p *PgTable) GetAdapterName() string {
	return "pg"
}

func (p *PgTable) SetTableName(name string) {
	p.initArgs[1] = name
}

func (p *PgTable) GetStrSymbol() byte {
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
			"WHERE c.table_schema='?v' AND c.table_name='?v'", p.initArgs[0], p.initArgs[1]).SetPrintLog(printLog).GetSqlStr()
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
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
		cacheCol2InfoMap[info.Field] = &info
	}
	return cacheCol2InfoMap, nil
}
