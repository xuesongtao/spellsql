package spellsql

import (
	"database/sql"
	"reflect"
	"sync"
)

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
)
