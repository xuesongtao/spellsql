package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	defaultTableTag = "json"
	structErr       = "type User struct {\n" +
		"    Name string `json:\"name,omitempty\"`\n" +
		"    Age  int    `json:\"age,omitempty\"`\n" +
		"    Addr string `json:\"addr,omitempty\"`\n" +
		"}"
)

// TableFiledInfo 表列详情
type TableFiledInfo struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

type Table struct {
	db        DBer
	tmpSqlObj *SqlStrObj // 暂存对象
	tag       string     // 解析字段的tag
	name      string
	filedMap  map[string]struct{} // 记录该表的所有字段名, 用于json tag的时候
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
		db:  db,
		tag: defaultTableTag,
	}

	switch len(args) {
	case 1:
		t.name = args[0]
	case 2:
		t.name = args[0]
		t.tag = args[1]
	}

	// 由于 json 应用比较多, 在后续执行insert等通过对象取值会存在取值错误现象, 所以需要预处理下
	if t.tag == defaultTableTag && t.name != "" {
		if err := t.initFileMap(); err != nil {
			Error("initFileMap is failed, err:", err)
			return nil
		}
	}
	return t
}

// initFileMap 初始化表字段map
func (t *Table) initFileMap() error {
	// 已经初始化过了
	if t.filedMap != nil {
		return nil
	}

	sqlStr := GetSqlStr("SHOW COLUMNS FROM ?v", t.name)
	rows, err := t.db.Query(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	l := len(columns)
	t.filedMap = make(map[string]struct{}, l)
	for rows.Next() {
		var info TableFiledInfo
		err = rows.Scan(&info.Field, &info.Type, &info.Null, &info.Key, &info.Default, &info.Extra)
		if err != nil {
			return fmt.Errorf("mysql scan is failed, err: %v", err)
		}
		// 由于mysql, tidb 第一个字段为字段名, 所有这样处理
		t.filedMap[info.Field] = struct{}{}
	}
	return nil
}

// parseTable 解析字段
func (t *Table) parseTable(v interface{}, filedTag string, tableName ...string) (columns []string, values []interface{}, err error) {
	tv, err := getStructReflectValue(v)
	if err != nil {
		return
	}

	ty := tv.Type()
	if t.name == "" {
		t.name = parseTableName(ty.Name())
		t.initFileMap()
	}
	filedNum := ty.NumField()
	columns = make([]string, 0, filedNum)
	values = make([]interface{}, 0, filedNum)
	for i := 0; i < filedNum; i++ {
		structField := ty.Field(i)
		if structField.Anonymous || !isExported(structField.Name) {
			continue
		}

		column := structField.Tag.Get(filedTag)
		if column == "" {
			continue
		}

		// 排除tag中包含的其他的内容
		column = t.parseTagTableField(column)

		// 判断字段是否有效
		if t.tag == defaultTableTag {
			if _, ok := t.filedMap[column]; !ok {
				continue
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

// parseTagTableField 解析tag中表的列名
func (t *Table) parseTagTableField(tag string) (column string) {
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
		columns, values, err := t.parseTable(insertObj, t.tag, t.name)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			insertSql = NewCacheSql("INSERT INTO ?v (?v)", t.name, strings.Join(columns, ", "))
		}
		insertSql.SetInsertValues(values...)
	}
	return t.db.Exec(insertSql.GetSqlStr())
}

// Delete 会以对象中有值得为条件进行删除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		columns, values, err := t.parseTable(deleteObj[0], t.tag, t.name)
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
	columns, values, err := t.parseTable(updateObj, t.tag, t.name)
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
	t.tmpSqlObj = NewCacheSql("SELECT ?v FROM ?v", fileds, t.name)
	return t
}

// Count 获取总数
func (t *Table) Count(total interface{}) error {
	return t.db.QueryRow(t.tmpSqlObj.GetTotalSqlStr()).Scan(total)
}

// Find 单行查询
func (t *Table) Find(res interface{}) error {
	return t.find(res)
}

// 多行查询
func (t *Table) FindAll(res interface{}) error {
	return nil
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
		column2IndexMap[t.parseTagTableField(val)] = i
	}
	return column2IndexMap
}

func (t *Table) find(res interface{}) error {
	tv := reflect.ValueOf(res)
	switch tv.Kind() {
	case reflect.Ptr:
		dest := removeValuePtr(tv)
		// 非结构体就为单字段查询
		if dest.Kind() != reflect.Struct {
			return t.QueryRowScan(res)
		}

		// 结构体
		rows, err := t.Query()
		if err != nil {
			return err
		}
		defer rows.Close()
		
		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		column2Index := t.parseCol2FiledIndex(tv.Type())
		destStruct := reflect.New(dest.Type()).Elem()
		values := make([]interface{}, len(columns))
		for _, column := range columns {
			values = append(values, destStruct.Field(column2Index[column]).Addr().Interface())
		}
		t.QueryRowScan(values...)
		tv.Set(destStruct)
	case reflect.Slice:
	default:
		return errors.New("res it should ptr/slice")
	}
	// sqlStr := t.tmpSqlObj.GetSqlStr()
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

// Raw 执行原生sqlStr
func (t *Table) Raw(sqlStr string) *Table {
	t.tmpSqlObj = NewCacheSql(sqlStr)
	return t
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	return t.db.Exec(t.tmpSqlObj.GetSqlStr())
}

// QueryRowScan 单行查询
func (t *Table) QueryRowScan(dest ...interface{}) error {
	return t.db.QueryRow(t.tmpSqlObj.GetSqlStr()).Scan(dest...)
}

// Query 多行查询
func (t *Table) Query() (*sql.Rows, error) {
	return t.db.Query(t.tmpSqlObj.GetSqlStr())
}

// removeValuePtr 移除多指针
func removeValuePtr(t reflect.Value) reflect.Value {
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
