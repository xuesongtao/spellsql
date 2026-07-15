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
	haveSet := u.HaveStr(" SET")
	if u.tableName != "" {
		b.writeSql("UPDATE ")
		b.writeSql(u.tableName)
	}

	if len(u.columns) > 0 {
		if !haveSet {
			b.writeSql(" SET ")
		} else {
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
