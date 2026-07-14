package spellsql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

// Select 查询内容
// fields:
// 1. 可以多个通过逗号隔开
// 2. 也可以直接添加
func (t *Table) Select(fields ...string) *Table {
	if len(fields) == 0 {
		sLog.Error(t.ctx, "fields is null")
		return t
	}
	if len(fields) == 1 { // 如果只有一个字段, 可能有多个字段拼接的字符串, 需要解析
		return t.setSelect(t.parseCols(fields[0])...)
	} else {
		return t.setSelect(fields...)
	}
}

func (t *Table) parseCols(fields string) []string {
	if utils.Null(fields) {
		return nil
	}
	arr := make([]string, 0, 5)
	for _, col := range strings.Split(fields, ",") {
		arr = append(arr, strings.TrimSpace(col))
	}
	return arr
}

func (t *Table) setSelect(col ...string) *Table {
	if len(col) == 0 {
		sLog.Error(t.ctx, "fields is null")
		return t
	}

	if !utils.Null(t.name) {
		t.builder = builder.NewSelect(t.dbType).Select(col...).From(t.name)
	} else {
		t.handleCols = col
	}
	return t
}

// SelectAuto 根据输入类型进行自动推断要查询的字段值
// src 如下:
//  1. 为 string 的话会被直接解析成查询字段
//  2. 为 struct/struct slice 会按 struct 进行解析, 查询字段为 struct 的 tag, 同时会过滤掉非当前表字段名
//  3. 其他情况会被解析为查询所有
//
// tableName 在 NewTable 时设置过了, 就不需要设置, 如果设置了优先级最高
// 如果实现了 TableName 方法, 使用该方法返回的表名,
// 如果没有实现该方法, 会使用 struct 的类型名进行解析, 解析规则为: 驼峰转下划线
func (t *Table) SelectAuto(src interface{}, tableName ...string) *Table {
	if len(tableName) > 0 {
		t.name = tableName[0]
	}

	if val, ok := src.(string); ok {
		return t.Select(val)
	}

	ty := utils.RemoveTypePtr(reflect.TypeOf(src))
	selectFields := make([]string, 0, 5)
	switch kind := ty.Kind(); kind {
	case reflect.Struct, reflect.Slice:
		if ty.Kind() == reflect.Slice {
			ty = ty.Elem()
			if ty.Kind() == reflect.Ptr {
				ty = utils.RemoveTypePtr(ty)
			}
		}

		if err := t.initTableName(reflect.ValueOf(src), tableName...).initCacheCol2InfoMap(); err != nil {
			sLog.Error(t.ctx, "initCacheCol2InfoMap is failed, err:", err)
			return t
		}

		_, sortCol := t.parseCol2StructField(ty, true)
		for _, col := range sortCol {
			// 排除结构体中的字段, 数据库没有
			if _, ok := t.cacheCol2InfoMap[col]; !ok {
				continue
			}
			selectFields = append(selectFields, col)
		}

		if len(selectFields) == 0 {
			sLog.Error(t.ctx, "parse col is failed, you need to confirm whether to add correct tag(defaultTag: json)")
		}
		t.setSelect(selectFields...)
	default:
		if utils.IsOneField(kind) { // 因为单字段不能解析查内容, 所以直接返回, 在最终调用处报错
			return t
		}
		sLog.Warning(t.ctx, "src kind is not struct or slice struct")
		t.SelectAll()
	}
	return t
}

// SelectAll 查询所有字段
func (t *Table) SelectAll() *Table {
	return t.setSelect("*")
}

// SelectCount 查询总数
func (t *Table) SelectCount() *Table {
	return t.setSelect("COUNT(*)")
}

// From 设置表名
func (t *Table) From(tableName string) *Table {
	t.name = tableName
	if len(t.handleCols) > 0 {
		t.setSelect(t.handleCols...)
	} else {
		t.SelectAll()
	}
	return t
}

