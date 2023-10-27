package spellsql

import (
	"context"
	"database/sql"
)

// *******************************************************************************
// *                             spellsql 常用封装                                 *
// *******************************************************************************

// GetSqlStr 适用直接获取 sqlStr, 每次会自动打印日志
func GetSqlStr(sqlStr string, args ...interface{}) string {
	return NewCacheSql(sqlStr, args...).SetCallerSkip(2).GetSqlStr()
}

// GetSqlStrCtx 适用直接获取 sqlStr, 每次会自动打印日志
func GetSqlStrCtx(ctx context.Context, sqlStr string, args ...interface{}) string {
	return NewCacheSql(sqlStr, args...).SetCtx(ctx).SetCallerSkip(2).GetSqlStr()
}

// FmtSqlStr 适用直接获取 sqlStr, 不会打印日志
func FmtSqlStr(sqlStr string, args ...interface{}) string {
	return NewCacheSql(sqlStr, args...).FmtSql()
}

// GetLikeSqlStr 针对 LIKE 语句, 只有一个条件
// 如: obj := GetLikeSqlStr(ALK, "SELECT id, username FROM sys_user", "name", "xue")
//
//	=> SELECT id, username FROM sys_user WHERE name LIKE "%xue%"
func GetLikeSqlStr(likeType uint8, sqlStr, fieldName, value string, printLog ...bool) string {
	sqlObj := NewCacheSql(sqlStr)
	switch likeType {
	case ALK:
		sqlObj.SetAllLike(fieldName, value)
	case RLK:
		sqlObj.SetRightLike(fieldName, value)
	case LLK:
		sqlObj.SetLeftLike(fieldName, value)
	}
	isPrintLog := false
	endSymbol := ""

	// 判断下是否打印 log
	if len(printLog) > 0 {
		isPrintLog = true
		endSymbol = ";"
	}
	return sqlObj.SetPrintLog(isPrintLog).SetCallerSkip(2).GetSqlStr("sqlStr", endSymbol)
}

// *******************************************************************************
// *                             orm 常用封装                                     *
// *******************************************************************************

// IsNullRow 根据 err 判断是否结果为空
func IsNullRow(err error) bool {
	return err == nullRowErr
}

// ExecForSql 根据 sql 进行执行 INSERT/UPDATE/DELETE 等操作
// sql sqlStr 或 *SqlStrObj
func ExecForSql(db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).Exec()
}

// ExecForSqlCtx 根据 sql 进行执行 INSERT/UPDATE/DELETE 等操作
// sql sqlStr 或 *SqlStrObj
func ExecForSqlCtx(ctx context.Context, db DBer, sql interface{}) (sql.Result, error) {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).Raw(sql).Exec()
}

// Count 获取总数
func Count(db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).PrintSqlCallSkip(3).SelectCount().Where(where, args...).Count(dest)
}

// CountCtx 获取总数
func CountCtx(ctx context.Context, db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).SelectCount().Where(where, args...).Count(dest)
}

// InsertForObj 根据对象新增
func InsertForObj(db DBer, tableName string, src ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Insert(src...).Exec()
}

// InsertForObjCtx 根据对象新增
func InsertForObjCtx(ctx context.Context, db DBer, tableName string, src ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).Insert(src...).Exec()
}

// InsertHasDefaultForObj 根据对象新增, 同时支持默认值
func InsertHasDefaultForObj(db DBer, tableName string, tag2DefaultMap map[string]interface{}, src interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).TagDefault(tag2DefaultMap).Insert(src).Exec()
}

// InsertHasDefaultForObjCtx 根据对象新增, 同时支持默认值
func InsertsHasDefaultForObjCtx(ctx context.Context, db DBer, tableName string, tag2DefaultMap map[string]interface{}, src ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).TagDefault(tag2DefaultMap).Insert(src...).Exec()
}

// InsertHasDefaultForObjCtx 根据对象新增, 同时支持默认值
func InsertHasDefaultForObjCtx(ctx context.Context, db DBer, tableName string, tag2DefaultMap map[string]interface{}, src interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).TagDefault(tag2DefaultMap).Insert(src).Exec()
}

// InsertODKUForObj 根据对象新增, 冲突更新
func InsertODKUForObj(db DBer, tableName string, src interface{}, keys ...string) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).InsertODKU(src, keys...).Exec()
}

// InsertODKUForObjCtx 根据对象新增, 冲突更新
func InsertODKUForObjCtx(ctx context.Context, db DBer, tableName string, src interface{}, keys ...string) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).InsertODKU(src, keys...).Exec()
}

// InsertIgForObj 根据对象新增, 冲突忽略
func InsertIgForObj(db DBer, tableName string, src interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).InsertIg(src).Exec()
}

