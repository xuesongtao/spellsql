package spellsql

import (
	"context"
	"reflect"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
)

const (
	ALK = internal.ALK // 全模糊 如: xxx LIKE "%xxx%"
	RLK = internal.RLK // 右模糊 如: xxx LIKE "xxx%"
	LLK = internal.LLK // 左模糊 如: xxx LIKE "%xxx"

	// sql join 语句
	LJI = internal.LJI // 左连接
	RJI = internal.RJI // 右连接

	TABLE_NAME = "TableName"

	NULL = internal.NULL
)

// DBer
type DBer = dialect.DBer

// Logger
type Logger interface {
	Info(ctx context.Context, v ...any)
	Error(ctx context.Context, v ...any)
	Warning(ctx context.Context, v ...any)
}

type TableNamer interface {
	// TableName 返回表名
	TableName() string
}

var tableNameType = reflect.TypeFor[TableNamer]()

// SelectCallBackFn 对每行查询结果进行取出处理
type SelectCallBackFn func(_row any) error

type MarshalFn func(v any) ([]byte, error)

type UnmarshalFn func(data []byte, v any) error