func (t *Table) getSelectBuilder() *builder.Select {
	return t.builder.(*builder.Select)
}

// Join 连表查询
// 说明: 连表查询时, 如果两个表有相同字段名查询结果会出现错误
// 解决方法: 1. 推荐使用别名来区分; 2. 使用 Query 对结果我们自己进行处理
func (t *Table) Join(joinTable, on string, joinType ...uint8) *Table {
	t.getSelectBuilder().Join(joinTable, on, joinType...)
	return t
}

// LefJoin 连表查询
// 说明: 连表查询时, 如果两个表有相同字段名查询结果会出现错误
// 解决方法: 1. 推荐使用别名来区分; 2. 使用 Query 对结果我们自己进行处理
// Deprecated: 使用 LeftJoin 替代, 单词拼写错误
func (t *Table) LefJoin(joinTable, on string) *Table {
	return t.LeftJoin(joinTable, on)
}

// LeftJoin 连表查询,
// 说明: 连表查询时, 如果两个表有相同字段名查询结果会出现错误
// 解决方法: 1. 推荐使用别名来区分; 2. 使用 Query 对结果我们自己进行处理
func (t *Table) LeftJoin(joinTable, on string) *Table {
	t.getSelectBuilder().LeftJoin(joinTable, on)
	return t
}

// RightJoin 连表查询
// 说明: 连表查询时, 如果两个表有相同字段名查询结果会出现错误
// 解决方法: 1. 推荐使用别名来区分; 2. 使用 Query 对结果我们自己进行处理
func (t *Table) RightJoin(joinTable, on string) *Table {
	t.getSelectBuilder().RightJoin(joinTable, on)
	return t
}

// Where 支持占位符
// 如: Where("username = ? AND password = ?d", "test", "123")
// => xxx AND "username = "test" AND password = 123
func (t *Table) Where(sqlStr string, args ...interface{}) *Table {
	builder.WhereCb(t.builder, func(wb *builder.Where) {
		wb.And(sqlStr, args...)
	})
	return t
}

// WhereBuilder 使用 builder.Where 进行查询条件构建
func (t *Table) WhereBuilder(wb *builder.Where) *Table {
	builder.WhereCb(t.builder, func(innerWb *builder.Where) {
		sqlStr, args := wb.GetNoParseSql2Args()
		innerWb.And("("+sqlStr+")", args...)
	})
	return t
}

// OrWhereBuilder 使用 builder.Where 进行查询条件构建
func (t *Table) OrWhereBuilder(wb *builder.Where) *Table {
	builder.WhereCb(t.builder, func(innerWb *builder.Where) {
		sqlStr, args := wb.GetNoParseSql2Args()
		innerWb.Or("("+sqlStr+")", args...)
	})
	return t
}

// OrWhere 支持占位符
// 如: OrWhere("username = ? AND password = ?d", "test", "123")
// => xxx OR "username = "test" AND password = 123
func (t *Table) OrWhere(sqlStr string, args ...interface{}) *Table {
	builder.WhereCb(t.builder, func(wb *builder.Where) {
		wb.Or(sqlStr, args...)
	})
	return t
}

// WhereLike like 查询
// likeType ALK-全模糊 RLK-右模糊 LLK-左模糊
func (t *Table) WhereLike(likeType uint8, filedName, value string) *Table {
	switch likeType {
	case ALK:
		builder.WhereCb(t.builder, func(wb *builder.Where) {
			wb.Like(filedName, value)
		})
	case RLK:
		builder.WhereCb(t.builder, func(wb *builder.Where) {
			wb.LikeRight(filedName, value)
		})
	case LLK:
		builder.WhereCb(t.builder, func(wb *builder.Where) {
			wb.LikeLeft(filedName, value)
		})
	}
	return t
}

// AllLike 全模糊查询
func (t *Table) AllLike(filedName, value string) *Table {
	return t.WhereLike(ALK, filedName, value)
}

