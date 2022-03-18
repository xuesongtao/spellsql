package spellsql

import (
	"database/sql"
	"errors"

	"reflect"
	"strings"
)

const (
	defaultTableTag = "json"
)

type CommonDB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Session struct {
	db  *sql.DB
	tx  *sql.Tx
	tag string // 解析字段的tag
}

// NewSession
func NewSession(db *sql.DB, tag ...string) *Session {
	defaultTag := defaultTableTag
	if len(tag) > 0 {
		defaultTag = tag[0]
	}

	return &Session{
		db:  db,
		tag: defaultTag,
	}
}

// DB 如果有事务就返回事务, 没有就直接返回
func (s *Session) DB() CommonDB {
	if s.tx != nil {
		return s.tx
	}
	return s.db
}

// parseTableName 解析表名
func (s *Session) parseTableName(objName string) string {
	res := new(strings.Builder)
	for i, v := range objName {
		if v >= 'A' && v <= 'Z' {
			if i > 0 {
				res.WriteRune('_')
			}
			res.WriteRune(v | ' ')
			continue
		}
		res.WriteRune(v)
	}
	return res.String()
}

// parseTable 解析字段
func (s *Session) parseTable(v interface{}) (table string, columns []string, values []interface{}, err error) {
	tv, err := s.reflectValue(v)
	if err != nil {
		return
	}

	ty := tv.Type()
	if table == "" {
		table = s.parseTableName(ty.Name())
	}
	filedNum := ty.NumField()
	columns = make([]string, 0, filedNum)
	values = make([]interface{}, 0, filedNum)
	for i := 0; i < filedNum; i++ {
		structField := ty.Field(i)
		if structField.Anonymous || !isExported(structField.Name) {
			continue
		}
		tag := structField.Tag.Get(s.tag)
		if tag == "" {
			continue
		}
		val := tv.Field(i)
		tmpIndex := IndexForBF(false, tag, ",")
		if tmpIndex > -1 {
			tag = tag[:tmpIndex]
		}
		columns = append(columns, tag)
		values = append(values, val.Interface())
	}

	if len(columns) == 0 || len(values) == 0 {
		err = errors.New("struct is not ok")
		return
	}
	return
}

// reflectValue
func (s *Session) reflectValue(v interface{}) (reflect.Value, error) {
	tv := removeValuePtr(reflect.ValueOf(v))
	if tv.Kind() != reflect.Struct {
		return tv, errors.New("it must is struct")
	}
	return tv, nil
}

// InsertForObj 新增, 通过指定结构体进行新增
// tableName 会通过输入对象按驼峰转为表面
func (s *Session) InsertForObj(insertObj interface{}, tableName ...string) (sql.Result, error) {
	var inputTable string
	if len(tableName) > 0 {
		inputTable = tableName[0]
	}

	table, columns, values, err := s.parseTable(insertObj)
	if err != nil {
		return nil, err
	}

	if inputTable == "" {
		inputTable = table
	}
	insertSql := NewCacheSql("INSERT INTO ?v (?v)", inputTable, strings.Join(columns, ", "))
	insertSql.SetInsertValues(values...)
	return s.Exec(insertSql.GetSqlStr())
}

// InsertBatchForObj 批量提交
// tableName 如果有值会以此为准, 反之会通过输入对象按驼峰转为表面
func (s *Session) InsertBatchForObj(tableName string, insertObjs ...interface{}) error {
	inputTable := tableName
	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		table, columns, values, err := s.parseTable(insertObj)
		if err != nil {
			return err
		}
		if i == 0 {
			if inputTable == "" {
				inputTable = table
			}
			insertSql = NewCacheSql("INSERT INTO ?v (?v)", inputTable, strings.Join(columns, ", "))
		}
		insertSql.SetInsertValues(values...)
	}

	if !insertSql.ValueIsEmpty() {
		_, err := s.Exec(insertSql.GetSqlStr())
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteForObj 根据对象删除
func (s *Session) DeleteForObj(deleteObj interface{}, tableName ...string) (sql.Result, error) {
	var inputTable string
	if len(tableName) > 0 {
		inputTable = tableName[0]
	}

	table, columns, values, err := s.parseTable(deleteObj)
	if err != nil {
		return nil, err
	}

	if inputTable == "" {
		inputTable = table
	}
	l := len(columns)
	delSqlObj := NewCacheSql("DELETE FROM ?v WHERE", inputTable)
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		delSqlObj.SetWhereArgs("?v=?", k, v)
	}
	return s.DB().Exec(delSqlObj.GetSqlStr())
}

func (s *Session) Select() error {
	return nil
}

func (s *Session) Selects() error {
	return nil
}

func (s *Session) Update() error {
	return nil
}

// Exec 通过执行原生sql
func (s *Session) Exec(sqlStr string) (sql.Result, error) {
	return s.DB().Exec(sqlStr)
}

// Begin 开启事务
func (s *Session) Begin() error {
	var err error
	s.tx, err = s.db.Begin()
	if err != nil {
		return err
	}
	return nil
}

// Commit 提交
func (s *Session) Commit() error {
	return s.tx.Commit()
}

// Rollback 回滚
func (s *Session) Rollback() error {
	return s.tx.Rollback()
}
