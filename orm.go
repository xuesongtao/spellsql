package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	// "github.com/gogf/gf/os/glog"
)

const (
	defaultTableTag        = "json"
	defaultBatchSelectSize = 10 // 批量查询默认条数

	// 查询时, 用于标记查询的 dest type
	structNo   = 0
	sliceNo    = 1
	mapNo      = 2
	oneFieldNo = 3

	// 标记是否需要对字段进行序列化处理
	sureMarshal   uint8 = 1
	sureUnmarshal uint8 = 2
)

var (
	structTagErr = fmt.Errorf("you should sure struct is ok, eg: %s", "type User struct {\n"+
		"    Name string `json:\"name\"`\n"+
		"}")
	sqlObjErr             = errors.New("tmpSqlObj is nil")
	tableNameIsUnknownErr = errors.New("table name is unknown")
	nullRowErr            = errors.New("row is null")
)

var (
	cacheTableName2ColInfoMap    = sync.Map{} // 缓存表的字段元信息
	cacheStructTag2FieldIndexMap = sync.Map{} // 缓存结构体 tag 对应的 field index

	// 常用就缓存下
	cacheTabObj     = sync.Pool{New: func() interface{} { return new(Table) }}
	cacheNullString = sync.Pool{New: func() interface{} { return new(sql.NullString) }}
	cacheNullInt64  = sync.Pool{New: func() interface{} { return new(sql.NullInt64) }}
)

type SelectCallBackFn func(_row interface{}) error // 对每行查询结果进行取出处理
type Marshal func(v interface{}) ([]byte, error)
type Unmarshal func(data []byte, v interface{}) error

// tableColInfo 表列详情
type tableColInfo struct {
	Field   string // 字段名
	Type    string // 数据库类型
	Null    string // 是否为 NULL
	Key     string
	Default sql.NullString
	Extra   string
}

type handleColFn struct {
	marshal   Marshal
	unmarshal Unmarshal
}

// Table
type Table struct {
	db               DBer
	printSqlCallSkip uint8                    // 标记打印 sql 时, 需要跳过的 skip, 该参数为 runtime.Caller(skip)
	isPrintSql       bool                     // 标记是否打印 sql
	haveFree         bool                     // 标记 table 释放已释放
	needSetSize      bool                     // 标记批量查询的时候是否需要设置默认返回条数
	tmpSqlObj        *SqlStrObj               // 暂存 SqlStrObj 对象
	tag              string                   // 记录解析 struct 中字段名的 tag
	name             string                   // 表名
	destTypeBitmap   [4]bool                  // 查询时, 用于标记 dest 类型的位图
	cacheCol2InfoMap map[string]*tableColInfo // 记录该表的所有字段名
	handleColFnMap   map[string]*handleColFn  // 处理 col 的方法
}

// NewTable 初始化, 通过 sync.Pool 缓存对象来提高性能
// 注: 使用 INSERT/UPDATE/DELETE/SELECT(SELECT 排除使用 Count)操作后该对象就会被释放, 如果继续使用会出现 panic
// args 支持两个参数
// args[0]: 会解析为 tableName, 这里如果有值, 在进行操作表的时候就会以此表为准,
// 如果为空时, 在通过对象进行操作时按驼峰规则进行解析表名, 解析规则如: UserInfo => user_info
// args[1]: 会解析为待解析的 tag, 默认 defaultTableTag
func NewTable(db DBer, args ...string) *Table {
	if db == nil {
		return nil
	}

	t := cacheTabObj.Get().(*Table)
	t.db = db
	t.printSqlCallSkip = 2
	t.isPrintSql = true
	t.haveFree = false
	t.needSetSize = false
	t.tag = defaultTableTag
	t.name = ""

	switch len(args) {
	case 1:
		t.name = args[0]
	case 2:
		t.name = args[0]
		t.tag = args[1]
	}

	if t.name != "" {
		if err := t.initCacheCol2InfoMap(); err != nil {
			cjLog.Error("initCacheCol2InfoMap is failed, err:", err)
			// glog.Error("initCacheCol2InfoMap is failed, err:", err)
			return nil
		}
	}
	return t
}

func (t *Table) free() {
	if t.haveFree {
		cjLog.Panic("table have free, you can't again use")
		// glog.Panic("table have free, you can't again use")
		return
	}
	t.haveFree = true
	t.db = nil
	t.tmpSqlObj = nil
	t.name = ""
	t.cacheCol2InfoMap = nil
	t.handleColFnMap = nil

	// 存放缓存
	cacheTabObj.Put(t)
}

// Name 设置表名
func (t *Table) Name(tableName string) *Table {
	t.name = tableName
	return t
}