// LeftLike 左模糊
func (t *Table) LeftLike(filedName, value string) *Table {
	return t.WhereLike(LLK, filedName, value)
}

// RightLike 右模糊
func (t *Table) RightLike(filedName, value string) *Table {
	return t.WhereLike(RLK, filedName, value)
}

// Between
func (t *Table) Between(filedName string, leftVal, rightVal interface{}) *Table {
	builder.WhereCb(t.builder, func(wb *builder.Where) {
		wb.Between(filedName, leftVal, rightVal)
	})
	return t
}

// OrderBy
func (t *Table) OrderBy(sqlStr string) *Table {
	t.getSelectBuilder().OrderBy(sqlStr)
	return t
}

func (t *Table) OrderByAsc(fieldName string) *Table {
	t.getSelectBuilder().OrderByAsc(fieldName)
	return t
}

func (t *Table) OrderByDesc(fieldName string) *Table {
	t.getSelectBuilder().OrderByDesc(fieldName)
	return t
}

// Limit 分页
// 会对 page, size 进行校验处理
// 注: page, size 只支持 int 系列类型
func (t *Table) Limit(page, size interface{}) *Table {
	t.getSelectBuilder().Limit(page, size)
	return t
}

// GroupBy
func (t *Table) GroupBy(sqlStr string) *Table {
	t.getSelectBuilder().GroupBy(sqlStr)
	return t
}

// Having
func (t *Table) Having(sqlStr string, args ...interface{}) *Table {
	t.getSelectBuilder().Having(sqlStr, args...)
	return t
}

// Count 获取总数
func (t *Table) Count(total interface{}) error {
	if err := t.prevCheck(); err != nil {
		return err
	}

	// 这里不要释放, 如果是列表查询的话, 还会再进行查询内容操作
	// defer t.free()
	// st := time.Now()
	sqlStr, args := t.getSelectBuilder().GetTotalSql2Args()
	err := t.db.QueryRowContext(t.ctx, sqlStr, args...).Scan(total)
	if err != nil {
		return err
	}
	// defer printCostTimeLog(t.ctx, st, t.tmpSqlObj.getSqlLogStr("sql", sqlStr), t.isPrintSql)
	return nil
}

// FindOne 单行查询
// 注: 如果为空的话, 会返回 nullRowErr
// dest 长度 > 1 时, 支持多个字段查询
// dest 长度 == 1 时, 支持 struct/单字段/map
func (t *Table) FindOne(dest ...interface{}) error {
	if err := t.prevCheck(); err != nil {
		return err
	}

	t.getSelectBuilder().Limit(0, 1)

	if len(dest) == 1 {
		ty, err := t.getDestReflectType(dest[0], []reflect.Kind{reflect.Struct, reflect.Map}, internal.FindOneDestTypeErr)
		if err != nil && !utils.IsOneField(ty.Kind()) { // 需要排除单字段查询
			return err
		}
		return t.find(dest[0], ty, false)
	}
	t.printSqlCallSkip += 1
	return t.QueryRowScan(dest...)
}

// FindOneFn 单行查询
// 注: 如果为空的话, 会返回 nullRowErr
// dest 支持 struct/单字段/map
// fn 支持将查询结果行进行修改, 需要修改的时候 fn 回调的 _row 需要类型断言为[指针]对象才能处理
func (t *Table) FindOneFn(dest interface{}, fn ...SelectCallBackFn) error {
	if err := t.prevCheck(); err != nil {
		return err
	}

	t.getSelectBuilder().Limit(0, 1)

	ty, err := t.getDestReflectType(dest, []reflect.Kind{reflect.Struct, reflect.Map}, internal.FindOneDestTypeErr)
	if err != nil && !utils.IsOneField(ty.Kind()) { // 需要排除单字段查询
		return err
	}
	return t.find(dest, ty, false, fn...)
}

