package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
)

type WhereBuilder struct {
	dbType   dialect.DbType
	finalBuf strings.Builder // 记录带有占位符 (?) 的 WHERE 条件
	args     []interface{}   // 记录 WHERE 条件的所有参数
}

func NewWhereBuilder(dt dialect.DbType) *WhereBuilder {
	obj := &WhereBuilder{
		dbType: dt,
		args:   make([]interface{}, 0, 4),
	}
	return obj
}

func (w *WhereBuilder) Eq(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" = "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrEq(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" = "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) IsNull(field string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field) + " IS NULL")
}

func (w *WhereBuilder) OrIsNull(field string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field) + " IS NULL")
}

func (w *WhereBuilder) NotEq(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" <> "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrNotEq(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" <> "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) Gt(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" > "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrGt(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" > "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) Gte(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" >= "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrGte(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" >= "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) Lt(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" < "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrLt(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" < "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) Lte(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" <= "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) OrLte(field string, arg interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" <= "+dialect.Placeholders(), arg)
}

func (w *WhereBuilder) Between(field string, arg1, arg2 interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *WhereBuilder) OrBetween(field string, arg1, arg2 interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *WhereBuilder) In(field string, args ...interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *WhereBuilder) OrIn(field string, args ...interface{}) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *WhereBuilder) LikeLeft(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *WhereBuilder) OrLikeLeft(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *WhereBuilder) LikeRight(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *WhereBuilder) OrLikeRight(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *WhereBuilder) Like(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *WhereBuilder) OrLike(field string, arg string) *WhereBuilder {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *WhereBuilder) And(sqlStr string, args ...interface{}) *WhereBuilder {
	if w.finalBuf.Len() > 0 {
		w.finalBuf.WriteString(" AND ")
	}
	w.finalBuf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) Or(sqlStr string, args ...interface{}) *WhereBuilder {
	if w.finalBuf.Len() > 0 {
		w.finalBuf.WriteString(" OR ")
	}
	w.finalBuf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) GetNoParseSql2Args() (string, []interface{}) {
	return w.finalBuf.String(), w.args
}

func (w *WhereBuilder) GetSqlStr() string {
	return dialect.NewParsePlaceholder(w.dbType, w.finalBuf.String(), w.args...).Parse().Result()
}

func (w *WhereBuilder) GetSql2Args() (string, []interface{}) {
	return dialect.NewParsePlaceholder(w.dbType, w.finalBuf.String(), w.args...).Replace().Result(), w.args
}

// EscapeLike 转义 like
func EscapeLike(val string) string {
	res := internal.Escape(
		[]byte(val),
		map[byte][]byte{
			'_': {'\\', '_'},
			'%': {'\\', '%'},
		})
	return string(res)
}
