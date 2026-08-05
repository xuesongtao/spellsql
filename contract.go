package spellsql

import (
	"context"
	"reflect"
	"runtime"
	"time"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
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

// AfterHook 执行完的 hook
type AfterHook struct {
	St       time.Time          // 执行开始时间
	Builder  builder.SQLBuilder // 查询 sqlBuilder
	CallInfo []string           // 调用的位置, 长度为 2, 第一个为文件名, 第二个为行号
}

func getCallInfo(skip int) []string {
	_, file, line, _ := runtime.Caller(skip + 1)
	return []string{file, utils.Int2Str(int64(line))}
}
