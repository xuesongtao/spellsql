package spellsql

import "strings"

type WhereBuilder struct {
	buf  strings.Builder // 记录带有占位符 (?) 的 WHERE 条件
	args []interface{}   // 记录 WHERE 条件的所有参数
}

func NewWhereBuilder() *WhereBuilder {
	obj := &WhereBuilder{
		buf:  strings.Builder{},
		args: make([]interface{}, 0, 4),
	}
	return obj
}

func (w *WhereBuilder) Where(sqlStr string, args ...interface{}) *WhereBuilder {
	w.buf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) And(sqlStr string, args ...interface{}) *WhereBuilder {
	if w.buf.Len() > 0 {
		w.buf.WriteString(" AND ")
	}
	w.buf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) Or(sqlStr string, args ...interface{}) *WhereBuilder {
	if w.buf.Len() > 0 {
		w.buf.WriteString(" OR ")
	}
	w.buf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) Append(sqlStr string, args ...interface{}) *WhereBuilder {
	w.buf.WriteString(sqlStr)
	w.args = append(w.args, args...)
	return w
}

func (w *WhereBuilder) GetExecArgs() (string, []interface{}) {
	return w.buf.String(), w.args
}

func (w *WhereBuilder) GetSqlStr() string {
	return w.buf.String()
}