// IsPrintSql 是否打印 sql
func (t *Table) IsPrintSql(is bool) *Table {
	t.isPrintSql = is
	return t
}

// NeedSetSize 查询的时候, 是否需要设置默认 size
func (t *Table) NeedSetSize(need bool) *Table {
	t.needSetSize = need
	return t
}

// PrintSqlCallSkip 用于 sql 打印时候显示调用处的信息
func (t *Table) PrintSqlCallSkip(skip uint8) *Table {
	t.printSqlCallSkip = skip
	return t
}

// initCacheCol2InfoMap 初始化表字段 map, 由于json tag 应用比较多, 为了在后续执行 INSERT/UPDATE 等通过对象取值会存在取值错误现象, 所以需要预处理下
func (t *Table) initCacheCol2InfoMap() error {
	// 已经初始化过了
	if t.cacheCol2InfoMap != nil {
		return nil
	}

	if t.name == "" {
		return tableNameIsUnknownErr
	}

	// 先判断下缓存中有没有
	if info, ok := cacheTableName2ColInfoMap.Load(t.name); ok {
		t.cacheCol2InfoMap, ok = info.(map[string]*tableColInfo)
		if ok {
			return nil
		}
	}

	sqlStr := NewCacheSql("SHOW COLUMNS FROM ?v", t.name).SetPrintLog(t.isPrintSql).GetSqlStr()
	rows, err := t.db.Query(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	t.cacheCol2InfoMap = make(map[string]*tableColInfo, len(columns))
	for rows.Next() {
		var info tableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		t.cacheCol2InfoMap[info.Field] = &info
	}

	cacheTableName2ColInfoMap.Store(t.name, t.cacheCol2InfoMap)
	return nil
}

// SetMarshalFn 设置列名的序列化方法
// 注: 调用必须优先 Insert/Update 操作的方法, 防止通过对象解析字段时被排除
func (t *Table) SetMarshalFn(fn Marshal, cols ...string) *Table {
	if t.handleColFnMap == nil {
		t.handleColFnMap = make(map[string]*handleColFn, len(cols))
	}
	for _, col := range cols {
		if _, ok := t.handleColFnMap[col]; ok {
			t.handleColFnMap[col].marshal = fn
		} else {
			t.handleColFnMap[col] = &handleColFn{marshal: fn}
		}
	}
	return t
}

// SetMarshalFn 设置列名的反序列化方法
// 注: 调用必须优先于 SelectAuto, 防止 SelectAuto 解析时查询字段被排除
func (t *Table) SetUnmarshalFn(fn Unmarshal, cols ...string) *Table {
	if t.handleColFnMap == nil {
		t.handleColFnMap = make(map[string]*handleColFn, len(cols))
	}
	for _, col := range cols {
		if _, ok := t.handleColFnMap[col]; ok {
			t.handleColFnMap[col].unmarshal = fn
		} else {
			t.handleColFnMap[col] = &handleColFn{unmarshal: fn}
		}
	}
	return t
}

// parseStructField2Col 从结构体的 tag 中解析出列名, 同时跳过嵌套, 包含: 对象, 指针对象, 切片, 不可导出字段
func (t *Table) parseStructField2Col(fieldInfo reflect.StructField, args ...uint8) (col string, need bool) {
	if !isExported(fieldInfo.Name) {
		return
	}

	col = fieldInfo.Tag.Get(t.tag)
	if col == "" {
		return
	}
	// 去除 tag 中的干扰, 如: json:"xxx,omitempty"
	col = t.parseTag2Col(col)

	switch fieldInfo.Type.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Struct: // 默认不处理嵌套, 如果有序列化操作就需要处理
		if len(args) == 0 {
			col = "" // 没有的话就直接跳过
			return
		}

		v, ok := t.handleColFnMap[col]
		if !ok {
			col = "" // 没有的话就直接跳过
			return
		}
		switch args[0] {
		case sureMarshal:
			need = v.marshal != nil
		case sureUnmarshal:
			need = v.unmarshal != nil
		default:
			col = "" // 没有的话就直接跳过
		}
		return
	}
	return
}

