package builder

import (
	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
)

var _ Builder = (*Where)(nil)

type Where struct {
	*builder
	dbType dialect.DbType
}

func NewWhere(dt dialect.DbType) *Where {
	obj := &Where{
		dbType:  dt,
		builder: newBuilder(dt),
	}
	return obj
}

func (w *Where) Eq(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) OrEq(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) IsNull(field string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field) + " IS NULL")
}

func (w *Where) OrIsNull(field string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field) + " IS NULL")
}

func (w *Where) NotEq(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) OrNotEq(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) Gt(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) OrGt(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) Gte(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) OrGte(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) Lt(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) OrLt(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) Lte(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) OrLte(field string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) Between(field string, arg1, arg2 interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) OrBetween(field string, arg1, arg2 interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) In(field string, args ...interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *Where) OrIn(field string, args ...interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *Where) LikeLeft(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) OrLikeLeft(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) LikeRight(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) OrLikeRight(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) Like(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) OrLike(field string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpField(gd, field)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) And(sqlStr string, args ...interface{}) *Where {
	if w.len() > 0 {
		w.appendSql(" AND ")
	}
	w.appendSql2Args(sqlStr, args...)
	return w
}

func (w *Where) Or(sqlStr string, args ...interface{}) *Where {
	if w.len() > 0 {
		w.appendSql(" OR ")
	}
	w.appendSql2Args(sqlStr, args...)
	return w
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
