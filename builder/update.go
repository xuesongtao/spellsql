package builder

import "gitee.com/xuesongtao/spellsql/v2/dialect"

var _ SQLBuilder = (*Update)(nil)

type Update struct {
	*Builder
	tableName string
	columns   []string
	values    []interface{}
	where     *Where
}

func NewUpdate(dt ...dialect.DbType) *Update {
	obj := &Update{
		Builder: NewBuilder(dt...),
		where:   NewWhere(dt...),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (u *Update) Table(tableName string) *Update {
	u.tableName = tableName
	return u
}

func (u *Update) Set(col string, value interface{}) *Update {
	if u.columns == nil {
		u.columns = make([]string, 0, 5)
	}
	if u.values == nil {
		u.values = make([]interface{}, 0, 5)
	}
	u.columns = append(u.columns, col)
	u.values = append(u.values, value)
	return u
}

func (u *Update) Where() *Where {
	return u.where
}

func (u *Update) WhereCb(f func(wb *Where)) *Update {
	wb := u.Where()
	f(wb)
	return u
}

func (u *Update) SetWhere(where *Where) *Update {
	u.where = where
	return u
}

func (u *Update) mergeSQL(b *Builder) {
	if u.tableName != "" {
		b.writeSql("UPDATE ")
		b.writeSql(u.tableName)
	}

	if len(u.columns) > 0 {
		if i := u.index(" SET"); i == -1 {
			b.writeSql(" SET ")
		} else if i+3+2 < u.len()-1 { // 如: " SET x", 需要加 ,
			b.writeSql(", ")
		} else if i+3 == u.len()-1 { // " SET" 后面没有内容, 直接追加
			b.writeSql(" ")
		}
		for i, col := range u.columns {
			if i > 0 {
				b.writeSql(", ")
			}
			b.writeSql2Args(u.warpCol(col)+" = "+Placeholders(), u.values[i])
		}
	}

	if u.where != nil && !u.where.empty() {
		u.mergeWhere(u.where)
	}
}
