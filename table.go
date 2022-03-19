package spellsql

import (
	"database/sql"
	"errors"
	"strings"
)

type Table struct {
	db        DBer
	tmpSqlObj *SqlStrObj // 暂存对象
	tag       string     // 解析字段的tag
	name      string
	// filedMap  map[string]struct{} // 记录该表的所有字段名
}

// NewTable
func NewTable(db DBer, tableName string, tag ...string) *Table {
	defaultTag := defaultTableTag
	if len(tag) > 0 {
		defaultTag = tag[0]
	}

	return &Table{
		db:   db,
		tag:  defaultTag,
		name: tableName,
	}
}

// Insert 提交, 支持批量提交
// tableName 如果有值会以此为准, 反之会通过输入对象按驼峰转为表面
func (t *Table) Insert(insertObjs ...interface{}) (sql.Result, error) {
	if len(insertObjs) == 0 {
		return nil, errors.New("insertObjs is empty")
	}

	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		table, columns, values, err := parseTable(insertObj, t.tag, t.name)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			insertSql = NewCacheSql("INSERT INTO ?v (?v)", table, strings.Join(columns, ", "))
		}
		insertSql.SetInsertValues(values...)
	}
	return t.db.Exec(insertSql.GetSqlStr())
}

// Delete 会以对象中有值得为条件进行删除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		table, columns, values, err := parseTable(deleteObj[0], t.tag, t.name)
		if err != nil {
			Error("parseTable is failed, err:", err)
			return nil
		}

		l := len(columns)
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", table)
		for i := 0; i < l; i++ {
			k := columns[i]
			v := values[i]
			t.tmpSqlObj.SetWhereArgs("?v=?", k, v)
		}
	} else {
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
func (t *Table) Update(updateObj interface{}) *Table {
	table, columns, values, err := parseTable(updateObj, t.tag, t.name)
	if err != nil {
		Error("parseTable is failed, err:", err)
		return nil
	}

	l := len(columns)
	t.tmpSqlObj = NewCacheSql("UPDATE ?v WHERE", table)
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		t.tmpSqlObj.SetUpdateValueArgs("?v=?", k, v)
	}
	return t
}

func (t *Table) GetOne() error {
	return nil
}

func (t *Table) GetAll() error {
	return nil
}

// Where 支持占位符
// 如: Where("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (t *Table) Where(sqlStr string, args ...interface{}) *Table {
	t.tmpSqlObj.SetWhereArgs(sqlStr, args...)
	return t
}

// OrWhere 支持占位符
// 如: OrWhere("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (t *Table) OrWhere(sqlStr string, args ...interface{}) *Table {
	t.tmpSqlObj.SetOrWhereArgs(sqlStr, args...)
	return t
}

// Raw 执行原生sqlStr
func (t *Table) Raw(sqlStr string) *Table {
	t.tmpSqlObj = NewCacheSql(sqlStr)
	return t
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	return t.db.Exec(t.tmpSqlObj.GetSqlStr())
}
