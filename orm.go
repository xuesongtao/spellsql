package spellsql

import (
	"database/sql"
	"errors"
	"reflect"
	"sort"
	"strings"
)

// handleStructField 用于记录 struct 字段的处理方法
type handleStructField struct {
	needExclude bool        // 是否需要排除
	tagAlias    string      // 别名, 便于将数据库的字段映射到 struct
	marshal     marshalFn   // 序列化方法
	unmarshal   unmarshalFn // 反序列化方法
	defaultVal  interface{} // 默认值
}

// structField 结构体字段信息
type structField struct {
	offsetIndex int
	tagName     string
}

// Table 表的信息
type Table struct {
	db                       DBer
	tmer                     TableMetaer                   // 记录表初始化元信息对象
	printSqlCallSkip         uint8                         // 标记打印 sql 时, 需要跳过的 skip, 该参数为 runtime.Caller(skip)
	destTypeFlag             uint8                         // 查询时, 用于标记 dest 类型的
	isPrintSql               bool                          // 标记是否打印 sql
	haveFree                 bool                          // 标记 table 释放已释放
	needSetSize              bool                          // 标记批量查询的时候是否需要设置默认返回条数
	checkNull                bool                          // 在 Insert 时, db 字段为非 null 时检查
	tag                      string                        // 记录解析 struct 中字段名的 tag
	name                     string                        // 表名
	handleCols               string                        // Insert/Update/Delete/Select 操作的表字段名
	clonedSqlStr             string                        // 记录克隆前的 sqlStr
	tmpSqlObj                *SqlStrObj                    // 暂存 SqlStrObj 对象
	cacheCol2InfoMap         map[string]*TableColInfo      // 记录该表的所有字段名
	waitHandleStructFieldMap map[string]*handleStructField // 处理 struct 字段的方法, key: tag, value: 处理方法集
}

// NewTable 初始化, 通过 sync.Pool 缓存对象来提高性能
// 注: 使用 INSERT/UPDATE/DELETE/SELECT(SELECT 排除使用 Count)操作后该对象就会被释放, 如果继续使用会出现 panic
// args 支持两个参数
// args[0]: 会解析为 tableName, 这里如果有值, 在进行操作表的时候就会以此表为准,
// 如果为空时, 在通过对象进行操作时按驼峰规则进行解析表名, 解析规则如: UserInfo => user_info
// args[1]: 会解析为待解析的 tag, 默认 defaultTableTag
func NewTable(db DBer, args ...string) *Table {
	t := cacheTabObj.Get().(*Table)
	t.init()

	// 赋值
	t.db = db
	switch len(args) {
	case 1:
		t.name = args[0]
	case 2:
		t.name = args[0]
		t.tag = args[1]
	}
	return t
}

// init 初始化
func (t *Table) init() {
	t.printSqlCallSkip = 2
	t.isPrintSql = true
	t.haveFree = false
	t.needSetSize = false
	t.checkNull = false
	t.tag = defaultTableTag
}

// free 释放
func (t *Table) free() {
	// clone 了对象就不放回
	if !null(t.clonedSqlStr) {
		return
	}

	if t.haveFree {
		sLog.Error("table already free")
		return
	}

	t.haveFree = true // 标记释放

	// 释放内容
	t.db = nil
	t.tmer = nil
	t.name = ""
	t.handleCols = ""
	t.clonedSqlStr = ""
	t.tmpSqlObj = nil
	t.cacheCol2InfoMap = nil
	t.waitHandleStructFieldMap = nil

	// 存放缓存
	cacheTabObj.Put(t)
}

// Clone 克隆对象
func (t *Table) Clone() *Table {
	if null(t.clonedSqlStr) {
		t.clonedSqlStr = t.tmpSqlObj.FmtSql()
	}
	t.tmpSqlObj = NewCacheSql(t.clonedSqlStr)
	t.init()
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

	if err := t.prevCheck(false); err != nil {
		return err
	}

	if null(t.name) {
		return tableNameIsUnknownErr
	}

	// 防止 name 中包含 别名
	tableName := parseTableName(t.name)

	// 先判断下缓存中有没有
	if info, ok := cacheTableName2ColInfoMap.Load(tableName); ok {
		t.cacheCol2InfoMap, ok = info.(map[string]*TableColInfo)
		if ok {
			return nil
		}
	}

	if err := t.initTmer(); err != nil {
		return err
	}
	var err error
	t.cacheCol2InfoMap, err = t.tmer.GetField2ColInfoMap(t.db)
	if err != nil {
		return err
	}

	cacheTableName2ColInfoMap.Store(t.name, t.cacheCol2InfoMap)
	return nil
}

