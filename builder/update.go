package builder

import "gitee.com/xuesongtao/spellsql/v2/dialect"

var _ SQLBuilder = (*Update)(nil)

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
		builder: NewBuilder(dt),
		where:   NewWhere(dt),
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
	u.SetWhere(wb)
	return u
}

func (u *Update) SetWhere(where *Where) *Update {
	u.where = where
	return u
}

func (u *Update) mergeSQL(b *builder) {
	if u.tableName != "" {
		b.appendSql("UPDATE ")
		b.appendSql(u.tableName)
		b.appendSql(" SET ")
	}

	dg := dialect.GetDialect(u.dbType)
	for i, col := range u.columns {
		if i > 0 {
			b.appendSql(", ")
		}
		b.appendSql2Args(dialect.WarpCol(dg, col)+" = "+dialect.Placeholders(), u.values[i])
	}

	if u.where != nil && !u.where.empty() {
		sqlStr, sqlArgs := u.where.GetNoParseSql2Args()
		b.appendSql(" WHERE ")
		b.appendSql2Args(sqlStr, sqlArgs...)
	}
}
