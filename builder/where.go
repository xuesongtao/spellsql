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
	return w.And(w.warpCol(col)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) OrEq(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" = "+dialect.Placeholders(), arg)
}

func (w *Where) IsNull(col string) *Where {
	return w.And(w.warpCol(col) + " IS NULL")
}

func (w *Where) OrIsNull(col string) *Where {
	return w.Or(w.warpCol(col) + " IS NULL")
}

func (w *Where) NotEq(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) OrNotEq(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" <> "+dialect.Placeholders(), arg)
}

func (w *Where) Gt(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) OrGt(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" > "+dialect.Placeholders(), arg)
}

func (w *Where) Gte(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) OrGte(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" >= "+dialect.Placeholders(), arg)
}

func (w *Where) Lt(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) OrLt(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" < "+dialect.Placeholders(), arg)
}

func (w *Where) Lte(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) OrLte(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" <= "+dialect.Placeholders(), arg)
}

func (w *Where) Between(col string, arg1, arg2 interface{}) *Where {
	return w.And(w.warpCol(col)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) OrBetween(col string, arg1, arg2 interface{}) *Where {
	return w.Or(w.warpCol(col)+" (BETWEEN "+dialect.Placeholders()+" AND "+dialect.Placeholders()+")", arg1, arg2)
}

func (w *Where) In(col string, arg interface{}) *Where {
	return w.And(w.warpCol(col)+" IN ("+dialect.Placeholders()+")", arg)
}

func (w *Where) OrIn(col string, arg interface{}) *Where {
	return w.Or(w.warpCol(col)+" IN ("+dialect.Placeholders()+")", arg)
}

func (w *Where) LikeLeft(col string, arg string) *Where {
	return w.And(w.warpCol(col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) OrLikeLeft(col string, arg string) *Where {
	return w.Or(w.warpCol(col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg))
}

func (w *Where) LikeRight(col string, arg string) *Where {
	return w.And(w.warpCol(col)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) OrLikeRight(col string, arg string) *Where {
	return w.Or(w.warpCol(col)+" LIKE "+dialect.Placeholders(), EscapeLike(arg)+"%")
}

func (w *Where) Like(col string, arg string) *Where {
	return w.And(w.warpCol(col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) OrLike(col string, arg string) *Where {
	return w.Or(w.warpCol(col)+" LIKE "+dialect.Placeholders(), "%"+EscapeLike(arg)+"%")
}

func (w *Where) WhereCb(f func(wb *Where)) *Where {
	f(w)
	return w
}

// And 添加 AND 条件, 如果已有条件, 则追加 AND
func (w *Where) And(sqlStr string, args ...interface{}) *Where {
	if w.len() > 0 {
		w.writeSql(" AND ")
	}
	w.writeSql2Args(sqlStr, args...)
	return w
}

// AndGroup 外部传入 new builder.Where 作为一个整体进行拼接
// 格式如: AND (xxx AND xxx)
func (w *Where) AndGroup(wb *Where) *Where {
	sqlStr, args := wb.GetNoParseSql2Args()
	w.And("("+sqlStr+")", args...)
	return w
}

// AndNewGroup 内部 new 一个 builder.Where 作为一个整体进行拼接
// 格式如: AND (xxx AND xxx)
func (w *Where) AndNewGroup(cb func(wb *Where)) *Where {
	wb := NewWhere(w.dbType)
	cb(wb)
	return w.AndGroup(wb)
}

// Or 添加 OR 条件, 如果已有条件, 则追加 OR
func (w *Where) Or(sqlStr string, args ...interface{}) *Where {
	if w.len() > 0 {
		w.writeSql(" OR ")
	}
	w.writeSql2Args(sqlStr, args...)
	return w
}

// OrGroup 外部传入 new builder.Where 作为一个整体进行拼接
// 格式如: OR (xxx OR xxx)
func (w *Where) OrGroup(wb *Where) *Where {
	sqlStr, args := wb.GetNoParseSql2Args()
	w.Or("("+sqlStr+")", args...)
	return w
}

// OrNewGroup 内部 new 一个 builder.Where 作为一个整体进行拼接
// 格式如: OR (xxx OR xxx)
func (w *Where) OrNewGroup(cb func(wb *Where)) *Where {
	wb := NewWhere(w.dbType)
	cb(wb)
	return w.OrGroup(wb)
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
