package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	// "github.com/gogf/gf/os/glog"
)

const (
	defaultTableTag = "json"
	structErr       = "type User struct {\n" +
		"    Name string `json:\"name\"`\n" +
		"    Age  int    `json:\"age\"`\n" +
		"    Addr string `json:\"addr\"`\n" +
		"}"
	sqlObjErr = "tmpSqlObj is nil"

	selectForOne uint8 = 1 // 单条查询
	selectForAll uint8 = 2 // 多条查询
)

var (
	cacheTableName2ColInfoMap    = sync.Map{}                   // 缓存表的字段元信息
	cacheStructTag2FieldIndexMap = sync.Map{}                   // 缓存结构体 tag 对应的 field index
	NoDataErr                    = errors.New("result is null") // 查询结果为空
)

type SelectCallBackFn func(_row interface{}) error // 对每行查询结果进行取出处理

// TableColInfo 表列详情
type TableColInfo struct {
	Field   string // 字段名
	Type    string // 数据库类型
	Null    string // 是否为 NULL
	Key     string
	Default sql.NullString
	Extra   string
}

type Table struct {
	db               DBer
	isPrintSql       bool       // 是否打印sql
	tmpSqlObj        *SqlStrObj // 暂存对象
	tag              string     // 解析字段的tag
	name             string
	cacheCol2InfoMap map[string]*TableColInfo // 记录该表的所有字段名
}

// NewTable 初始化
// args 支持两个参数
// args[0]: 会解析为 tableName, 这里如果有值, 在进行操作表的时候就会以此表为准,
// 如果为空时, 在通过对象进行操作时按驼峰规则进行解析表名, 解析规则如: UserInfo => user_info
// args[1]: 会解析为待解析的 tag
func NewTable(db DBer, args ...string) *Table {
	if db == nil {
		return nil
	}

	t := &Table{
		db:         db,
		isPrintSql: true,
		tag:        defaultTableTag,
	}

	switch len(args) {
	case 1:
		t.name = args[0]
	case 2:
		t.name = args[0]
		t.tag = args[1]
	}
	return t
}

// IsPrintSql 是否打印 sql
func (t *Table) IsPrintSql(is bool) *Table {
	t.isPrintSql = is
	return t
}

// Name 设置表名
func (t *Table) Name(tableName string) *Table {
	t.name = tableName
	return t
}

// initCacheCol2InfoMap 初始化表字段map, 由于json tag应用比较多, 为了在后续执行insert等通过对象取值会存在取值错误现象, 所以需要预处理下
func (t *Table) initCacheCol2InfoMap() error {
	// 已经初始化过了
	if t.cacheCol2InfoMap != nil {
		return nil
	}

	// 先判断下缓存中有没有
	if info, ok := cacheTableName2ColInfoMap.Load(t.name); ok {
		t.cacheCol2InfoMap, ok = info.(map[string]*TableColInfo)
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
	t.cacheCol2InfoMap = make(map[string]*TableColInfo, len(columns))
	for rows.Next() {
		var info TableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		t.cacheCol2InfoMap[info.Field] = &info
	}

	cacheTableName2ColInfoMap.Store(t.name, t.cacheCol2InfoMap)
	return nil
}

// skip 跳过嵌套, 包含: 对象, 指针对象, 切片, 不可导出字段
func (t *Table) skip(fieldInfo reflect.StructField) bool {
	switch fieldInfo.Type.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Struct:
		return true
	}

	return !isExported(fieldInfo.Name)
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
	t.initCacheCol2InfoMap()
	fieldNum := ty.NumField()
	columns = make([]string, 0, fieldNum)
	values = make([]interface{}, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		structField := ty.Field(i)
		if t.skip(structField) {
			continue
		}

		column := structField.Tag.Get(t.tag)
		if column == "" {
			continue
		}

		// 排除tag中包含的其他的内容
		column = t.parseTag2Col(column)

		// 判断下数据库字段是否存在
		tableField, ok := t.cacheCol2InfoMap[column]
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
		columns = append(columns, column)
		values = append(values, val.Interface())
	}

	if len(columns) == 0 || len(values) == 0 {
		err = fmt.Errorf("you should sure struct is ok, eg: %s", structErr)
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
func (t *Table) Insert(insertObjs ...interface{}) (sql.Result, error) {
	if len(insertObjs) == 0 {
		return nil, errors.New("insertObjs is empty")
	}

	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		columns, values, err := t.getHandleTableCol2Val(insertObj, true, t.name)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			insertSql = NewCacheSql("INSERT INTO ?v (?v)", t.name, strings.Join(columns, ", "))
		}
		insertSql.SetInsertValues(values...)
	}
	t.tmpSqlObj = insertSql
	return t.Exec()
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
			t.tmpSqlObj.SetWhereArgs("?v=?", k, v)
		}
	} else {
		if t.name == "" {
			cjLog.Error("table name is null")
			// glog.Error("table name is null")
			return nil
		}
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
func (t *Table) Update(updateObj interface{}) *Table {
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
		t.tmpSqlObj.SetUpdateValueArgs("?v=?", k, v)
	}
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
		cjLog.Error("table name is unknown")
		// glog.Error("table is unknown")
		return nil
	}

	t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", fields, t.name)
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
	return t.db.QueryRow(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetTotalSqlStr()).Scan(total)
}

