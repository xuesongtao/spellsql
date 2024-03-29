package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ====================================== spellsql =============================================

const (
	// sql 操作数字
	none uint8 = iota
	INSERT
	DELETE
	SELECT
	UPDATE

	// sql LIKE 语句
	ALK // 全模糊 如: xxx LIKE "%xxx%"
	RLK // 右模糊 如: xxx LIKE "xxx%"
	LLK // 左模糊 如: xxx LIKE "%xxx"

	// sql join 语句
	LJI // 左连接
	RJI // 右连接
)

var (
	sqlSyncPool = sync.Pool{New: func() interface{} { return new(SqlStrObj) }} // 考虑到性能问题, 这里用 pool
)

// ====================================== orm =============================================

const (
	defaultTableTag        = "json"
	defaultBatchSelectSize = 10 // 批量查询默认条数
	NULL                   = "NULL"
)

const (
	_ uint8 = iota
	// 查询时, 用于标记查询的 dest type
	structFlag   // struct
	sliceFlag    // 切片
	mapFlag      // map
	oneFieldFlag // 单字段

	// 标记是否需要对字段进行序列化处理
	sureMarshal
	sureUnmarshal
)

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

	// 标记每次使用完后, 是否释放, 因为几乎都是共用同一个适配器, 减少初始化, 如果要释放的话将这里
	isFreeTmerFlag = false
	getTmerOnce    sync.Once
	getTmerFn      = func() TableMetaer { return Mysql() } // 获取表初始化表元信息, 默认 mysql

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

// FreeTmerFlag 是否每次调用 orm 完后需要释放 tmer
// 如果都是适配相同的数据库, 则可以设置 false, 避免每次都需要初始化适配器
// 反之应该设置为 true
func FreeTmerFlag(is bool) {
	isFreeTmerFlag = is
}

// GlobalTmer 设置全局 tmer, 如果要局部使用, 请使用 Tmer
func GlobalTmer(f func() TableMetaer) {
	getTmerOnce.Do(func() {
		getTmerFn = f
	})
}

// ====================================== other =============================================

// 公共部分
var (
	tmpBuf = sync.Pool{New: func() interface{} { return new(strings.Builder) }}
)

// log 处理
var (
	sLog    Logger
	logOnce sync.Once
)

func init() {
	logOnce.Do(func() {
		sLog = NewLogger()
	})
}

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

// SetLogger 设置 logger
func SetLogger(logger Logger) {
	sLog = logger
}