// getHandleTableCol2Val 用于新增/删除/修改时, 解析结构体中对应列名和值
func (t *Table) getHandleTableCol2Val(v interface{}, isExcludePri bool, tableName ...string) (columns []string, values []interface{}, err error) {
	tv := removeValuePtr(reflect.ValueOf(v))
	if tv.Kind() != reflect.Struct {
		err = errors.New("it must is struct")
		return
	}

	ty := tv.Type()
	if t.name == "" {
		t.name = parseTableName(ty.Name())
	}

	if err := t.initCacheCol2InfoMap(); err != nil {
		return nil, nil, err
	}

	fieldNum := ty.NumField()
	columns = make([]string, 0, fieldNum)
	values = make([]interface{}, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		col, needMarshal := t.parseStructField2Col(ty.Field(i), sureMarshal)
		if col == "" {
			continue
		}

		// 判断下数据库字段是否存在
		tableField, ok := t.cacheCol2InfoMap[col]
		if !ok {
			continue
		}

		if isExcludePri && tableField.Key == "PRI" { // 主键, 防止更新
			continue
		}

		// 值为空也跳过
		val := tv.Field(i)
		if val.IsZero() {
			continue
		}

		columns = append(columns, col)
		if needMarshal {
			dataBytes, err := t.handleColFnMap[col].marshal(val.Interface())
			if err != nil {
				return nil, nil, err
			}
			values = append(values, string(dataBytes))
		} else {
			values = append(values, val.Interface())
		}
	}

	if len(columns) == 0 || len(values) == 0 {
		err = structTagErr
		return
	}
	return
}

// parseTag2Col 解析 tag 中表的列名
func (t *Table) parseTag2Col(tag string) (column string) {
	// 因为 tag 中有可能出现多个值, 需要处理下
	tmpIndex := IndexForBF(true, tag, ",")
	if tmpIndex > -1 {
		column = tag[:tmpIndex]
	} else {
		column = tag
	}
	return
}

// Insert 提交, 支持批量提交
func (t *Table) Insert(insertObjs ...interface{}) *Table {
	if len(insertObjs) == 0 {
		cjLog.Error("insertObjs is empty")
		// glog.Error("insertObjs is empty")
		return nil
	}

	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		columns, values, err := t.getHandleTableCol2Val(insertObj, true, t.name)
		if err != nil {
			cjLog.Error("getHandleTableCol2Val is failed, err:", err)
			// glog.Error("getHandleTableCol2Val is failed, err:", err)
			return nil
		}
		if i == 0 {
			insertSql = NewCacheSql("INSERT INTO ?v (?v)", t.name, strings.Join(columns, ", "))
		}
		insertSql.SetInsertValues(values...)
	}
	t.tmpSqlObj = insertSql
	return t
}

// Delete 会以对象中有值得为条件进行删除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		columns, values, err := t.getHandleTableCol2Val(deleteObj[0], false, t.name)
		if err != nil {
			cjLog.Error("getHandleTableCol2Val is failed, err:", err)
			// glog.Error("getHandleTableCol2Val is failed, err:", err)
			return nil
		}

		l := len(columns)
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
		for i := 0; i < l; i++ {
			k := columns[i]
			v := values[i]
			t.tmpSqlObj.SetWhereArgs("?v = ?", k, v)
		}
	} else {
		if t.name == "" {
			cjLog.Error(tableNameIsUnknownErr)
			// glog.Error(tableNameIsUnknownErr)
			return nil
		}
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
func (t *Table) Update(updateObj interface{}, where string, args ...interface{}) *Table {
	columns, values, err := t.getHandleTableCol2Val(updateObj, true, t.name)
	if err != nil {
		cjLog.Error("getHandleTableCol2Val is failed, err:", err)
		// glog.Error("getHandleTableCol2Val is failed, err:", err)
		return nil
	}

	l := len(columns)
	t.tmpSqlObj = NewCacheSql("UPDATE ?v SET", t.name)
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		t.tmpSqlObj.SetUpdateValueArgs("?v = ?", k, v)
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)
	return t
}

// Select 查询内容
// fields 多个通过逗号隔开
func (t *Table) Select(fields string) *Table {
	if fields == "" {
		cjLog.Error("fields is null")
		// glog.Error("fields is null")
		return nil
	}

	if t.name == "" {
		cjLog.Error(tableNameIsUnknownErr)
		// glog.Error(tableNameIsUnknownErr)
		return nil
	}

	t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", fields, t.name)
	return t
}