// FindOne 单行查询
// dest 长度 > 1 时, 支持多个字段查询
// dest 长度 == 1 时, 支持 struct/单字段
func (t *Table) FindOne(dest ...interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	t.tmpSqlObj.SetLimitStr("1")
	if len(dest) == 1 {
		return t.find(dest[0])
	}
	return t.QueryRowScan(dest...)
}

// FindAll 多行查询
func (t *Table) FindAll(dest interface{}, fn ...SelectCallBackFn) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	return t.find(dest, fn...)
}

// FindWhere 如果没有添加查询字段内容, 会根据输入对象进行解析查询
// dest 支持 struct, slice, 单字段
func (t *Table) FindWhere(dest interface{}, where string, args ...interface{}) error {
	if t.sqlObjIsNil() {
		ty := removeTypePtr(reflect.TypeOf(dest))
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
			t.initCacheCol2InfoMap()
			col2FieldIndexMap := t.parseCol2FieldIndex(ty)
			for col := range col2FieldIndexMap {
				// 排除结构体中的字段, 数据库没有
				if _, ok := t.cacheCol2InfoMap[col]; !ok {
					continue
				}
				selectFields = append(selectFields, col)
			}
			t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", strings.Join(selectFields, ", "), t.name)
		default:
			t.SelectAll()
		}
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)
	return t.find(dest)
}

// parseCol2FieldIndex 通过解析输入结构体, 返回 map[tag 名]字段偏移量, 同时缓存起来
func (t *Table) parseCol2FieldIndex(ty reflect.Type) map[string]int {
	// 非结构体就返回空
	if ty.Kind() != reflect.Struct {
		return nil
	}

	// 通过地址来取, 防止出现重复
	if cacheVal, ok := cacheStructTag2FieldIndexMap.Load(ty); ok {
		return cacheVal.(map[string]int)
	}
	fieldNum := ty.NumField()
	col2FieldIndexMap := make(map[string]int, fieldNum)
	for i := 0; i < fieldNum; i++ {
		structField := ty.Field(i)
		if t.skip(structField) {
			continue
		}
		val := structField.Tag.Get(t.tag)
		if val == "" {
			continue
		}
		col2FieldIndexMap[t.parseTag2Col(val)] = i
	}

	cacheStructTag2FieldIndexMap.Store(ty, col2FieldIndexMap)
	return col2FieldIndexMap
}