// initTmer 初始化表元数据对象
func (t *Table) initTmer() error {
	// 默认按 mysql 的方式处理
	if t.tmer == nil {
		t.tmer = defaultTmerObj
	}
	if null(t.name) {
		return tableNameIsUnknownErr
	}
	t.tmer.SetName(t.name)
	return nil
}

// getStrSymbol 获取适配器对应的字符串对应符号
func (t *Table) getStrSymbol() byte {
	_ = t.initTmer()
	return t.tmer.GetStrSymbol()
}

// setWaitHandleStructFieldMap 设置 waitHandleStructFieldMap 值
func (t *Table) setWaitHandleStructFieldMap(tag string, fn func(val *handleStructField)) {
	if t.waitHandleStructFieldMap == nil {
		t.waitHandleStructFieldMap = make(map[string]*handleStructField)
	}
	if _, ok := t.waitHandleStructFieldMap[tag]; !ok {
		t.waitHandleStructFieldMap[tag] = new(handleStructField)
	}
	fn(t.waitHandleStructFieldMap[tag])
}

// Exclude 对于 INSERT/UPDATE/DELETE/SELECT 操作中通过解析对象需要过滤的字段
// 注: 调用必须优先 Insert/Update/Delete/SelectAuto 操作的方法, 防止通过对象解析字段时失效
func (t *Table) Exclude(tags ...string) *Table {
	for _, tag := range tags {
		t.setWaitHandleStructFieldMap(tag, func(val *handleStructField) {
			val.needExclude = true
		})
	}
	return t
}

// TagAlias 设置 struct 字段别名, 默认是按字段的 tag 名
// 注: 调用必须优先 Insert/Update/Delete/SelectAuto 操作的方法, 防止通过对象解析字段时失效
// tag2AliasMap key: struct 的 tag 名, value: 表的列名
func (t *Table) TagAlias(tag2AliasMap map[string]string) *Table {
	for tag, alias := range tag2AliasMap {
		t.setWaitHandleStructFieldMap(tag, func(val *handleStructField) {
			val.tagAlias = alias
		})
	}
	return t
}

// TagDefault 设置 struct 字段默认值
// 注: 调用必须优先 Insert/Update/Delete/SelectAuto 操作的方法, 防止通过对象解析字段时失效
// tag2DefaultMap key: struct 的 tag 名, value: 字段默认值
func (t *Table) TagDefault(tag2DefaultMap map[string]interface{}) *Table {
	for tag, defaultVal := range tag2DefaultMap {
		t.setWaitHandleStructFieldMap(tag, func(val *handleStructField) {
			val.defaultVal = defaultVal
		})
	}
	return t
}

// SetMarshalFn 设置 struct 字段待序列化方法
// 注: 调用必须优先 Insert/Update 操作的方法, 防止通过对象解析字段时被排除
func (t *Table) SetMarshalFn(fn marshalFn, tags ...string) *Table {
	for _, tag := range tags {
		t.setWaitHandleStructFieldMap(tag, func(val *handleStructField) {
			val.marshal = fn
		})
	}
	return t
}

// SetUnmarshalFn 设置 struct 字段待反序列化方法
// 注: 调用必须优先于 SelectAuto, 防止 SelectAuto 解析时查询字段被排除
func (t *Table) SetUnmarshalFn(fn unmarshalFn, tags ...string) *Table {
	for _, tag := range tags {
		t.setWaitHandleStructFieldMap(tag, func(val *handleStructField) {
			val.unmarshal = fn
		})
	}
	return t
}