// SelectAuto 根据输入类型进行自动推断要查询的字段值
func (t *Table) SelectAuto(src interface{}, tableName ...string) *Table {
	if len(tableName) > 0 {
		t.name = tableName[0]
	}

	if val, ok := src.(string); ok {
		t.Select(val)
		return t
	}

	ty := removeTypePtr(reflect.TypeOf(src))
	selectFields := make([]string, 0, 5)
	switch ty.Kind() {
	case reflect.Struct, reflect.Slice:
		if ty.Kind() == reflect.Slice {
			ty = ty.Elem()
			if ty.Kind() == reflect.Ptr {
				ty = removeTypePtr(ty)
			}
		}
		if t.name == "" {
			t.name = parseTableName(ty.Name())
		}
		if err := t.initCacheCol2InfoMap(); err != nil {
			cjLog.Error("initCacheCol2InfoMap is failed, err:", err)
			// glog.Error("initCacheCol2InfoMap is failed, err:", err)
			return nil
		}

		_, sortCol := t.parseCol2FieldIndex(ty, true)
		for _, col := range sortCol {
			// 排除结构体中的字段, 数据库没有
			if _, ok := t.cacheCol2InfoMap[col]; !ok {
				continue
			}
			selectFields = append(selectFields, col)
		}
		t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", strings.Join(selectFields, ", "), t.name)
	default:
		cjLog.Warning("src kind is not struct or slice struct")
		// glog.Warning("src kind is not struct or slice struct")
		t.SelectAll()
	}
	return t
}

//  SelectAll() 查询所有字段
func (t *Table) SelectAll() *Table {
	return t.Select("*")
}

// SelectCount 查询总数
func (t *Table) SelectCount() *Table {
	return t.Select("COUNT(*)")
}

// Count 获取总数
func (t *Table) Count(total interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectCount()
	}

	// 这里不要释放, 如果是列表查询的话, 还会再进行查询内容操作
	// defer t.free()
	return t.db.QueryRow(t.tmpSqlObj.SetPrintLog(t.isPrintSql).SetCallerSkip(t.printSqlCallSkip).GetTotalSqlStr()).Scan(total)
}

// FindOne 单行查询
// dest 长度 > 1 时, 支持多个字段查询
// dest 长度 == 1 时, 支持 struct/单字段/map
func (t *Table) FindOne(dest ...interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	if t.needSetSize {
		t.tmpSqlObj.SetLimit(0, 1)
	}
	if len(dest) == 1 {
		return t.find(dest[0])
	}
	t.printSqlCallSkip += 1
	return t.QueryRowScan(dest...)
}

// FindOneFn 单行查询
// dest 支持 struct/单字段/map
// fn 支持将查询结果行进行修改
func (t *Table) FindOneFn(dest interface{}, fn ...SelectCallBackFn) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	if t.needSetSize {
		t.tmpSqlObj.SetLimit(0, 1)
	}
	return t.find(dest, fn...)
}

// FindAll 多行查询
// 如果没有指定查询条数, 默认 defaultBatchSelectSize
// dest 支持(struct/单字段/map)切片
// fn 支持将查询结果行进行处理
func (t *Table) FindAll(dest interface{}, fn ...SelectCallBackFn) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	if t.tmpSqlObj.LimitIsEmpty() && t.needSetSize {
		t.tmpSqlObj.SetLimit(0, defaultBatchSelectSize)
	}
	return t.find(dest, fn...)
}

// FindWhere 如果没有添加查询字段内容, 会根据输入对象进行解析查询
// 如果没有指定查询条数, 默认 defaultBatchSelectSize
// dest 支持 struct/slice/单字段/map
func (t *Table) FindWhere(dest interface{}, where string, args ...interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectAuto(dest)
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)

	if t.tmpSqlObj.LimitIsEmpty() && t.needSetSize {
		t.tmpSqlObj.SetLimit(0, defaultBatchSelectSize)
	}
	return t.find(dest)
}

// parseCol2FieldIndex 通过解析输入结构体, 返回 map[tag 名]字段偏移量, 同时缓存起来
func (t *Table) parseCol2FieldIndex(ty reflect.Type, isNeedSort bool) (col2FieldIndexMap map[string]int, sortCol []string) {
	// 非结构体就返回空
	if ty.Kind() != reflect.Struct {
		return nil, nil
	}

	// 通过地址来取, 防止出现重复
	// 当 t.handleColFnMap != nil 不等于空时, 为了防止解析 selectFields 缺少, 不能走缓存中取
	if t.handleColFnMap == nil {
		if cacheVal, ok := cacheStructTag2FieldIndexMap.Load(ty); ok { // 需要排除再包含 t.handleColFnMap 不为空的
			col2FieldIndexMap = cacheVal.(map[string]int)
			if isNeedSort { // 按照col2FieldIndexMap的value进行排序
				l := len(col2FieldIndexMap)
				tmpMap := make(map[int]string, l)
				tmpSortVal := make([]int, 0, l)
				for col, fieldIndex := range col2FieldIndexMap {
					tmpMap[fieldIndex] = col
					tmpSortVal = append(tmpSortVal, fieldIndex)
				}
				sort.Ints(tmpSortVal)
				sortCol = make([]string, 0, l)
				for _, fieldIndex := range tmpSortVal {
					sortCol = append(sortCol, tmpMap[fieldIndex])
				}
			}
			return
		}
	}

	fieldNum := ty.NumField()
	col2FieldIndexMap = make(map[string]int, fieldNum)
	sortCol = make([]string, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		col, _ := t.parseStructField2Col(ty.Field(i), sureUnmarshal)
		if col == "" {
			continue
		}
		col2FieldIndexMap[col] = i
		sortCol = append(sortCol, col)
	}

	if t.handleColFnMap == nil {
		cacheStructTag2FieldIndexMap.Store(ty, col2FieldIndexMap)
	}
	return
}

