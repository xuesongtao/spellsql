package builder

import "gitee.com/xuesongtao/spellsql/dialect"

type Update struct {
	*builder
	dbType    dialect.DbType
	tableName string
	columns   []string
	values    []interface{}
	where     *Where
}

func NewUpdate(dt dialect.DbType) *Update {
	obj := &Update{
		dbType:  dt,
		builder: newBuilder(dt),
		where:   NewWhere(dt),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (u *Update) Table(tableName string) *Update {
	u.tableName = tableName
	return u
}

func (u *Update) Set(column string, value interface{}) *Update {
	if u.columns == nil {
		u.columns = make([]string, 0, 5)
	}
	if u.values == nil {
		u.values = make([]interface{}, 0, 5)
	}
	u.columns = append(u.columns, column)
	u.values = append(u.values, value)
	return u
}

func (u *Update) Where() *Where {
	return u.where
}

func (u *Update) WhereCb(f func(wb *Where)) *Update {
	wb := u.Where()
	f(wb)
	u.SetWhere(wb)
	return u
}

func (u *Update) SetWhere(where *Where) *Update {
	u.where = where
	return u
}

func (u *Update) mergeSQL() {
	u.appendSql("UPDATE ")
	u.appendSql(u.tableName)
	u.appendSql(" SET ")
	dg := dialect.GetDialect(u.dbType)
	for i, col := range u.columns {
		if i > 0 {
			u.appendSql(", ")
		}
		u.appendSql2Args(dialect.WarpField(dg, col)+" = "+dialect.Placeholders(), u.values[i])
	}
	if u.where != nil && !u.where.empty() {
		sqlStr, sqlArgs := u.where.GetNoParseSql2Args()
		u.appendSql(" WHERE ")
		u.appendSql2Args(sqlStr, sqlArgs...)
	}
}
