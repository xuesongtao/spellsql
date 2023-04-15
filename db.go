package spellsql

import (
	"database/sql"
	"fmt"
)

const (
	PriFlag     = "PRI" // 主键标识
	NotNullFlag = "NO"  // 非空标识
)

// TableColInfo 表列详情
type TableColInfo struct {
	Field   string         // 字段名
	Type    string         // 数据库类型
	Null    string         // 是否为 NULL
	Key     string         // 索引名
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

// Tmer 设置不同数据库表初始化方式, 调用的时候应该首先调用
// 如: NewTable(db).Tmer(Pg("man")).xxx
func (t *Table) Tmer(obj TableMetaer) *Table {
	t.tmer = obj
	return t
}

// *******************************************************************************
// *                             mysql                                           *
// *******************************************************************************

type MysqlTable struct {
	initArgs []string
}

// Mysql
// initArgs 只允许一个参数, table name
func Mysql(initArgs ...string) *MysqlTable {
	return &MysqlTable{initArgs: initArgs}
}

func (m *MysqlTable) GetStrSymbol() byte {
	return '"'
}

func (m *MysqlTable) GetField2ColInfoMap(db DBer) (map[string]*TableColInfo, error) {
	if len(m.initArgs) != 1 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, "mysql")
	}
	sqlStr := NewCacheSql("SHOW COLUMNS FROM ?v", m.initArgs[0]).GetSqlStr()
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
	initArgs []string
}

// Pg
// initArgs 允许自定义两个参数, 如果只有一个的时候, 会将其处理为 table name
func Pg(initArgs ...string) *PgTable {
	args := initArgs
	if len(initArgs) == 1 {
		args = []string{"public", initArgs[0]} // 默认 public
	}
	return &PgTable{initArgs: args}
}

func (p *PgTable) GetStrSymbol() byte {
	return '\''
}

func (p *PgTable) GetField2ColInfoMap(db DBer) (map[string]*TableColInfo, error) {
	if len(p.initArgs) != 2 {
		return nil, fmt.Errorf(getField2ColInfoMapErr, "pg")
	}
	sqlStr := NewCacheSql(
		"SELECT c.column_name,c.data_type,c.is_nullable,tc.constraint_type,c.column_default FROM information_schema.columns AS c "+
			"LEFT JOIN information_schema.constraint_column_usage AS ccu USING (column_name,table_name) "+
			"LEFT JOIN information_schema.table_constraints tc ON tc.constraint_name=ccu.constraint_name "+
			"WHERE c.table_schema='?v' AND c.table_name='?v'", p.initArgs[0], p.initArgs[1]).GetSqlStr()
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
