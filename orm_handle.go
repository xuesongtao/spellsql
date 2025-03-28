package spellsql

import (
	"database/sql"
	"errors"
	"reflect"
	"sort"
	"strings"
)

// Slice2Interfaces 切片转 interfaces
func Slice2Interfaces(l int, to func(i int) interface{}) []interface{} {
	res := make([]interface{}, l)
	for i := 0; i < l; i++ {
		res[i] = to(i)
	}
	return res
}

// Insert 提交, 支持批量提交
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Insert(insertObjs ...interface{}) *Table {
	// 默认插入全量字段
	if _, err := t.insert(nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	return t
}

// InsertOfField 批量新增, 指定新增列
func (t *Table) InsertOfFields(cols []string, insertObjs ...interface{}) *Table {
	if _, err := t.insert(cols, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	return t
}

func (t *Table) insert(cols []string, insertObjs ...interface{}) ([]string, error) {
	if len(insertObjs) == 0 {
		return nil, errors.New("insertObjs is empty")
	}

	var (
		insertSql  *SqlStrObj
		needCols   = t.getNeedCols(cols)
		handleCols []string
	)
	isOnlyInsert := len(insertObjs) == 1 // 仅仅只有一个
	for i, insertObj := range insertObjs {
		if isOnlyInsert { // insert 一个值的时候, 在解析列的时候跳过零值
			needCols = nil
		}
		columns, values, err := t.getHandleTableCol2Val(insertObj, INSERT, needCols, t.name)
		if err != nil {
			return nil, errors.New("getHandleTableCol2Val is failed, err:" + err.Error())
		}
		if i == 0 {
			insertSql = t.getSqlObj("INSERT INTO ?v (?v) VALUES", t.name, t.GetParcelFields(columns...))
			handleCols = columns
		}
		insertSql.SetInsertValues(values...)
	}
	t.tmpSqlObj = insertSql
	return handleCols, nil
}

// getNeedCols 获取需要 cols
func (t *Table) getNeedCols(cols []string) map[string]bool {
	if len(cols) == 0 {
		cols = t.GetCols() // 获取全量字段
	}

	res := make(map[string]bool, len(cols))
	for _, col := range cols {
		res[col] = true
	}
	return res
}

// InsertODKU insert 主键冲突更新
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertODKU(insertObj interface{}, keys ...string) *Table {
	return t.InsertsODKU([]interface{}{insertObj}, keys...)
}

// InsertsODKU insert 主键冲突更新批量
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertsODKU(insertObjs []interface{}, keys ...string) *Table {
	if _, err := t.insert(nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	kv := make([]string, 0)
	keys = t.GetParcelFieldArr(keys...)
	for _, key := range keys {
		kv = append(kv, key+"=VALUES("+key+")")
	}

	if len(kv) > 0 {
		t.AppendSql("ON DUPLICATE KEY UPDATE " + strings.Join(kv, ", "))
	}
	return t
}

// AppendSql 对 sql 进行自定义追加
func (t *Table) AppendSql(sqlStr string, args ...interface{}) *Table {
	if sqlStr != "" {
		t.tmpSqlObj.Append(sqlStr, args...)
	}
	return t
}

// InsertIg insert ignore into xxx  新增忽略
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertIg(insertObj interface{}) *Table {
	return t.InsertsIg(insertObj)
}

// InsertsIg insert ignore into xxx  新增批量忽略
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) InsertsIg(insertObjs ...interface{}) *Table {
	if _, err := t.insert(nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	insertSqlStr := strings.Replace(t.tmpSqlObj.FmtSql(), "INSERT INTO", "INSERT IGNORE INTO", 1)
	t.tmpSqlObj = t.getSqlObj(insertSqlStr)
	return t
}

// Delete 会以对象中有值得为条件进行删除
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		columns, values, err := t.getHandleTableCol2Val(deleteObj[0], DELETE, nil, t.name)
		if err != nil {
			sLog.Error(t.ctx, "getHandleTableCol2Val is failed, err:", err)
			return t
		}

		l := len(columns)
		t.tmpSqlObj = t.getSqlObj("DELETE FROM ?v WHERE", t.name)
		for i := 0; i < l; i++ {
			k := columns[i]
			v := values[i]
			t.tmpSqlObj.SetWhereArgs("?v = ?", t.GetParcelFields(k), v)
		}
	} else {
		if null(t.name) {
			sLog.Error(t.ctx, tableNameIsUnknownErr)
			return t
		}
		t.tmpSqlObj = t.getSqlObj("DELETE FROM ?v WHERE", t.name)
	}
	return t
}

// Update 会更新输入的值
// 默认排除更新主键, 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Update(updateObj interface{}, where string, args ...interface{}) *Table {
	columns, values, err := t.getHandleTableCol2Val(updateObj, UPDATE, nil, t.name)
	if err != nil {
		sLog.Error(t.ctx, "getHandleTableCol2Val is failed, err:", err)
		return t
	}

	l := len(columns)
	t.tmpSqlObj = t.getSqlObj("UPDATE ?v SET", t.name)
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		t.tmpSqlObj.SetUpdateValueArgs("?v = ?", t.GetParcelFields(k), v)
	}
	t.tmpSqlObj.SetWhereArgs(where, args...)
	return t
}

