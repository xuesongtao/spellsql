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
		"    Name string `json:\"name,omitempty\"`\n" +
		"    Age  int    `json:\"age,omitempty\"`\n" +
		"    Addr string `json:\"addr,omitempty\"`\n" +
		"}"
	sqlObjErr = "tmpSqlObj is null"

	selectForOne uint8 = 1 // 单条查询
	selectForAll uint8 = 2 // 多条查询
)

var (
	cacheTableName2ColInfoMap    = sync.Map{} // 缓存表的字段元信息
	cacheStructTag2FiledIndexMap = sync.Map{} // 缓存结构体 tag 对应的 filed index
)

type HandleSelectRowFn func(_rowModel interface{}) error // 对每行查询结果进行取出处理

// TableColInfo 表列详情
type TableColInfo struct {
	Field   string
	Type    string
	Null    string
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

// SetName 设置表名
func (t *Table) SetName(tableName string) *Table {
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
func (t *Table) skip(filedInfo reflect.StructField) bool {
	switch filedInfo.Type.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Struct:
		return true
	}

	return !isExported(filedInfo.Name)
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
	filedNum := ty.NumField()
	columns = make([]string, 0, filedNum)
	values = make([]interface{}, 0, filedNum)
	for i := 0; i < filedNum; i++ {
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
		// 判断字段是否有效, 由于 json tag 使用比较多, 所有需要与数据库字段取交, 避免查询报错
		if t.tag == defaultTableTag {
			if tableFiled, ok := t.cacheCol2InfoMap[column]; !ok {
				continue
			} else {
				if isExcludePri && tableFiled.Key == "PRI" { // 主键, 防止更新
					continue
				}
			}
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
		err = fmt.Errorf("you should sure struct is ok, eg:%s", structErr)
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
// fileds 多个通过逗号隔开
func (t *Table) Select(fileds string) *Table {
	if fileds == "" {
		cjLog.Error("fileds is null")
		// glog.Error("fileds is null")
		return nil
	}

	if t.name == "" {
		cjLog.Error("table is unknown")
		// glog.Error("table is unknown")
		return nil
	}

	t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", fileds, t.name)
	return t
}

//  SelectAll() 查询所有字段
func (t *Table) SelectAll() *Table {
	return t.Select("*")
}

// Count 获取总数
func (t *Table) Count(total interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	return t.Raw(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetTotalSqlStr()).QueryRowScan(total)
}

// Find 单行查询
// dest 长度 > 1 时, 支持多个字段查询
// dest 长度 == 1 时, 支持 struct/单字段
func (t *Table) FindOne(dest ...interface{}) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	t.tmpSqlObj.SetLimitStr("1")
	if len(dest) > 1 {
		return t.find(dest)
	}
	return t.QueryRowScan(dest...)
}

// FindAll 多行查询
func (t *Table) FindAll(dest interface{}, fn ...HandleSelectRowFn) error {
	if t.sqlObjIsNil() {
		t.SelectAll()
	}
	return t.find(dest, fn...)
}

// FindWhere 如果没有添加查询字段内容, 会根据输入对象进行解析查询
// dest 支持 struct, slice, 单字段
func (t *Table) FindWhere(dest interface{}, where string, args ...interface{}) error {
	tv := removeValuePtr(reflect.ValueOf(dest))
	if t.sqlObjIsNil() {
		selectFileds := make([]string, 0, 5)
		switch tv.Kind() {
		case reflect.Struct, reflect.Slice:
			ty := tv.Type()
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
			col2FiledIndexMap := t.parseCol2FiledIndex(ty)
			for col := range col2FiledIndexMap {
				// 排除结构体中的字段, 数据库没有
				if _, ok := t.cacheCol2InfoMap[col]; !ok {
					continue
				}
				selectFileds = append(selectFileds, col)
			}
			t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", strings.Join(selectFileds, ","), t.name)
		default:
			return errors.New("dest must struct/slice ptr")
		}
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)
	return t.find(dest)
}

// parseCol2FiledIndex 解析列对应的结构体偏移值
func (t *Table) parseCol2FiledIndex(ty reflect.Type) map[string]int {
	// 非结构体就返回空
	if ty.Kind() != reflect.Struct {
		return nil
	}

	// 通过地址来取, 防止出现重复
	if cacheVal, ok := cacheStructTag2FiledIndexMap.Load(ty); ok {
		return cacheVal.(map[string]int)
	}
	filedNum := ty.NumField()
	col2FiledIndexMap := make(map[string]int, filedNum)
	for i := 0; i < filedNum; i++ {
		structField := ty.Field(i)
		if t.skip(structField) {
			continue
		}
		val := structField.Tag.Get(t.tag)
		if val == "" {
			continue
		}
		col2FiledIndexMap[t.parseTag2Col(val)] = i
	}

	cacheStructTag2FiledIndexMap.Store(ty, col2FiledIndexMap)
	return col2FiledIndexMap
}

// find 查询
func (t *Table) find(dest interface{}, fn ...HandleSelectRowFn) error {
	ty := reflect.TypeOf(dest)
	switch ty.Kind() {
	case reflect.Ptr, reflect.Slice:
	default:
		return errors.New("dest it should ptr/slice")
	}
	ty = removeTypePtr(ty)
	switch ty.Kind() {
	case reflect.Struct:
		return t.queryScan(ty, selectForOne, false, dest, fn...)
	case reflect.Slice:
		ty = ty.Elem()
		isPtr := ty.Kind() == reflect.Ptr
		if isPtr {
			ty = removeTypePtr(ty) // 找到结构体
		}
		return t.queryScan(ty, selectForAll, isPtr, dest, fn...)
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
func (t *Table) queryScan(ty reflect.Type, selectType uint8, isPtr bool, dest interface{}, fn ...HandleSelectRowFn) error {
	rows, err := t.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	colTypes, _ := rows.ColumnTypes()
	col2FiledIndexMap := t.parseCol2FiledIndex(ty)
	destReflectValue := reflect.Indirect(reflect.ValueOf(dest))
	for rows.Next() {
		tmp := reflect.New(ty).Elem()
		filedIndex2NullIndexMap, isStruct, values := t.getScanValues(tmp, col2FiledIndexMap, colTypes)
		if !isStruct && len(values) > 1 {
			return errors.New("check scan is failed, your dest is not ok")
		}

		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}

		if err := t.setDest(tmp, filedIndex2NullIndexMap, values); err != nil {
			return err
		}

		if len(fn) == 1 { // 用于将查询结果暴露出去
			fn[0](tmp.Interface())
		}

		if selectType == selectForAll { // 切片类型(结构体/单字段)
			if isPtr { // 判断下切片中是指针类型还是值类型
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
func (t *Table) getScanValues(dest reflect.Value, col2FiledIndexMap map[string]int, colTypes []*sql.ColumnType) (filedIndex2NullIndexMap map[int]int, isStruct bool, values []interface{}) {
	l := len(colTypes)
	values = make([]interface{}, l)

	// 判断下是否为结构体, 结构体才给结构体里的值进行处理
	isStruct = dest.Kind() == reflect.Struct
	filedIndex2NullIndexMap = make(map[int]int, l)
	for i, colType := range colTypes {
		var (
			filedIndex       int
			structFiledExist bool = true
		)

		if isStruct {
			filedIndex, structFiledExist = col2FiledIndexMap[colType.Name()]
		}

		// NULL 值处理, 防止 sql 报错, 否则就直接 scan 到 struct 字段值
		canNull, _ := colType.Nullable()
		if canNull || !structFiledExist {
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
			if isStruct && structFiledExist {
				filedIndex2NullIndexMap[filedIndex] = i
			}

			// 单字段, 为了减少创建标记, 借助 filedIndex2NullIndexMap 用于标识单字段是否包含空值
			if !isStruct {
				filedIndex2NullIndexMap[-1] = i // 在 setDest 使用
			}
			continue
		}

		// 处理数据库字段非 NULL 部分
		if isStruct { // 结构体
			values[i] = dest.Field(filedIndex).Addr().Interface()
		} else { // 单字段
			values[i] = dest.Addr().Interface()
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
func (t *Table) setDest(dest reflect.Value, filedIndex2NullIndexMap map[int]int, scanResult []interface{}) error {
	// 说明直接映射到里 dest
	if len(filedIndex2NullIndexMap) == 0 {
		return nil
	}

	// 非结构体
	if _, ok := filedIndex2NullIndexMap[-1]; ok {
		return t.nullScan(dest.Addr().Interface(), scanResult[0])
	}

	// 结构体
	for filedIndex, nullIndex := range filedIndex2NullIndexMap {
		err := t.nullScan(dest.Field(filedIndex).Addr().Interface(), scanResult[nullIndex])
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
	return t.db.QueryRow(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr()).Scan(dest...)
}

// Query 多行查询
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
func isExported(filedName string) bool {
	if filedName == "" {
		return false
	}
	first := filedName[0]
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