// find 查询
func (t *Table) find(dest interface{}, fn ...SelectCallBackFn) error {
	ty := reflect.TypeOf(dest)
	switch ty.Kind() {
	case reflect.Ptr:
	default:
		return errors.New("dest it should ptr")
	}
	ty = removeTypePtr(ty)
	switch ty.Kind() {
	case reflect.Struct:
		return t.queryScan(ty, selectForOne, false, dest, fn...)
	case reflect.Slice:
		ty = ty.Elem()
		sliceValIsPtr := ty.Kind() == reflect.Ptr
		if sliceValIsPtr {
			ty = removeTypePtr(ty) // 找到结构体
		}
		return t.queryScan(ty, selectForAll, sliceValIsPtr, dest, fn...)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return t.queryScan(ty, selectForOne, false, dest, fn...)
	default:
		return errors.New("dest kind not found")
	}
}

// queryScan 将数据库查询的内容映射到目标对象
func (t *Table) queryScan(ty reflect.Type, selectType uint8, sliceValIsPtr bool, dest interface{}, fn ...SelectCallBackFn) error {
	rows, err := t.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	colTypes, _ := rows.ColumnTypes()
	colLen := len(colTypes)
	col2FieldIndexMap := t.parseCol2FieldIndex(ty)
	destReflectValue := reflect.Indirect(reflect.ValueOf(dest))
	for rows.Next() {
		tmp := reflect.New(ty).Elem()
		fieldIndex2NullIndexMap, canScanValueNum, values := t.getScanValues(tmp, col2FieldIndexMap, colTypes)
		if colLen != canScanValueNum {
			return fmt.Errorf("check scan is failed, select result len %d, dest len %d", colLen, canScanValueNum)
		}

		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}

		if err := t.setDest(tmp, fieldIndex2NullIndexMap, values); err != nil {
			return err
		}

		if len(fn) == 1 { // 回调方法
			if sliceValIsPtr { // 指针类型
				if err := fn[0](tmp.Addr().Interface()); err != nil {
					return err
				}
			} else { // 值类型
				if err := fn[0](tmp.Interface()); err != nil {
					return err
				}
			}
		}

		if selectType == selectForAll { // 切片类型(结构体/单字段)
			if sliceValIsPtr { // 判断下切片中是指针类型还是值类型
				destReflectValue.Set(reflect.Append(destReflectValue, tmp.Addr()))
			} else {
				destReflectValue.Set(reflect.Append(destReflectValue, tmp))
			}
		} else { // 结构体
			destReflectValue.Set(tmp)
		}
	}
	return nil
}

// getScanValues 获取待 Scan 的内容
func (t *Table) getScanValues(dest reflect.Value, col2FieldIndexMap map[string]int, colTypes []*sql.ColumnType) (fieldIndex2NullIndexMap map[int]int, canScanValueNum int, values []interface{}) {
	l := len(colTypes)
	values = make([]interface{}, l)

	// 判断下是否为结构体, 结构体才给结构体里的值进行处理
	isStruct := dest.Kind() == reflect.Struct
	fieldIndex2NullIndexMap = make(map[int]int, l)
	for i, colType := range colTypes {
		var (
			fieldIndex       int
			structFieldExist bool = true
		)

		if isStruct {
			fieldIndex, structFieldExist = col2FieldIndexMap[colType.Name()]
		}

		if !structFieldExist {
			// 说明结构里查询的值不存在
			cjLog.Errorf("col %q no found struct dest", colType.Name())
			// glog.Errorf("col %q no found struct dest", colType.Name())
			continue
		}

		canScanValueNum++
		// NULL 值处理, 防止 sql 报错, 否则就直接 scan 到 struct 字段值
		canNull, _ := colType.Nullable()
		if canNull {
			// fmt.Println(colType.Name(), colType.ScanType().Name())
			switch colType.ScanType().Name() {
			case "NullInt64":
				values[i] = new(sql.NullInt64)
			case "NullFloat64":
				values[i] = new(sql.NullFloat64)
			default:
				values[i] = new(sql.NullString)
			}

			// 结构体, 这里记录 struct 那个字段需要映射 NULL 值
			if structFieldExist {
				fieldIndex2NullIndexMap[fieldIndex] = i
			}

			// 单字段, 为了减少创建标记, 借助 fieldIndex2NullIndexMap 用于标识单字段是否包含空值
			if !isStruct {
				fieldIndex2NullIndexMap[-1] = i // 在 setDest 使用
				break
			}
			continue
		}

		// 处理数据库字段非 NULL 部分
		if isStruct { // 结构体
			values[i] = dest.Field(fieldIndex).Addr().Interface()
		} else { // 单字段
			values[i] = dest.Addr().Interface()
			break
		}
	}
	return
}