// loadDestTypeBitmap 记录 dest 的类型, 因为对应的操作不会同时调用所有不存在数据竞争
func (t *Table) loadDestTypeBitmap(dest reflect.Type) {
	for i := 0; i < len(t.destTypeBitmap); i++ {
		t.destTypeBitmap[i] = false
	}

	switch dest.Kind() {
	case reflect.Struct:
		t.destTypeBitmap[structNo] = true
	case reflect.Slice:
		t.destTypeBitmap[sliceNo] = true
	case reflect.Map:
		t.destTypeBitmap[mapNo] = true
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		t.destTypeBitmap[oneFieldNo] = true
	}
}

// find 查询处理入口, 根据 dest 类型进行分配处理
func (t *Table) find(dest interface{}, fn ...SelectCallBackFn) error {
	defer t.free()
	ty := reflect.TypeOf(dest)
	if ty.Kind() != reflect.Ptr {
		return errors.New("dest should is ptr")
	}
	ty = removeTypePtr(ty)
	t.loadDestTypeBitmap(ty)
	t.printSqlCallSkip += 2

	rows, err := t.Query(false)
	if err != nil {
		return err
	}
	defer rows.Close()

	if t.destTypeBitmap[structNo] || t.destTypeBitmap[mapNo] || t.destTypeBitmap[oneFieldNo] {
		return t.scanOne(rows, ty, dest, fn...)
	} else if t.destTypeBitmap[sliceNo] {
		return t.scanAll(rows, ty.Elem(), dest, fn...)
	} else {
		return errors.New("dest kind nonsupport")
	}
}

// scanAll 处理多个结果集
func (t *Table) scanAll(rows *sql.Rows, ty reflect.Type, dest interface{}, fn ...SelectCallBackFn) error {
	isPtr := ty.Kind() == reflect.Ptr
	if isPtr {
		ty = removeTypePtr(ty) // 去指针
	}

	t.loadDestTypeBitmap(ty)
	colTypes, _ := rows.ColumnTypes()
	colLen := len(colTypes)
	col2FieldIndexMap, _ := t.parseCol2FieldIndex(ty, false)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	values := make([]interface{}, colLen)
	destReflectValue := reflect.Indirect(reflect.ValueOf(dest))
	if destReflectValue.IsNil() {
		destReflectValue.Set(reflect.MakeSlice(destReflectValue.Type(), 0, colLen))
	}
	for rows.Next() {
		base := reflect.New(ty).Elem()
		if err := t.getScanValues(base, col2FieldIndexMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
			return err
		}

		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("rows scan is failed, err: %v", err)
		}

		if err := t.setDest(base, colTypes, fieldIndex2NullIndexMap, values); err != nil {
			return err
		}

		if len(fn) == 1 { // 回调方法
			if isPtr && !t.destTypeBitmap[mapNo] { // 指针类型
				if err := fn[0](base.Addr().Interface()); err != nil {
					return err
				}
			} else { // 值类型
				if err := fn[0](base.Interface()); err != nil {
					return err
				}
			}
		}

		if isPtr { // 判断下切片中是指针类型还是值类型
			destReflectValue.Set(reflect.Append(destReflectValue, base.Addr()))
		} else {
			destReflectValue.Set(reflect.Append(destReflectValue, base))
		}
	}
	return nil
}

