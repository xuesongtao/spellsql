package spellsql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	defaultTableTag = "db"
	structErr       = "type User struct {\n" +
		"    Name string `db:\"name,omitempty\"`\n" +
		"    Age  int    `db:\"age,omitempty\"`\n" +
		"    Addr string `db:\"addr,omitempty\"`\n" +
		"}"
)

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

// parseTable 解析字段
func parseTable(v interface{}, filedTag string, tableName ...string) (table string, columns []string, values []interface{}, err error) {
	tv, err := getStructReflectValue(v)
	if err != nil {
		return
	}

	ty := tv.Type()
	if len(tableName) > 0 && tableName[0] != "" {
		table = tableName[0]
	} else {
		table = parseTableName(ty.Name())
	}
	filedNum := ty.NumField()
	columns = make([]string, 0, filedNum)
	values = make([]interface{}, 0, filedNum)
	for i := 0; i < filedNum; i++ {
		structField := ty.Field(i)
		if structField.Anonymous || !isExported(structField.Name) {
			continue
		}
		tag := structField.Tag.Get(filedTag)
		if tag == "" {
			continue
		}
		val := tv.Field(i)
		if val.IsZero() {
			continue
		}
		tmpIndex := IndexForBF(true, tag, ",")
		if tmpIndex > -1 {
			tag = tag[:tmpIndex]
		}
		columns = append(columns, tag)
		values = append(values, val.Interface())
	}

	if len(columns) == 0 || len(values) == 0 {
		err = fmt.Errorf("you should sure struct is ok, eg:%s", structErr)
		return
	}
	return
}
