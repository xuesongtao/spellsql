package builder

import (
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
)

var _ SQLBuilder = (*Where)(nil)

type Where struct {
	*Builder
}

func NewWhere(dt ...dialect.DbType) *Where {
	obj := &Where{
		Builder: NewBuilder(dt...),
	}
	return obj
}

func (w *Where) Eq(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) OrEq(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) IsNull(col string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col) + " IS NULL")
}

func (w *Where) OrIsNull(col string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col) + " IS NULL")
}

func (w *Where) NotEq(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) OrNotEq(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) Gt(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) OrGt(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) Gte(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) OrGte(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) Lt(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) OrLt(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) Lte(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) OrLte(col string, arg interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) Between(col string, arg1, arg2 interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) OrBetween(col string, arg1, arg2 interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) In(col string, args ...interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *Where) OrIn(col string, args ...interface{}) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" IN ("+dialect.Placeholders(len(args))+")", args...)
}

func (w *Where) LikeLeft(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) OrLikeLeft(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) LikeRight(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) OrLikeRight(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) Like(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.And(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) OrLike(col string, arg string) *Where {
	gd := dialect.GetDialect(w.dbType)
	return w.Or(dialect.WarpCol(gd, col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) WhereCb(f func(wb *Where)) *Where {
	f(w)
	return w
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

func (w *Where) Empty() bool {
	return w.len() == 0
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

// WhereCb 根据不同的 builder 类型调用对应的 WhereCb 方法
func WhereCb(bld SQLBuilder, cb func(wb *Where)) {
	switch b := bld.(type) {
	case *Select:
		b.WhereCb(cb)
	case *Update:
		b.WhereCb(cb)
	case *Delete:
		b.WhereCb(cb)
	case *Where:
		b.WhereCb(cb)
	}
}