// FindOneIgnoreResult 查询结果支持多个, 此使用场景为需要使用 SelectCallBackFn 对每行进行处理
// 注: 因为查询的结果集为多个, dest 不为切片, 所有这个结果是不准确的
// dest 支持 struct/map
// fn 支持将查询结果行进行修改, 需要修改的时候 fn 回调的 _row 需要类型断言为[指针]对象才能处理
func (t *Table) FindOneIgnoreResult(dest interface{}, fn ...SelectCallBackFn) error {
	if err := t.prevCheck(); err != nil {
		return err
	}

	ty, err := t.getDestReflectType(dest, []reflect.Kind{reflect.Struct, reflect.Map}, internal.FindOneDestTypeErr)
	if err != nil && !utils.IsOneField(ty.Kind()) { // 需要排除单字段查询
		return err
	}
	return t.find(dest, ty, true, fn...)
}

// FindAll 多行查询
// dest 支持(struct/单字段/map)切片
// fn 支持将查询结果行进行处理, 需要处理每行内容时, fn 回调的 _row 需要类型断言为[切片中的类型]
func (t *Table) FindAll(dest interface{}, fn ...SelectCallBackFn) error {
	if err := t.prevCheck(); err != nil {
		return err
	}

	ty, err := t.getDestReflectType(dest, []reflect.Kind{reflect.Slice}, internal.FindAllDestTypeErr)
	if err != nil {
		return err
	}
	return t.find(dest, ty, false, fn...)
}

// FindWhere 如果没有添加查询字段内容, 会根据输入对象进行解析查询
// 注: 如果为单行查询的话, 当为空的话, 会返回 nullRowErr
// 如果没有指定查询条数, 默认 internal.DefaultBatchSelectSize
// dest 支持 struct/slice/单字段/map
func (t *Table) FindWhere(dest interface{}, where string, args ...interface{}) error {
	if t.builder == nil || t.getSelectBuilder().ColsEmpty() {
		t.SelectAuto(dest)
	}

	if err := t.prevCheck(); err != nil {
		return err
	}

	t.getSelectBuilder().WhereCb(func(wb *builder.Where) {
		wb.And(where, args...)
	})

	ty, err := t.getDestReflectType(dest, nil, nil)
	if err != nil {
		return err
	}
	return t.find(dest, ty, false)
}

