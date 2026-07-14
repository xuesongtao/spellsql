package spellsql

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"sync"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

// ====================================== spellsql =============================================

// ====================================== orm =============================================

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
	cacheTableName2ColInfoMap      = utils.NewLRU() // 缓存表的字段元信息, key: tableName, value: tableColInfo
	cacheStructType2StructFieldMap = utils.NewLRU() // 缓存结构体 reflect.Type 对应的 field 信息, key: struct 的 reflect.Type, value: map[colName]structField

	// 常用就缓存下
	cacheTabObj     = sync.Pool{New: func() interface{} { return new(Table) }}
	cacheNullString = sync.Pool{New: func() interface{} { return new(sql.NullString) }}
	cacheNullInt64  = sync.Pool{New: func() interface{} { return new(sql.NullInt64) }}

	// null 类型
	nullInt64Type   = reflect.TypeOf(sql.NullInt64{})
	nullFloat64Type = reflect.TypeOf(sql.NullFloat64{})

	globalDbTypeOnce = sync.Once{}
)

func newTable(db DBer, args ...string) *Table {
	if v := cacheTabObj.Get(); v != nil {
		t := v.(*Table)
		t.Reset()
		if t.builder != nil {
			panic("builder is no null")
		}
		return t.initDb(db, args...)
	}
	return NewTable(db, args...)
}

func freeTable(t *Table) {
	if t == nil {
		return
	}
	t.Reset()
	cacheTabObj.Put(t)
}

func GlobalDbType(dt dialect.DbType) {
	json.Marshal(dt) // 仅用于触发 dt 的 init, 以便注册 dialect
	globalDbTypeOnce.Do(func() {
		dialect.DefaultDbType = dt
	})
}

// ====================================== other =============================================
// log 处理
var (
	sLog Logger
)

func init() {
	sLog = internal.NewLogger()
}

// SetLogger 设置 logger
func SetLogger(logger Logger) {
	sLog = logger
}
