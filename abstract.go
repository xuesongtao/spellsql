package spellsql

import "database/sql"

// DBer
type DBer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Logger
type Logger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Warning(v ...interface{})
	Warningf(format string, v ...interface{})
}

// SelectCallBackFn 对每行查询结果进行取出处理
type SelectCallBackFn func(_row interface{}) error

// marshalFn 序列化方法
type marshalFn func(v interface{}) ([]byte, error)

// unmarshalFn 反序列化方法
type unmarshalFn func(data []byte, v interface{}) error