// scanOne 处理单个结果集
func (t *Table) scanOne(rows *sql.Rows, ty reflect.Type, dest interface{}, fn ...SelectCallBackFn) error {
	// t.loadDestTypeBitmap(ty) // 这里可以不用再处理
	colTypes, _ := rows.ColumnTypes()
	colLen := len(colTypes)
	col2FieldIndexMap, _ := t.parseCol2FieldIndex(ty, false)
	values := make([]interface{}, colLen)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	destReflectValue := reflect.Indirect(reflect.ValueOf(dest))
	base := reflect.New(ty).Elem()
	if err := t.getScanValues(base, col2FieldIndexMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
		return err
	}

	if !rows.Next() { // 没有数据
		return nullRowErr
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	if err := t.setDest(base, colTypes, fieldIndex2NullIndexMap, values); err != nil {
		return err
	}

	if len(fn) == 1 { // 回调方法, 方便修改
		if t.destTypeBitmap[mapNo] {
			if err := fn[0](base.Interface()); err != nil {
				return err
			}
		} else {
			if err := fn[0](base.Addr().Interface()); err != nil {
				return err
			}
		}
	}
	destReflectValue.Set(base)
	return nil
}

// getScanValues 获取待 Scan 的内容
func (t *Table) getScanValues(dest reflect.Value, col2FieldIndexMap map[string]int, fieldIndex2NullIndexMap map[int]int, colTypes []*sql.ColumnType, values []interface{}) error {
	var (
		isStruct         = t.destTypeBitmap[structNo]   // struct
		isMap            = t.destTypeBitmap[mapNo]      // map
		isOneField       = t.destTypeBitmap[oneFieldNo] // 单字段
		isSliceField     = t.destTypeBitmap[sliceNo]    // 用于 QueryRowScan, 单行多字段查询
		structMissFields []string
	)
	for i, colType := range colTypes {
		var (
			fieldIndex       int
			structFieldExist bool = true
			colName               = colType.Name()
		)
		if isStruct && col2FieldIndexMap != nil {
			fieldIndex, structFieldExist = col2FieldIndexMap[colName]
		}

		// 说明结构里查询的值不存在
		if !structFieldExist {
			if structMissFields == nil {
				structMissFields = make([]string, 0, len(colTypes)/2)
			}
			structMissFields = append(structMissFields, colName)
			continue
		}

		// NULL 值处理, 防止 sql 报错, 否则就直接 scan 到 struct 字段值
		// canNull, _ := colType.Nullable() // 根据此获取的 NULL 值不准确, 但如果有值的话会是准确的
		colInfo := t.cacheCol2InfoMap[colName]
		// fmt.Printf("canNull: %v colInfo: %+v\n", canNull, colInfo) canNull || colInfo == nil ||
		if colInfo == nil || (colInfo != nil && colInfo.Null == "YES") { // colInfo == nil 说明初始化表失败, 就直接通过空处理, 查询的时候只会在 Query 里初始化
			// fmt.Println(colName)
			switch colType.ScanType().Name() {
			case "NullInt64":
				values[i] = cacheNullInt64.Get().(*sql.NullInt64)
			case "NullFloat64":
				values[i] = new(sql.NullFloat64)
			default:
				values[i] = cacheNullString.Get().(*sql.NullString)
			}

			// struct, 这里记录 struct 那个字段需要映射 NULL 值
			// map/单字段, 为了减少创建标记, 借助 fieldIndex2NullIndexMap 用于标识单字段是否包含空值,  在 setDest 使用
			if isStruct && structFieldExist {
				fieldIndex2NullIndexMap[fieldIndex] = i
			} else if isMap || isSliceField || isOneField {
				fieldIndex2NullIndexMap[i] = i
			}
			continue
		}

		// 处理数据库字段非 NULL 部分
		if isStruct { // 结构体
			// 在非 NULL 的时候, 也判断下是否需要反序列化
			if handleCol, ok := t.handleColFnMap[colName]; ok && handleCol.unmarshal != nil {
				values[i] = cacheNullString.Get().(*sql.NullString)
				fieldIndex2NullIndexMap[fieldIndex] = i
				continue
			}
			values[i] = dest.Field(fieldIndex).Addr().Interface()
		} else if isMap {
			destValType := dest.Type().Elem()
			if destValType.Kind() == reflect.Interface {
				// 如果 map 的 value 为 interface{} 时, 数据库类型为字符串时 driver.Value 的类型为 RawBytes, 再经过 Scan 后, 会被处理为 []byte
				// 为了避免这种就直接处理为字符串
				values[i] = cacheNullString.Get().(*sql.NullString)
				fieldIndex2NullIndexMap[i] = i
			} else {
				values[i] = reflect.New(destValType).Interface()
			}
		} else if isOneField { // 单字段, 其自需占一个位置查询即可
			values[i] = dest.Addr().Interface()
			break
		} else if isSliceField { // 单行, 多字段查询时
			values[i] = dest.Index(i).Interface()
		}
	}

	if isStruct && len(structMissFields) > 0 {
		return fmt.Errorf("getScanValues is failed, cols %q is miss dest struct", strings.Join(structMissFields, ","))
	}
	return nil
}

// nullScan 空值scan
func (t *Table) nullScan(dest, src interface{}, needUnmarshalCol ...string) (err error) {
	switch val := src.(type) {
	case *sql.NullString:
		if len(needUnmarshalCol) > 0 && needUnmarshalCol[0] != "" { // 判断下是否需要反序列化
			handleColFn := t.handleColFnMap[needUnmarshalCol[0]]
			if val.String != "" {
				err = handleColFn.unmarshal([]byte(val.String), dest)
			}
		} else {
			err = convertAssign(dest, val.String)
		}
		val.String = ""
		cacheNullString.Put(val)
	case *sql.NullInt64:
		err = convertAssign(dest, val.Int64)
		val.Int64 = 0
		cacheNullInt64.Put(val)
	case *sql.NullFloat64:
		err = convertAssign(dest, val.Float64)
	default:
		err = errors.New("unknown null type")
	}
	return
}

// setDestIsSkip 设置值得时候判断是否跳过
func (t *Table) setDestIsSkip(kind reflect.Kind) bool {
	switch kind {
	case reflect.Ptr, reflect.Slice, reflect.Struct:
		return true
	}
	return false
}

// setDest 设置值
func (t *Table) setDest(dest reflect.Value, colTypes []*sql.ColumnType, fieldIndex2NullIndexMap map[int]int, scanResult []interface{}) error {
	if t.destTypeBitmap[structNo] {
		for fieldIndex, nullIndex := range fieldIndex2NullIndexMap {
			destFieldValue := dest.Field(fieldIndex)
			if _, ok := t.handleColFnMap[colTypes[nullIndex].Name()]; ok { // 需要反序列化的
				if err := t.nullScan(destFieldValue.Addr().Interface(), scanResult[nullIndex], colTypes[nullIndex].Name()); err != nil {
					return err
				}
				continue
			}

			// 防止如果 destFieldValue 能反序列化, 但又没有设置反序列化方法, 就跳过赋值操作
			if t.setDestIsSkip(destFieldValue.Kind()) {
				continue
			}

			if err := t.nullScan(destFieldValue.Addr().Interface(), scanResult[nullIndex]); err != nil {
				return err
			}
		}
	} else if t.destTypeBitmap[mapNo] {
		destType := dest.Type()
		if destType.Key().Kind() != reflect.String {
			return errors.New("map key must is string")
		}
		if dest.IsNil() {
			dest.Set(reflect.MakeMapWithSize(destType, len(colTypes)))
		}
		for i, col := range colTypes {
			key := reflect.ValueOf(col.Name())
			val := reflect.Value{}
			nullIndex, ok := fieldIndex2NullIndexMap[i]
			if ok {
				val = reflect.New(destType.Elem())
				if err := t.nullScan(val.Interface(), scanResult[nullIndex]); err != nil {
					return err
				}
			} else {
				val = reflect.ValueOf(scanResult[i])
			}
			dest.SetMapIndex(key, val.Elem())
		}
	} else if t.destTypeBitmap[oneFieldNo] {
		if _, ok := fieldIndex2NullIndexMap[0]; ok {
			return t.nullScan(dest.Addr().Interface(), scanResult[0])
		}
	} else if t.destTypeBitmap[sliceNo] { // QueryRowScan 方法会用, 单行多字段查询
		for _, nullIndex := range fieldIndex2NullIndexMap {
			if err := t.nullScan(dest.Index(nullIndex).Interface(), scanResult[nullIndex]); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetSqlObj 获取 SqlStrObj, 方便外部使用该对象的方法
func (t *Table) GetSqlObj() *SqlStrObj {
	return t.tmpSqlObj
}

// sqlObjIsNil 判断 sqlObj 是否为空
func (t *Table) sqlObjIsNil() bool {
	return t.tmpSqlObj == nil
}

// Where 支持占位符
// 如: Where("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (t *Table) Where(sqlStr string, args ...interface{}) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetWhereArgs(sqlStr, args...)
	return t
}

// OrWhere 支持占位符
// 如: OrWhere("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (t *Table) OrWhere(sqlStr string, args ...interface{}) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetOrWhereArgs(sqlStr, args...)
	return t
}

// OrderBy
func (t *Table) OrderBy(sqlStr string) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetOrderByStr(sqlStr)
	return t
}

// Limit
func (t *Table) Limit(page int32, size int32) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetLimit(page, size)
	return t
}

