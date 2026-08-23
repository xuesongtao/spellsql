package spellsql

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/sqldb"
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
type DBer = sqldb.DBer

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

type Hooker interface {
	// BeforeHook 执行前的 hook
	BeforeHook(ctx context.Context, event *HookEvent) (context.Context, error)

	// AfterHook 执行完的 hook
	AfterHook(ctx context.Context, event *HookEvent)
}

var tableNameType = reflect.TypeFor[TableNamer]()

// SelectCallBackFn 对每行查询结果进行取出处理
type SelectCallBackFn func(_row any) error

type MarshalFn func(v any) ([]byte, error)

type UnmarshalFn func(data []byte, v any) error

// HookEvent sql 执行前后 hook 事件
type HookEvent struct {
	ctx          context.Context
	NeedPrintSql bool               // 是否需要打印 sql
	St           time.Time          // 执行开始时间
	Builder      builder.SQLBuilder // 查询 sqlBuilder
	CallInfo     []string           // 调用的位置, 长度为 2, 第一个为文件名, 第二个为行号
	Err          error              // 执行的错误
}

func (a *HookEvent) GetCall() string {
	if len(a.CallInfo) != 2 {
		return ""
	}
	return filepath.Base(a.CallInfo[0]) + ":" + a.CallInfo[1]
}

type DefaultHook struct{}

func (d *DefaultHook) BeforeHook(ctx context.Context, event *HookEvent) (context.Context, error) {
	return ctx, nil
}

func (d *DefaultHook) AfterHook(ctx context.Context, event *HookEvent) {
	prefix := "[" + event.GetCall() + " " + "cost:" + fmt.Sprintf("%.3f", float64(time.Since(event.St).Nanoseconds())/1e6) + "ms]"
	if event.Err != nil {
		sLog.Error(ctx, prefix, "err:", event.Err.Error()+";", "sql:", event.Builder.GetSqlStr())
		return
	}
	if !event.NeedPrintSql {
		return
	}
	sLog.Info(ctx, prefix, event.Builder.GetSqlStr())
}

func getCallInfo(skip int) []string {
	_, file, line, _ := runtime.Caller(skip)
	return []string{file, utils.Int2Str(int64(line))}
}
