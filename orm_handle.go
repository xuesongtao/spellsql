package spellsql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Insert 提交, 支持批量提交
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Insert(insertObjs ...interface{}) *Table {
	if len(insertObjs) == 0 {
		sLog.Error("insertObjs is empty")
		return t
	}

	t.checkNull = true
	var insertSql *SqlStrObj
	for i, insertObj := range insertObjs {
		columns, values, err := t.getHandleTableCol2Val(insertObj, INSERT, t.name)
		if err != nil {
			sLog.Error("getHandleTableCol2Val is failed, err:", err)
			return t
		}
		if i == 0 {
			insertSql = NewCacheSql("INSERT INTO ?v (?v) VALUES", t.name, strings.Join(columns, ", "))
			insertSql.SetStrSymbol(t.getStrSymbol())
		}
		insertSql.SetInsertValues(values...)
	}
	t.tmpSqlObj = insertSql
	return t
}

// InsertODKU insert 主键冲突更新
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertODKU(insertObj interface{}, keys ...string) *Table {
	if insertObj == nil {
		sLog.Error("insertObj is nil")
		return t
	}

	t.checkNull = true
	columns, values, err := t.getHandleTableCol2Val(insertObj, INSERT, t.name)
	if err != nil {
		sLog.Error("getHandleTableCol2Val is failed, err:", err)
		return t
	}
	insertSql := NewCacheSql("INSERT INTO ?v (?v) VALUES", t.name, strings.Join(columns, ", "))
	insertSql.SetStrSymbol(t.getStrSymbol())
	insertSql.SetInsertValues(values...)
	kv := make([]string, 0, len(columns))
	if len(keys) == 0 {
		keys = columns
	}
	for _, key := range keys {
		kv = append(kv, key+"=VALUES("+key+")")
	}
	insertSql.Append("ON DUPLICATE KEY UPDATE " + strings.Join(kv, ", "))
	t.tmpSqlObj = insertSql
	return t
}

// InsertIg insert ignore into xxx  新增忽略
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertIg(insertObj interface{}) *Table {
	if insertObj == nil {
		sLog.Error("insertObj is nil")
		return t
	}

	t.checkNull = true
	columns, values, err := t.getHandleTableCol2Val(insertObj, INSERT, t.name)
	if err != nil {
		sLog.Error("getHandleTableCol2Val is failed, err:", err)
		return t
	}
	insertSql := NewCacheSql("INSERT IGNORE INTO ?v (?v) VALUES", t.name, strings.Join(columns, ", "))
	insertSql.SetStrSymbol(t.getStrSymbol())
	insertSql.SetInsertValues(values...)
	t.tmpSqlObj = insertSql
	return t
}

// Delete 会以对象中有值得为条件进行删除
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		columns, values, err := t.getHandleTableCol2Val(deleteObj[0], DELETE, t.name)
		if err != nil {
			sLog.Error("getHandleTableCol2Val is failed, err:", err)
			return t
		}

		l := len(columns)
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name).SetStrSymbol(t.getStrSymbol())
		for i := 0; i < l; i++ {
			k := columns[i]
			v := values[i]
			t.tmpSqlObj.SetWhereArgs("?v = ?", k, v)
		}
	} else {
		if null(t.name) {
			sLog.Error(tableNameIsUnknownErr)
			return t
		}
		t.tmpSqlObj = NewCacheSql("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
// 默认排除更新主键, 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Update(updateObj interface{}, where string, args ...interface{}) *Table {
	columns, values, err := t.getHandleTableCol2Val(updateObj, UPDATE, t.name)
	if err != nil {
		sLog.Error("getHandleTableCol2Val is failed, err:", err)
		return t
	}

	l := len(columns)
	t.tmpSqlObj = NewCacheSql("UPDATE ?v SET", t.name).SetStrSymbol(t.getStrSymbol())
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		t.tmpSqlObj.SetUpdateValueArgs("?v = ?", k, v)
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)
	return t
}

// getHandleTableCol2Val 用于Insert/Delete/Update时, 解析结构体中对应列名和值
// 从对象中以 tag 做为 key, 值作为 value, 同时 key 会过滤掉不是表的字段名
func (t *Table) getHandleTableCol2Val(v interface{}, op uint8, tableName ...string) (columns []string, values []interface{}, err error) {
	tv := removeValuePtr(reflect.ValueOf(v))
	if tv.Kind() != reflect.Struct {
		err = errors.New("it must is struct")
		return
	}

	ty := tv.Type()
	if null(t.name) {
		t.name = parseTableName(ty.Name())
	}

	if err := t.initCacheCol2InfoMap(); err != nil {
		return nil, nil, err
	}

	fieldNum := ty.NumField()
	columns = make([]string, 0, fieldNum)
	values = make([]interface{}, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		col, tag, needMarshal := t.parseStructField(ty.Field(i), sureMarshal)
		if null(col) {
			continue
		}

		// 判断下数据库字段是否存在
		tableField, ok := t.cacheCol2InfoMap[col]
		if !ok {
			continue
		}

		// 空值处理
		val := tv.Field(i)
		isZero := val.IsZero()
		if tableField.IsPri() { // 主键, 防止更新
			if (equal(op, INSERT) && isZero) ||
				(equal(op, DELETE) && isZero) ||
				equal(op, UPDATE) {
				continue
			}
		}

		if isZero {
			if t.checkNull { // 检查下 null
				tmp, ok := t.waitHandleStructFieldMap[tag]
				if ok && tmp.defaultVal != nil && tableField.NotNull() { // orm 中设置了默认值
					columns = append(columns, col)
					values = append(values, tmp.defaultVal)
				} else if tableField.NotNull() && !tableField.Default.Valid { // db 中设置了默认值
					return nil, nil, fmt.Errorf("field %q should't null, you can first call TagDefault", col)
				}
			}
			continue
		}

		columns = append(columns, col)
		if needMarshal {
			dataBytes, err := t.waitHandleStructFieldMap[tag].marshal(val.Interface())
			if err != nil {
				return nil, nil, err
			}
			values = append(values, dataBytes)
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