// Group
func (t *Table) GroupBy(sqlStr string) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetGroupByStr(sqlStr)
	return t
}

// Raw 执行原生操作
// sql sqlStr 或 *SqlStrObj
// 说明: 在使用时, 设置了 tableName 时查询性能更好, 因为在调用 getScanValues 前需要
// 通过 tableName 获取表元信息, 再判断字段是否为 NULL, 在没有表元信息时会将所有查询结果都按 NULL 类型处理
func (t *Table) Raw(sql interface{}) *Table {
	switch val := sql.(type) {
	case string:
		t.tmpSqlObj = NewCacheSql(val)
	case *SqlStrObj:
		t.tmpSqlObj = val
	default:
		cjLog.Error("sql only support string/SqlStrObjPtr")
		// glog.Error("sql only support string/SqlStrObjPtr")
		return nil
	}
	return t
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	defer t.free()
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil, nil
	}
	return t.db.Exec(t.tmpSqlObj.SetPrintLog(t.isPrintSql).SetCallerSkip(t.printSqlCallSkip).GetSqlStr())
}

// QueryRowScan 单行查询
func (t *Table) QueryRowScan(dest ...interface{}) error {
	defer t.free()
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.printSqlCallSkip += 1

	rows, err := t.Query(false)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() { // 没有就为空
		return nullRowErr
	}

	colTypes, _ := rows.ColumnTypes()
	colLen := len(colTypes)
	if colLen != len(dest) {
		return fmt.Errorf("select res len %d, dest len %d", colLen, len(dest))
	}
	values := make([]interface{}, colLen)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	// 将 dest 转为 []dest
	destsReflectValue := reflect.ValueOf(append([]interface{}{}, dest...))
	t.loadDestTypeBitmap(destsReflectValue.Type())
	if err := t.getScanValues(destsReflectValue, nil, fieldIndex2NullIndexMap, colTypes, values); err != nil {
		return err
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	if err := t.setDest(destsReflectValue, colTypes, fieldIndex2NullIndexMap, values); err != nil {
		return err
	}
	return err
}

// Query 多行查询
// 注: 返回的 sql.Rows 需要调用 Close
func (t *Table) Query(isNeedCache ...bool) (*sql.Rows, error) {
	defaultNeedCache := true
	if len(isNeedCache) > 0 {
		defaultNeedCache = isNeedCache[0]
	}

	if defaultNeedCache {
		defer t.free()
	}

	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil, nil
	}
	_ = t.initCacheCol2InfoMap() // 为 getScanValues 解析 NULL 值做准备, 由于调用 Raw 时, 可能会出现没有表面, 所有需要忽略错误
	return t.db.Query(t.tmpSqlObj.SetPrintLog(t.isPrintSql).SetCallerSkip(t.printSqlCallSkip).GetSqlStr())
}

