package spellsql

import (
	"context"

	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
)

const (
	ALK = internal.ALK // 全模糊 如: xxx LIKE "%xxx%"
	RLK = internal.RLK // 右模糊 如: xxx LIKE "xxx%"
	LLK = internal.LLK // 左模糊 如: xxx LIKE "%xxx"

	// sql join 语句
	LJI = internal.LJI // 左连接
	RJI = internal.RJI // 右连接
)

// DBer
type DBer = dialect.DBer

// Logger
type Logger interface {
	Info(ctx context.Context, v ...interface{})
	Error(ctx context.Context, v ...interface{})
	Warning(ctx context.Context, v ...interface{})
}

// SelectCallBackFn 对每行查询结果进行取出处理
type SelectCallBackFn func(_row interface{}) error

type MarshalFn func(v interface{}) ([]byte, error)

type UnmarshalFn func(data []byte, v interface{}) error