// QueryRowScan 单行多值查询
func (t *Table) QueryRowScan(dest ...interface{}) error {
	if err := t.prevCheck(); err != nil {
		return err
	}
	t.printSqlCallSkip += 1

	rows, err := t.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() { // 没有就为空
		return internal.NullRowErr
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	colLen := len(colTypes)
	if colLen != len(dest) {
		return fmt.Errorf("select res len %d, dest len %d", colLen, len(dest))
	}
	values := make([]interface{}, colLen)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	// 将 dest 转为 []dest
	destReflectValues := reflect.ValueOf(append([]interface{}{}, dest...))
	t.loadDestType(destReflectValues.Type())
	if err := t.getScanValues(destReflectValues, nil, fieldIndex2NullIndexMap, colTypes, values); err != nil {
		return err
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	if err := t.setNullDest(destReflectValues, nil, fieldIndex2NullIndexMap, colTypes, values); err != nil {
		return err
	}
	return nil
}

// Query 多行查询
// 注: 返回的 sql.Rows 需要调用 Close, 防止 goroutine 泄露
func (t *Table) Query() (*sql.Rows, error) {
	if err := t.prevCheck(); err != nil {
		return nil, err
	}
	_ = t.initCacheCol2InfoMap() // 为 getScanValues 解析 NULL 值做准备, 由于调用 Raw 时, 可能会出现没有表名, 所有需要忽略错误
	st := time.Now()
	sqlStr, args := t.builder.GetSql2Args()
	rows, err := t.db.QueryContext(t.ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer printCostTimeLog(t.ctx, st, t.builder.GetSqlStr(), t.isPrintSql)
	return rows, nil
}

// getDestReflectType 解析 dest kind
func (t *Table) getDestReflectType(dest interface{}, shouldInKinds []reflect.Kind, outErr error) (ty reflect.Type, err error) {
	ty = reflect.TypeOf(dest)
	if ty.Kind() != reflect.Ptr {
		err = errors.New("dest should is ptr")
		return
	}

	ty = utils.RemoveTypePtr(ty)
	isIn := false
	for _, kind := range shouldInKinds {
		if ty.Kind() == kind {
			isIn = true
			break
		}
	}

	if !isIn {
		err = outErr
		return
	}
	return
}

// loadDestType 记录 dest 的类型, 因为对应的操作不会同时调用所有不存在数据竞争
func (t *Table) loadDestType(dest reflect.Type) {
	t.destTypeFlag = 0

	switch kind := dest.Kind(); kind {
	case reflect.Struct:
		t.destTypeFlag = structFlag
	case reflect.Slice:
		t.destTypeFlag = sliceFlag
	case reflect.Map:
		t.destTypeFlag = mapFlag
	default:
		if utils.IsOneField(kind) {
			t.destTypeFlag = oneFieldFlag
		}
	}
}

// find 查询处理入口, 根据 dest 类型进行分配处理
func (t *Table) find(dest interface{}, ty reflect.Type, ignoreRes bool, fn ...SelectCallBackFn) error {
	t.printSqlCallSkip += 2

	rows, err := t.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	t.loadDestType(ty)
	if t.isDestType(structFlag) || t.isDestType(mapFlag) || t.isDestType(oneFieldFlag) {
		return t.scanOne(rows, ty, dest, ignoreRes, fn...)
	} else if t.isDestType(sliceFlag) {
		return t.scanAll(rows, ty.Elem(), dest, fn...)
	} else {
		return errors.New("dest kind nonsupport")
	}
}

// scanAll 处理多个结果集
func (t *Table) scanAll(rows *sql.Rows, ty reflect.Type, dest interface{}, fn ...SelectCallBackFn) error {
	isPtr := ty.Kind() == reflect.Ptr
	if isPtr {
		ty = utils.RemoveTypePtr(ty) // 去指针
	}

	t.loadDestType(ty)
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	colLen := len(colTypes)
	col2StructFieldMap, _ := t.parseCol2StructField(ty, false)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	values := make([]interface{}, colLen)
	destReflectValue := utils.RemoveValuePtr(reflect.ValueOf(dest))
	if destReflectValue.IsNil() {
		destReflectValue.Set(reflect.MakeSlice(destReflectValue.Type(), 0, colLen))
	}
	for rows.Next() {
		base := reflect.New(ty).Elem()
		if err := t.getScanValues(base, col2StructFieldMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
			return err
		}

		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("rows scan is failed, err: %v", err)
		}

		if err := t.setNullDest(base, col2StructFieldMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
			return err
		}

		if len(fn) == 1 { // 回调方法
			if isPtr && !t.isDestType(mapFlag) { // 指针类型
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
func (t *Table) scanOne(rows *sql.Rows, ty reflect.Type, dest interface{}, ignoreRes bool, fn ...SelectCallBackFn) error {
	// t.loadDestType(ty) // 这里可以不用再处理
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	colLen := len(colTypes)
	col2StructFieldMap, _ := t.parseCol2StructField(ty, false)
	values := make([]interface{}, colLen)
	fieldIndex2NullIndexMap := make(map[int]int, colLen) // 用于记录 NULL 值到 struct 的映射关系
	destReflectValue := utils.RemoveValuePtr(reflect.ValueOf(dest))
	haveNoData := true
	for rows.Next() {
		haveNoData = false
		base := reflect.New(ty).Elem()
		if err := t.getScanValues(base, col2StructFieldMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
			return err
		}

		if err := rows.Scan(values...); err != nil {
			return err
		}

		if err := t.setNullDest(base, col2StructFieldMap, fieldIndex2NullIndexMap, colTypes, values); err != nil {
			return err
		}

		if len(fn) == 1 { // 回调方法, 方便修改
			if t.destTypeFlag == mapFlag {
				if err := fn[0](base.Interface()); err != nil {
					return err
				}
			} else {
				if err := fn[0](base.Addr().Interface()); err != nil {
					return err
				}
			}
		}

		if !ignoreRes { // 不忽略结果, 那只能出现在单行查询
			if destReflectValue.Kind() == reflect.Ptr {
				destReflectValue.Set(base.Addr())
			} else {
				destReflectValue.Set(base)
			}
			break
		}
	}

	if haveNoData && !ignoreRes {
		return internal.NullRowErr
	}
	return nil
}

// isDestType
func (t *Table) isDestType(typeNum uint8) bool {
	return internal.Equal(t.destTypeFlag, typeNum)
}

// getScanValues 获取待 Scan 的内容
func (t *Table) getScanValues(dest reflect.Value, col2StructFieldMap map[string]structField, fieldIndex2NullIndexMap map[int]int, colTypes []*sql.ColumnType, values []interface{}) error {
	var structMissFields []string
	for i, colType := range colTypes {
		var (
			fieldIndex       int
			tagName          string
			colName          = colType.Name()
			structFieldExist = true
		)
		if t.isDestType(structFlag) && col2StructFieldMap != nil {
			var tmp structField
			tmp, structFieldExist = col2StructFieldMap[colName]
			fieldIndex = tmp.offsetIndex
			tagName = tmp.tagName
		}

		// 说明结构里查询的值不存在
		if !structFieldExist {
			if structMissFields == nil {
				structMissFields = make([]string, 0, len(colTypes)/2)
			}
			structMissFields = append(structMissFields, colName)
			continue
		}

		// NULL 值处理, 防止 sql 报错, 否则就直接 Scan 到输入的 dest addr
		mayIsNull, _ := colType.Nullable() // 根据此获取的 NULL 值不准确(不同的 drive 返回不同), 但如果为 true 的话就没有问题
		if !mayIsNull {                    // 防止误判, 再判断下
			colInfo := t.cacheCol2InfoMap[colName]

			// 当 colInfo == nil 就直接通过 NULL 值处理, 如以下情况:
			// 1. 说明初始化表失败(只要 tableName 存在就不会为空), 查询的时候只会在 Query 里初始化
			// 2. sql 语句中使用了字段别名与表元信息字段名不一致
			mayIsNull = colInfo == nil || !colInfo.NotNull()
			// fmt.Printf("mayIsNull: %v colInfo: %+v\n", mayIsNull, colInfo)
		}
		// fmt.Println(colName, colType.ScanType().Name())
		if mayIsNull {
			switch colType.ScanType() {
			case nullInt64Type:
				values[i] = cacheNullInt64.Get().(*sql.NullInt64)
			case nullFloat64Type:
				values[i] = new(sql.NullFloat64)
			default:
				values[i] = cacheNullString.Get().(*sql.NullString)
			}

			// struct, 这里记录 struct 那个字段需要映射 NULL 值
			// map/单字段, 为了减少创建标记, 借助 fieldIndex2NullIndexMap 用于标识单字段是否包含空值,  在 setNullDest 使用
			if t.isDestType(structFlag) && structFieldExist {
				fieldIndex2NullIndexMap[fieldIndex] = i
			} else if t.isDestType(mapFlag) || t.isDestType(sliceFlag) || t.isDestType(oneFieldFlag) {
				fieldIndex2NullIndexMap[i] = i
			}
			continue
		}

		// 处理数据库字段非 NULL 部分
		if t.isDestType(structFlag) { // 结构体
			// 在非 NULL 的时候, 也判断下是否需要反序列化
			if handleStructField, ok := t.waitHandleStructFieldMap[tagName]; ok && handleStructField.unmarshal != nil {
				values[i] = cacheNullString.Get().(*sql.NullString)
				fieldIndex2NullIndexMap[fieldIndex] = i
				continue
			}
			values[i] = dest.Field(fieldIndex).Addr().Interface()
		} else if t.isDestType(mapFlag) {
			destValType := dest.Type().Elem()
			if destValType.Kind() == reflect.Interface {
				// 如果 map 的 value 为 interface{} 时, 数据库类型为字符串时 driver.Value 的类型为 RawBytes, 再经过 Scan 后, 会被处理为 []byte
				// 为了避免这种就直接处理为字符串
				values[i] = cacheNullString.Get().(*sql.NullString)
				fieldIndex2NullIndexMap[i] = i
			} else {
				values[i] = reflect.New(destValType).Interface()
			}
		} else if t.isDestType(oneFieldFlag) { // 单字段, 其自需占一个位置查询即可
			values[i] = dest.Addr().Interface()
			break
		} else if t.isDestType(sliceFlag) { // 单行, 多字段查询时
			values[i] = dest.Index(i).Interface()
		}
	}

	if t.isDestType(structFlag) && len(structMissFields) > 0 {
		return fmt.Errorf("getScanValues is failed, cols %q is miss dest struct", strings.Join(structMissFields, ","))
	}
	return nil
}

// nullScan 空值scan
func (t *Table) nullScan(dest, src interface{}, needUnmarshalField ...string) (err error) {
	switch val := src.(type) {
	case *sql.NullString:
		if len(needUnmarshalField) > 0 { // 判断下是否需要反序列化
			handleStructField, ok := t.waitHandleStructFieldMap[needUnmarshalField[0]]
			if ok && handleStructField.unmarshal != nil {
				if !utils.Null(val.String) {
					err = handleStructField.unmarshal([]byte(val.String), dest)
				}
				val.String = ""
				cacheNullString.Put(val)
				return
			}
		}
		err = internal.ConvertAssign(dest, val.String)
		val.String = ""
		cacheNullString.Put(val)
	case *sql.NullInt64:
		err = internal.ConvertAssign(dest, val.Int64)
		val.Int64 = 0
		cacheNullInt64.Put(val)
	case *sql.NullFloat64:
		err = internal.ConvertAssign(dest, val.Float64)
	default:
		err = errors.New("unknown null type")
	}
	return
}

// setNullDest 设置值
func (t *Table) setNullDest(dest reflect.Value, col2StructFieldMap map[string]structField, fieldIndex2NullIndexMap map[int]int, colTypes []*sql.ColumnType, scanResult []interface{}) error {
	if t.isDestType(structFlag) {
		for fieldIndex, nullIndex := range fieldIndex2NullIndexMap {
			col := colTypes[nullIndex].Name()
			tag := col2StructFieldMap[col].tagName
			destFieldValue := dest.Field(fieldIndex)
			if err := t.nullScan(destFieldValue.Addr().Interface(), scanResult[nullIndex], tag); err != nil {
				return err
			}
		}
	} else if t.isDestType(mapFlag) {
		destType := dest.Type()
		if destType.Key().Kind() != reflect.String {
			return errors.New("map key must is string")
		}
		if dest.IsNil() {
			dest.Set(reflect.MakeMapWithSize(destType, len(colTypes)))
		}
		for i, col := range colTypes {
			var (
				key = reflect.ValueOf(col.Name())
				val reflect.Value
			)
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
	} else if t.isDestType(oneFieldFlag) {
		if _, ok := fieldIndex2NullIndexMap[0]; ok {
			return t.nullScan(dest.Addr().Interface(), scanResult[0])
		}
	} else if t.isDestType(sliceFlag) { // QueryRowScan 方法会用, 单行多字段查询
		for _, nullIndex := range fieldIndex2NullIndexMap {
			if err := t.nullScan(dest.Index(nullIndex).Interface(), scanResult[nullIndex]); err != nil {
				return err
			}
		}
	}
	return nil
}