// removeValuePtr 移除多指针
func removeValuePtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// removeTypePtr 移除多指针
func removeTypePtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isExported 是可导出
func isExported(fieldName string) bool {
	if fieldName == "" {
		return false
	}
	first := fieldName[0]
	return first >= 'A' && first <= 'Z'
}

// parseTableName 解析表名
func parseTableName(objName string) string {
	res := new(strings.Builder)
	for i, v := range objName {
		if v >= 'A' && v <= 'Z' {
			if i > 0 {
				res.WriteRune('_')
			}
			res.WriteRune(v | ' ')
			continue
		}
		res.WriteRune(v)
	}
	return res.String()
}

// ========================================= 以下为常用操作的封装 ==================================

// IsNullRow 根据 err 判断是否结果为空
func IsNullRow(err error) bool {
	return err == nullRowErr
}

// ExecForSql 根据 sql 进行执行 INSERT/UPDATE/DELETE 等操作
// sql sqlStr 或 *SqlStrObj
func ExecForSql(db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).Exec()
}

// Count 获取总数
func Count(db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).PrintSqlCallSkip(3).SelectCount().Where(where, args...).Count(dest)
}

// InsertForObj 根据对象新增
func InsertForObj(db DBer, tableName string, src ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Insert(src...).Exec()
}

// UpdateForObj 根据对象更新
func UpdateForObj(db DBer, tableName string, src interface{}, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Update(src, where, args...).Exec()
}

// DeleteWhere 根据条件删除
func DeleteWhere(db DBer, tableName string, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Delete().Where(where, args...).Exec()
}

// FindWhere 查询对象中的字段内容
func FindWhere(db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).PrintSqlCallSkip(3).FindWhere(dest, where, args...)
}

// SelectFindWhere 查询指定内容的
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindWhere(db DBer, fields interface{}, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).FindWhere(dest, where, args...)
}

// SelectFindOne 单行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOne(db DBer, fields interface{}, tableName string, where string, dest ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOne(dest...)
}

// FindOne 单查询
// sql sqlStr 或 *SqlStrObj
func FindOne(db DBer, sql interface{}, dest ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindOne(dest...)
}

// FindOneFn 单查询
// sql sqlStr 或 *SqlStrObj
func FindOneFn(db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindOneFn(dest, fn...)
}

// FindAll 多查询
// sql sqlStr 或 *SqlStrObj
func FindAll(db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindAll(dest, fn...)
}