// InsertIgForObjCtx 根据对象新增, 冲突忽略
func InsertIgForObjCtx(ctx context.Context, db DBer, tableName string, src interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).InsertIg(src).Exec()
}

// UpdateForObj 根据对象更新
func UpdateForObj(db DBer, tableName string, src interface{}, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Update(src, where, args...).Exec()
}

// UpdateForObjCtx 根据对象更新
func UpdateForObjCtx(ctx context.Context, db DBer, tableName string, src interface{}, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).Update(src, where, args...).Exec()
}

// DeleteWhere 根据条件删除
func DeleteWhere(db DBer, tableName string, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).PrintSqlCallSkip(3).Delete().Where(where, args...).Exec()
}

// DeleteWhereCtx 根据条件删除
func DeleteWhereCtx(ctx context.Context, db DBer, tableName string, where string, args ...interface{}) (sql.Result, error) {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).Delete().Where(where, args...).Exec()
}

// FindWhere 查询对象中的字段内容
func FindWhere(db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).PrintSqlCallSkip(3).FindWhere(dest, where, args...)
}

// FindWhereCtx 查询对象中的字段内容
func FindWhereCtx(ctx context.Context, db DBer, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db, tableName).Ctx(ctx).PrintSqlCallSkip(3).FindWhere(dest, where, args...)
}

// SelectFindWhere 查询指定内容的
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindWhere(db DBer, fields interface{}, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).FindWhere(dest, where, args...)
}

// SelectFindWhereCtx 查询指定内容的
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindWhereCtx(ctx context.Context, db DBer, fields interface{}, tableName string, dest interface{}, where string, args ...interface{}) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).SelectAuto(fields, tableName).FindWhere(dest, where, args...)
}

// SelectFindOne 单行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOne(db DBer, fields interface{}, tableName string, where string, dest ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOne(dest...)
}

// SelectFindOneCtx 单行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOneCtx(ctx context.Context, db DBer, fields interface{}, tableName string, where string, dest ...interface{}) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOne(dest...)
}

// SelectFindOneFn 单行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOneFn(db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOneFn(dest, fn...)
}

// SelectFindOneFnCtx 单行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOneFnCtx(ctx context.Context, db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOneFn(dest, fn...)
}

// SelectFindOneIgnoreResult 查询结果支持多个, 此使用场景为需要使用 SelectCallBackFn 对每行进行处理
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOneIgnoreResult(db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOneIgnoreResult(dest, fn...)
}

// SelectFindOneIgnoreResultCtx 查询结果支持多个, 此使用场景为需要使用 SelectCallBackFn 对每行进行处理
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindOneIgnoreResultCtx(cxt context.Context, db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Ctx(cxt).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindOneIgnoreResult(dest, fn...)
}

// SelectFindAll 多行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindAll(db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindAll(dest, fn...)
}

// SelectFindAllCtx 多行指定内容查询
// fields 可以字符串(如: "name,age,addr"), 同时也可以为 struct/struct slice(如: Man/[]Man), 会将 struct 的字段解析为查询内容
func SelectFindAllCtx(ctx context.Context, db DBer, fields interface{}, tableName string, where string, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).SelectAuto(fields, tableName).Where(where).FindAll(dest, fn...)
}

// FindOne 单查询
// sql sqlStr 或 *SqlStrObj
func FindOne(db DBer, sql interface{}, dest ...interface{}) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindOne(dest...)
}

// FindOneCtx 单查询
// sql sqlStr 或 *SqlStrObj
func FindOneCtx(ctx context.Context, db DBer, sql interface{}, dest ...interface{}) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).Raw(sql).FindOne(dest...)
}

// FindOneFn 单查询
// sql sqlStr 或 *SqlStrObj
func FindOneFn(db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindOneFn(dest, fn...)
}

// FindOneFnCtx 单查询
// sql sqlStr 或 *SqlStrObj
func FindOneFnCtx(ctx context.Context, db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).Raw(sql).FindOneFn(dest, fn...)
}

// FindAll 多查询
// sql sqlStr 或 *SqlStrObj
func FindAll(db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).PrintSqlCallSkip(3).Raw(sql).FindAll(dest, fn...)
}

// FindAllCtx 多查询
// sql sqlStr 或 *SqlStrObj
func FindAllCtx(ctx context.Context, db DBer, sql interface{}, dest interface{}, fn ...SelectCallBackFn) error {
	return NewTable(db).Ctx(ctx).PrintSqlCallSkip(3).Raw(sql).FindAll(dest, fn...)
}

// ConvStruct 转换 struct 的值
func ConvStruct(src interface{}, dest interface{}) error {
	obj := NewConvStruct()
	if err := obj.Init(src, dest); err != nil {
		return err
	}
	return obj.Convert()
}