// nullScan 空值scan
func (t *Table) nullScan(dest, src interface{}) (err error) {
	switch val := src.(type) {
	case *sql.NullString:
		err = convertAssign(dest, val.String)
	case *sql.NullInt64:
		err = convertAssign(dest, val.Int64)
	case *sql.NullFloat64:
		err = convertAssign(dest, val.Float64)
	default:
		err = errors.New("unknown null type")
	}
	return
}

// setDest 设置值
func (t *Table) setDest(dest reflect.Value, fieldIndex2NullIndexMap map[int]int, scanResult []interface{}) error {
	// 说明已经映射到 dest, 不需要再处理
	if len(fieldIndex2NullIndexMap) == 0 {
		return nil
	}

	// 非结构体, 只会出现单值
	if _, ok := fieldIndex2NullIndexMap[-1]; ok {
		return t.nullScan(dest.Addr().Interface(), scanResult[0])
	}

	// 结构体
	for fieldIndex, nullIndex := range fieldIndex2NullIndexMap {
		err := t.nullScan(dest.Field(fieldIndex).Addr().Interface(), scanResult[nullIndex])
		if err != nil {
			return err
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
func (t *Table) Group(groupSqlStr string) *Table {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}
	t.tmpSqlObj.SetGroupByStr(groupSqlStr)
	return t
}

// Raw 执行原生操作
// sql sqlStr 或 *SqlStrObj
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
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil, nil
	}
	return t.db.Exec(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr())
}

// QueryRowScan 单行查询
func (t *Table) QueryRowScan(dest ...interface{}) error {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil
	}

	err := t.db.QueryRow(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr()).Scan(dest...)
	if err == sql.ErrNoRows {
		cjLog.Error(err.Error())
		// glog.Error(err.Error())
		return NoDataErr
	}
	return err
}

// Query 多行查询, 返回的 sql.Rows 需要调用 Close
func (t *Table) Query() (*sql.Rows, error) {
	if t.sqlObjIsNil() {
		cjLog.Error(sqlObjErr)
		// glog.Error(sqlObjErr)
		return nil, nil
	}
	return t.db.Query(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr())
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

// Count 获取总数
func Count(db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).SelectCount().Where(where, args...).Count(dest)
}

// InsertForObj 根据对象新增
func InsertForObj(db DBer, tableName string, obj ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Insert(obj...)
}

// DeleteForObj 根据对象删除
func DeleteForObj(db DBer, tableName string, obj interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Delete(obj).Exec()
}

// UpdateForObj 根据对象更新
func UpdateForObj(db DBer, tableName string, obj interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Update(obj).Exec()
}

// FindWhere 查询对象中的字段内容
func FindWhere(db DBer, tableName string, obj interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).FindWhere(obj, where, args...)
}

// InsertForSql 根据 sql 新增
// sql sqlStr 或 *SqlStrObj
func InsertForSql(db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).Raw(sql).Exec()
}

// DeleteForSql 根据 sql 删除
// sql sqlStr 或 *SqlStrObj
func DeleteForSql(db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).Raw(sql).Exec()
}

// UpdateForSql 根据 sql 更新
// sql sqlStr 或 *SqlStrObj
func UpdateForSql(db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).Raw(sql).Exec()
}

// FindOne 单查询
// sql sqlStr 或 *SqlStrObj
func FindOne(db DBer, sql interface{}, dest ...interface{}) error {
	return NewTable(db).Raw(sql).FindOne(dest...)
}

// FindAll 多查询
// sql sqlStr 或 *SqlStrObj
func FindAll(db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Raw(sql).FindAll(dest, fn...)
}