// getHandleTableCol2Val 用于Insert/Delete/Update时, 解析结构体中对应列名和值
// 从对象中以 tag 做为 key, 值作为 value, 同时 key 会过滤掉不是表的字段名
func (t *Table) getHandleTableCol2Val(v interface{}, op uint8, needCols map[string]bool, tableName ...string) (columns []string, values []interface{}, err error) {
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
			if op == INSERT || op == UPDATE {
				// 判断下是否有设置了默认值
				tmp, ok := t.waitHandleStructFieldMap[tag]
				if ok && tmp.defaultVal != nil { // orm 中设置了默认值
					columns = append(columns, col)
					values = append(values, tmp.defaultVal)
					continue
				}
				// if tableField.NotNull() && !tableField.Default.Valid && !ok { // db 中没有设置默认值
				// 	return nil, nil, fmt.Errorf("field %q should't null, you can first call TagDefault", col)
				// }

			}

			if needCols == nil {
				continue
			}
		}

		// 需要的跳过
		if needCols != nil && !needCols[col] {
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

// ParseCol2Val 根据对象解析表的 col 和 val
func (t *Table) ParseCol2Val(src interface{}, op ...uint8) ([]string, []interface{}, error) {
	defaultOp := INSERT
	if len(op) > 0 {
		defaultOp = op[0]
	}
	columns, values, err := t.getHandleTableCol2Val(src, defaultOp, nil, t.name)
	if err != nil {
		return nil, nil, err
	}
	return columns, values, nil
}

// GetCols 获取所有列
func (t *Table) GetCols(skipCols ...string) []string {
	var skipMap map[string]bool
	if len(skipCols) > 0 {
		skipMap = make(map[string]bool, len(skipCols))
		for _, v := range skipCols {
			skipMap[v] = true
		}
	}
	if err := t.initCacheCol2InfoMap(); err != nil {
		sLog.Error(t.ctx, "t.initCacheCol2InfoMap is failed, err:", err)
		return nil
	}
	infos := make([]*TableColInfo, 0, len(t.cacheCol2InfoMap))
	for _, col := range t.cacheCol2InfoMap {
		if skipMap[col.Field] {
			continue
		}
		infos = append(infos, col)
	}
	sort.Sort(SortByTableColInfo(infos))
	l := len(infos)
	cols := make([]string, l)
	for i := 0; i < l; i++ {
		cols[i] = infos[i].Field
	}
	return cols
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	defer t.free()
	if err := t.prevCheck(); err != nil {
		return nil, err
	}
	sqlStr := t.tmpSqlObj.SetPrintLog(t.isPrintSql).SetCallerSkip(t.printSqlCallSkip).GetSqlStr()
	res, err := t.db.ExecContext(t.ctx, sqlStr)
	if err != nil {
		return res, errors.New("err:" + err.Error() + "; sqlStr:" + sqlStr)
	}
	return res, nil
}
