package spellsql

import (
	"context"
	"database/sql"
)

// DBer
type DBer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Logger
type Logger interface {
	Info(ctx context.Context, v ...interface{})
	Error(ctx context.Context, v ...interface{})
	Warning(ctx context.Context, v ...interface{})
}

// TableMetaer 表元信息, 为了适配不同数据库
type TableMetaer interface {
	GetValueStrSymbol() byte                                                      // 获取值字符串符号
	GetValueEscapeMap() map[byte][]byte                                           // 获取值转义规则
	GetParcelFieldSymbol() byte                                                   // 获取字段包裹符号
	GetAdapterName() string                                                       // 获取 db name
	SetTableName(tableName string)                                                // 方便框架调用设置 tableName 参数
	SetCtx(ctx context.Context)                                                   // 设置 context
	GetField2ColInfoMap(db DBer, printLog bool) (map[string]*TableColInfo, error) // key: field
}

// SelectCallBackFn 对每行查询结果进行取出处理
type SelectCallBackFn func(_row interface{}) error

// marshalFn 序列化方法
type marshalFn func(v interface{}) ([]byte, error)

// unmarshalFn 反序列化方法
type unmarshalFn func(data []byte, v interface{}) error
