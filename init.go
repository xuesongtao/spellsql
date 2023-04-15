package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// 公共部分
var (
	tmpBuf = sync.Pool{New: func() interface{} { return new(strings.Builder) }}
)

func getTmpBuf(size ...int) *strings.Builder {
	obj := tmpBuf.Get().(*strings.Builder)
	if len(size) > 0 {
		obj.Grow(size[0])
	}
	return obj
}

func putTmpBuf(obj *strings.Builder) {
	obj.Reset()
	tmpBuf.Put(obj)
}

// spellsql 部分
var (
	sqlSyncPool = sync.Pool{New: func() interface{} { return new(SqlStrObj) }} // 考虑到性能问题, 这里用 pool
)

// orm 部分
var (
	cacheTableName2ColInfoMap      = NewLRU(lruSize) // 缓存表的字段元信息, key: tableName, value: tableColInfo
	cacheStructType2StructFieldMap = NewLRU(lruSize) // 缓存结构体 reflect.Type 对应的 field 信息, key: struct 的 reflect.Type, value: map[colName]structField

	// 常用就缓存下
	cacheTabObj     = sync.Pool{New: func() interface{} { return new(Table) }}
	cacheNullString = sync.Pool{New: func() interface{} { return new(sql.NullString) }}
	cacheNullInt64  = sync.Pool{New: func() interface{} { return new(sql.NullInt64) }}

	// null 类型
	nullInt64Type   = reflect.TypeOf(sql.NullInt64{})
	nullFloat64Type = reflect.TypeOf(sql.NullFloat64{})

	// error
	structTagErr = fmt.Errorf("you should sure struct is ok, eg: %s", "type User struct {\n"+
		"    Name string `json:\"name\"`\n"+
		"}")
	tableNameIsUnknownErr  = errors.New("table name is unknown")
	nullRowErr             = errors.New("row is null")
	findOneDestTypeErr     = errors.New("dest should is struct/oneField/map")
	findAllDestTypeErr     = errors.New("dest should is struct/oneField/map slice")
	getField2ColInfoMapErr = "%q GetField2ColInfoMap initArgs is not ok"
)

// log 处理
var (
	sLog Logger
	once sync.Once
)

func init() {
	sLog = NewCjLogger()
}

// SetLogger 设置 logger
func SetLogger(logger Logger) {
	once.Do(func() {
		sLog = logger
	})
}
