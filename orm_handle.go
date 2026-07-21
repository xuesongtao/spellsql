package spellsql

import (
	"database/sql"
	"errors"
	"reflect"
	"sort"
	"time"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
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
	if _, err := t.insert(internal.INSERT, nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	return t
}

// InsertOfColumns 批量新增, 指定新增列
func (t *Table) InsertOfColumns(cols []string, insertObjs ...interface{}) *Table {
	if _, err := t.insert(internal.INSERT, cols, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	return t
}

func (t *Table) insert(opType internal.OpType, cols []string, insertObjs ...interface{}) ([]string, error) {
	if len(insertObjs) == 0 {
		return nil, errors.New("insertObjs is empty")
	}

	var (
		insertSql    *builder.Insert
		needCols     = t.getNeedCols(cols)
		handleCols   []string
		isOnlyInsert = len(insertObjs) == 1 // 仅仅只有一个
	)

	for i, insertObj := range insertObjs {
		if isOnlyInsert { // insert 一个值的时候, 在解析列的时候跳过零值
			needCols = nil
		}
		columns, values, err := t.getHandleTableCol2Val(insertObj, internal.INSERT, needCols)
		if err != nil {
			return nil, errors.New("getHandleTableCol2Val is failed, err:" + err.Error())
		}
		if i == 0 {
			insertSql = builder.NewInsert(t.dbType)
			switch opType {
			case internal.INSERT_REPLACE:
				insertSql.IntoReplace(t.name)
			case internal.INSERT_IGNORE:
				insertSql.IntoIgnore(t.name)
			default:
				insertSql.Into(t.name)
			}
			insertSql.Columns(columns...)
			handleCols = columns
		}
		insertSql.Values(values...)
	}
	t.builder = insertSql
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
	if _, err := t.insert(internal.INSERT, nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	t.builder.(*builder.Insert).DuplicateUpdate(keys)
	return t
}

// AppendSql 对 sql 进行自定义追加
func (t *Table) AppendSql(sqlStr string, args ...interface{}) *Table {
	t.builder.AppendSql2Args(sqlStr, args...)
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
	if _, err := t.insert(internal.INSERT_IGNORE, nil, insertObjs...); err != nil {
		sLog.Error(t.ctx, err)
		return nil
	}
	return t
}

// Delete 会以对象中有值得为条件进行删除
// 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Delete(deleteObj ...interface{}) *Table {
	if len(deleteObj) > 0 {
		columns, values, err := t.getHandleTableCol2Val(deleteObj[0], internal.DELETE, nil)
		if err != nil {
			sLog.Error(t.ctx, "getHandleTableCol2Val is failed, err:", err)
			return t
		}

		l := len(columns)
		bld := builder.NewDelete(t.dbType).From(t.name)
		bld.WhereCb(func(wb *builder.Where) {
			for i := 0; i < l; i++ {
				k := columns[i]
				v := values[i]
				wb.Eq(k, v)
			}
		})
		t.builder = bld
	} else {
		if utils.Null(t.name) {
			sLog.Error(t.ctx, internal.TableNameIsUnknownErr)
			return t
		}
		t.builder = builder.NewDelete(t.dbType).From(t.name)
	}
	return t
}

// DeleteWhere 根据条件删除
func (t *Table) DeleteWhere(where string, args ...interface{}) *Table {
	return t.Delete().Where(where, args...)
}

// Update 会更新输入的值
// 默认排除更新主键, 如果要排除其他可以调用 Exclude 方法自定义排除
func (t *Table) Update(updateObj interface{}, where string, args ...interface{}) *Table {
	columns, values, err := t.getHandleTableCol2Val(updateObj, internal.UPDATE, nil)
	if err != nil {
		sLog.Error(t.ctx, "getHandleTableCol2Val is failed, err:", err)
		return t
	}

	l := len(columns)
	updateBuilder := builder.NewUpdate(t.dbType).Table(t.name)
	for i := 0; i < l; i++ {
		k := columns[i]
		v := values[i]
		// t.tmpSqlObj.SetUpdateValueArgs("?v = ?", t.GetParcelFields(k), v)
		updateBuilder.Set(k, v)
	}
	updateBuilder.WhereCb(func(wb *builder.Where) {
		wb.And(where, args...)
	})
	t.builder = updateBuilder
	return t
}

// getHandleTableCol2Val 用于Insert/Delete/Update时, 解析结构体中对应列名和值
// 从对象中以 tag 做为 key, 值作为 value, 同时 key 会过滤掉不是表的字段名
func (t *Table) getHandleTableCol2Val(v interface{}, op uint8, needCols map[string]bool) (columns []string, values []interface{}, err error) {
	oldTv := reflect.ValueOf(v)
	tv := utils.RemoveValuePtr(oldTv)
	if tv.Kind() != reflect.Struct {
		err = errors.New("it must is struct")
		return
	}

	if err := t.initTableName(oldTv).initCacheCol2InfoMap(); err != nil {
		return nil, nil, err
	}

	ty := tv.Type()
	fieldNum := ty.NumField()
	columns = make([]string, 0, fieldNum)
	values = make([]interface{}, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		col, tag, needMarshal := t.parseStructField(ty.Field(i), sureMarshal)
		if utils.Null(col) {
			continue
		}

		// 判断下数据库字段是否存在
		tableField, ok := t.cacheCol2InfoMap[col]
		if !ok {
			// sLog.Error(t.ctx, "cacheCol2InfoMap is not found col:", col)
			continue
		}

		// 空值处理
		val := tv.Field(i)
		isZero := val.IsZero()
		if tableField.IsPri() { // 主键, 防止更新
			if (internal.Equal(op, internal.INSERT) && isZero) ||
				(internal.Equal(op, internal.DELETE) && isZero) ||
				internal.Equal(op, internal.UPDATE) {
				continue
			}
		}

		if isZero {
			if op == internal.INSERT || op == internal.UPDATE {
				// 判断下是否有设置了默认值
				if tmp, ok := t.waitHandleStructFieldMap[tag]; ok && tmp.defaultVal != nil { // orm 中设置了默认值
					columns = append(columns, col)
					values = append(values, tmp.defaultVal)
					continue
				} else if needCols != nil { // 需要的列, 但是没有设置默认值, 则使用数据库默认值
					columns = append(columns, col)
					values = append(values, dialect.GetTableMeter(t.dbType).GetDefaultVal(col, tableField))
					continue
				}
				// if tableField.NotNull() && !tableField.Default.Valid && !ok { // db 中没有设置默认值
				// 	return nil, nil, fmt.Errorf("field %q should't null, you can first call TagDefault", col)
				// }
			}

			if needCols == nil { // 单行
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
		err = internal.StructTagErr
		return
	}
	return
}

// ParseCol2Val 根据对象解析表的 col 和 val
func (t *Table) ParseCol2Val(src interface{}, op ...uint8) ([]string, []interface{}, error) {
	defaultOp := internal.INSERT
	if len(op) > 0 {
		defaultOp = op[0]
	}
	columns, values, err := t.getHandleTableCol2Val(src, defaultOp, nil)
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
	infos := make([]*dialect.TableColInfo, 0, len(t.cacheCol2InfoMap))
	for _, col := range t.cacheCol2InfoMap {
		if skipMap[col.Field] {
			continue
		}
		infos = append(infos, col)
	}
	sort.Sort(dialect.SortByTableColInfo(infos))
	l := len(infos)
	cols := make([]string, l)
	for i := 0; i < l; i++ {
		cols[i] = infos[i].Field
	}
	return cols
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	if err := t.prevCheck(); err != nil {
		return nil, err
	}
	st := time.Now()
	sqlStr, args := t.builder.GetSql2Args()
	res, err := t.db.ExecContext(t.ctx, sqlStr, args...)
	if err != nil {
		return res, errors.New("err:" + err.Error() + "; sqlStr:" + t.builder.GetSqlStr())
	}
	printCostTimeLog(t.ctx, st, t.builder.GetSqlStr(), t.isPrintSql)
	return res, nil
}