// parseCol2StructField 通过解析输入结构体, 返回 map[tag名]字段偏移量, 同时缓存起来
func (t *Table) parseCol2StructField(ty reflect.Type, isNeedSort bool) (col2StructFieldMap map[string]structField, sortCol []string) {
	// 非结构体就返回空
	if ty.Kind() != reflect.Struct {
		return nil, nil
	}

	// 通过地址来取, 防止出现重复
	// 当 t.waitHandleStructFieldMap != nil 不等于空时, 为了防止解析 selectFields 缺少, 不能走缓存中取
	if t.waitHandleStructFieldMap == nil {
		if cacheVal, ok := cacheStructType2StructFieldMap.Load(ty); ok { // 需要排除再包含 t.waitHandleStructFieldMap 不为空的
			col2StructFieldMap = cacheVal.(map[string]structField)
			if isNeedSort { // 按照col2FieldIndexMap的value进行排序
				l := len(col2StructFieldMap)
				tmpMap := make(map[int]string, l)
				tmpSortVal := make([]int, 0, l)
				for col, field := range col2StructFieldMap {
					tmpMap[field.offsetIndex] = col
					tmpSortVal = append(tmpSortVal, field.offsetIndex)
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
	col2StructFieldMap = make(map[string]structField, fieldNum)
	sortCol = make([]string, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		col, tag, _ := t.parseStructField(ty.Field(i), sureUnmarshal)
		if null(col) {
			continue
		}

		col2StructFieldMap[col] = structField{
			offsetIndex: i,
			tagName:     tag,
		}
		sortCol = append(sortCol, col)
	}

	if t.waitHandleStructFieldMap == nil {
		cacheStructType2StructFieldMap.Store(ty, col2StructFieldMap)
	}
	return
}

// parseStructField 从结构体的 tag 中解析出列名, 同时跳过嵌套, 包含: 对象, 指针对象, 切片, 不可导出字段
func (t *Table) parseStructField(fieldInfo reflect.StructField, args ...uint8) (col, tag string, need bool) {
	if !isExported(fieldInfo.Name) {
		return
	}

	// 解析 tag 中的列名
	tag = fieldInfo.Tag.Get(t.tag)
	if null(tag) {
		return
	}

	// 去除 tag 中的干扰, 如: json:"xxx,omitempty"
	tag = t.parseTag2Col(tag)

	// 处理下 tag
	var alias string
	handleStructField, needHandleField := t.waitHandleStructFieldMap[tag]
	if needHandleField {
		if handleStructField.needExclude { // 需要排除
			return
		}
		alias = handleStructField.tagAlias // tag 的别名, 用于待解析 col 名
	}

	// 如果是跳过嵌套/对象等, 同时需要判断下是否有序列化/反序列
	if t.needSkipObj(fieldInfo.Type.Kind()) {
		if len(args) == 0 {
			return
		}

		if !needHandleField {
			return
		}

		switch args[0] {
		case sureMarshal:
			need = handleStructField.marshal != nil
		case sureUnmarshal:
			need = handleStructField.unmarshal != nil
		}
		if !need {
			return
		}
	}

	col = tag
	if !null(alias) {
		col = alias
	}
	return
}

// needSkipObj 默认不处理嵌套对象
func (t *Table) needSkipObj(kind reflect.Kind) bool {
	switch kind {
	case reflect.Struct, reflect.Ptr, reflect.Slice, reflect.Array:
		return true
	}
	return false
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

// GetSqlObj 获取 SqlStrObj, 方便外部使用该对象的方法
func (t *Table) GetSqlObj() *SqlStrObj {
	return t.tmpSqlObj
}

// sqlObjIsNil 判断 sqlObj 是否为空
func (t *Table) sqlObjIsNil() bool {
	return t.tmpSqlObj == nil
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
		sLog.Error("sql only support string/SqlStrObjPtr")
		return t
	}
	t.tmpSqlObj.SetStrSymbol(t.getStrSymbol())
	return t
}

// Exec 执行
func (t *Table) Exec() (sql.Result, error) {
	defer t.free()
	if err := t.prevCheck(); err != nil {
		return nil, err
	}
	return t.db.Exec(t.tmpSqlObj.SetPrintLog(t.isPrintSql).SetCallerSkip(t.printSqlCallSkip).GetSqlStr())
}

// prevCheck 查询预检查
func (t *Table) prevCheck(checkSqlObj ...bool) error {
	if t.haveFree {
		return errors.New("tableObj have free, you can't again use")
	}

	if t.db == nil {
		return errors.New("db is nil")
	}

	defaultCheckSqlObj := true
	if len(checkSqlObj) > 0 {
		defaultCheckSqlObj = checkSqlObj[0]
	}

	if defaultCheckSqlObj && t.sqlObjIsNil() {
		if null(t.name) {
			return tableNameIsUnknownErr
		}
		return errors.New("tmpSqlObj is nil")
	}
	return nil
}

// parseTableName 解析表名
func parseTableName(objName string) string {
	// 排除有如含有表别名, 如 user_info ui => user_info
	if index := strings.Index(objName, " "); index != -1 {
		return objName[:index]
	} else if strings.Contains(objName, "_") { // 直接包含下划线
		return objName
	}

	// 解析对象名
	res := getTmpBuf(len(objName))
	defer putTmpBuf(res)
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
