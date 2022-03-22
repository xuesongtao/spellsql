package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

const (
	defaultTableTag = "json"
	structErr       = "type User struct {\n" +
		"    Name string `json:\"name,omitempty\"`\n" +
		"    Age  int    `json:\"age,omitempty\"`\n" +
		"    Addr string `json:\"addr,omitempty\"`\n" +
		"}"

	selectForOne uint8 = 1 // 单条查询
	selectForAll uint8 = 2 // 多条查询
)

var (
	tableColInfoSyncMap = sync.Map{}
)

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
	db          DBer
	isPrintSql  bool       // 是否打印sql
	tmpSqlObj   *SqlStrObj // 暂存对象
	tag         string     // 解析字段的tag
	name        string
	col2InfoMap map[string]*TableColInfo // 记录该表的所有字段名
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

// PrintSql 是否打印 sql
func (t *Table) PrintSql(is bool) *Table {
	t.isPrintSql = is
	return t
}

// initCol2InfoMap 初始化表字段map, 由于json tag应用比较多, 为了在后续执行insert等通过对象取值会存在取值错误现象, 所以需要预处理下
func (t *Table) initCol2InfoMap() error {
	// 已经初始化过了
	if t.col2InfoMap != nil {
		return nil
	}

	// 先判断下缓存中有没有
	if info, ok := tableColInfoSyncMap.Load(t.name); ok {
		t.col2InfoMap, ok = info.(map[string]*TableColInfo)
		if ok {
			return nil
		}
	}

	sqlStr := GetSqlStr("SHOW COLUMNS FROM ?v", t.name)
	rows, err := t.db.Query(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	t.col2InfoMap = make(map[string]*TableColInfo, len(columns))
	for rows.Next() {
		var info TableColInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		t.col2InfoMap[info.Field] = &info
	}

	tableColInfoSyncMap.Store(t.name, t.col2InfoMap)
	return nil
}

// parseTable 解析字段
func (t *Table) parseTable(v interface{}, isExcludePri bool, tableName ...string) (columns []string, values []interface{}, err error) {
	tv, err := getStructReflectValue(v)
	if err != nil {
		return
	}

	ty := tv.Type()
	if t.name == "" {
		t.name = parseTableName(ty.Name())
	}
	t.initCol2InfoMap()
	filedNum := ty.NumField()
	columns = make([]string, 0, filedNum)
	values = make([]interface{}, 0, filedNum)
	for i := 0; i < filedNum; i++ {
		structField := ty.Field(i)
		if structField.Anonymous || !isExported(structField.Name) {
			continue
		}

		column := structField.Tag.Get(t.tag)
		if column == "" {
			continue
		}

		// 排除tag中包含的其他的内容
		column = t.parseTag2TableCol(column)
		// 判断字段是否有效
		if t.tag == defaultTableTag {
			if tableFiled, ok := t.col2InfoMap[column]; !ok {
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

// parseTag2TableCol 解析tag中表的列名
func (t *Table) parseTag2TableCol(tag string) (column string) {
	tmpIndex := IndexForBF(true, tag, ",")
	if tmpIndex > -1 {
		column = tag[:tmpIndex]
	} else {
		column = tag
	}
	return
}

// Insert 提交, 支持批量提交
// tableName 如果有值会以此为准, 反之会通过输入对象按驼峰转为表面
func (t *Table) Insert(insertObjs ...interface{}) (sql.Result, error) {
	if len(insertObjs) == 0 {
		return nil, errors.New("insertObjs is empty")
	}

	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		columns, values, err := t.parseTable(insertObj, true, t.name)
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
		columns, values, err := t.parseTable(deleteObj[0], false, t.name)
		if err != nil {
			Error("parseTable is failed, err:", err)
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
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
func (t *Table) Update(updateObj interface{}) *Table {
	columns, values, err := t.parseTable(updateObj, true, t.name)
	if err != nil {
		Error("parseTable is failed, err:", err)
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
		Error("fileds is null")
		return nil
	}

	if t.name == "" {
		Error("table is unknown")
		return nil
	}
	t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", fileds, t.name)
	return t
}

// Count 获取总数
func (t *Table) Count(total interface{}) error {
	return t.Raw(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetTotalSqlStr()).QueryRowScan(total)
}

// Find 单行查询
func (t *Table) FindOne(dest interface{}) error {
	t.tmpSqlObj.SetLimitStr("1")
	return t.find(selectForOne, dest)
}

// 多行查询
func (t *Table) FindAll(dest interface{}) error {
	return t.find(selectForAll, dest)
}

// parseCol2FiledIndex 解析列对应的结构体偏移值
func (t *Table) parseCol2FiledIndex(ty reflect.Type) map[string]int {
	filedNum := ty.NumField()
	column2IndexMap := make(map[string]int, filedNum)
	for i := 0; i < filedNum; i++ {
		structFiled := ty.Field(i)
		val := structFiled.Tag.Get(t.tag)
		if val == "" {
			continue
		}
		column2IndexMap[t.parseTag2TableCol(val)] = i
	}
	return column2IndexMap
}

// find 查询
func (t *Table) find(selectType uint8, dest interface{}) error {
	ty := reflect.TypeOf(dest)
	switch ty.Kind() {
	case reflect.Ptr, reflect.Slice:
	default:
		return errors.New("dest it should ptr/slice")
	}

	ty = removeTypePtr(ty)
	switch ty.Kind() {
	case reflect.Struct:
		return t.queryScan(ty, selectType, false, dest)
	case reflect.Slice:
		ty = ty.Elem()
		isPtr := ty.Kind() == reflect.Ptr
		if isPtr {
			ty = removeTypePtr(ty) // 找到结构体
		}

		// 非结构体就为单字段查询
		if ty.Kind() != reflect.Struct {
			return errors.New("it should slice struct")
		}
		return t.queryScan(ty, selectType, isPtr, dest)
	default:
		if selectType == selectForOne {
			return errors.New("dest it should struct, or you can call QueryRowScan/Query")
		}
		return errors.New("dest it should slice")
	}
}

// queryScan 将数据库查询的内容映射到目标对象
func (t *Table) queryScan(ty reflect.Type, selectType uint8, isPtr bool, dest interface{}) error {
	rows, err := t.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()
	column2IndexMap := t.parseCol2FiledIndex(ty)
	destReflectValue := reflect.Indirect(reflect.ValueOf(dest))
	for rows.Next() {
		values := t.getScanValues(colTypes)
		if err := rows.Scan(values...); err != nil {
			Error("mysql scan is failed, err:", err)
			continue
		}
		tmpStruct := reflect.New(ty).Elem()
		if err := t.setDest(tmpStruct, columns, column2IndexMap, values); err != nil {
			return err
		}

		if selectType == selectForAll { // 切片类型
			if isPtr { // 判断下切片中是指针类型还是值类型
				destReflectValue.Set(reflect.Append(destReflectValue, tmpStruct.Addr()))
			} else {
				destReflectValue.Set(reflect.Append(destReflectValue, tmpStruct))

			}
		} else { // 结构体
			destReflectValue.Set(tmpStruct)
		}
	}
	return nil
}

// getScanValues 获取待 Scan 的内容
func (t *Table) getScanValues(colTypes []*sql.ColumnType) (values []interface{}) {
	values = make([]interface{}, len(colTypes))
	for i, colType := range colTypes {
		values[i] = t.initScanValue(colType.DatabaseTypeName())
	}
	return
}

// initScanValue 这里也是仅列出了常用的类型，如需扩展再进行类型添加
func (t *Table) initScanValue(dbType string) interface{} {
	switch dbType {
	case "TINYINT", "SMALLINT", "INT", "MEDIUMINT":
		return new(int32)
	case "BIGINT":
		return new(int64)
	case "FLOAT":
		return new(float32)
	case "DOUBLE":
		return new(float64)
	default:
		return new(sql.NullString)
	}
}

// setDest 设置值
func (t *Table) setDest(dest reflect.Value, cols []string, col2IndexMap map[string]int, scanResult []interface{}) error {
	for i, col := range cols {
		filedIndex, ok := col2IndexMap[col]
		if !ok {
			continue
		}
		switch val := scanResult[i].(type) {
		case *sql.NullString:
			err := convertAssign(dest.Field(filedIndex).Addr().Interface(), val.String)
			if err != nil {
				return err
			}
		default:
			reflectVal := reflect.ValueOf(val)
			err := convertAssign(dest.Field(filedIndex).Addr().Interface(), reflect.Indirect(reflectVal).Interface())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Where 支持占位符
// 如: Where("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (t *Table) Where(sqlStr string, args ...interface{}) *Table {
	t.tmpSqlObj.SetWhereArgs(sqlStr, args...)
	return t
}

// OrWhere 支持占位符
// 如: OrWhere("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (t *Table) OrWhere(sqlStr string, args ...interface{}) *Table {
	t.tmpSqlObj.SetOrWhereArgs(sqlStr, args...)
	return t
}

// OrderBy
func (t *Table) OrderBy(sqlStr string) *Table {
	t.tmpSqlObj.SetOrderByStr(sqlStr)
	return t
}

// Limit
func (t *Table) Limit(page int32, size int32) *Table {
	t.tmpSqlObj.SetLimit(page, size)
	return t
}

// Group
func (t *Table) Group(groupSqlStr string) *Table {
	t.tmpSqlObj.SetGroupByStr(groupSqlStr)
	return t
}

// Raw 执行原生sqlStr
func (t *Table) Raw(sqlStr string) *Table {
	t.tmpSqlObj = NewCacheSql(sqlStr)
	return t
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	return t.db.Exec(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr())
}

// QueryRowScan 单行查询
func (t *Table) QueryRowScan(dest ...interface{}) error {
	return t.db.QueryRow(t.tmpSqlObj.SetPrintLog(t.isPrintSql).GetSqlStr()).Scan(dest...)
}

// Query 多行查询
func (t *Table) Query() (*sql.Rows, error) {
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

// getStructReflectValue
func getStructReflectValue(v interface{}) (reflect.Value, error) {
	tv := removeValuePtr(reflect.ValueOf(v))
	if tv.Kind() != reflect.Struct {
		return tv, errors.New("it must is struct")
	}
	return tv, nil
}
