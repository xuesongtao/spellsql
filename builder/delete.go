package builder

import "gitee.com/xuesongtao/spellsql/dialect"

var _ Builder = (*Delete)(nil)

type Delete struct {
	*builder
	dbType    dialect.DbType
	tableName string
	where     *Where
}

func NewDelete(dt dialect.DbType) *Delete {
	obj := &Delete{
		dbType:  dt,
		builder: newBuilder(dt),
		where:   NewWhere(dt),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (d *Delete) From(tableName string) *Delete {
	d.tableName = tableName
	return d
}

func (d *Delete) Where() *Where {
	return d.where
}

func (d *Delete) SetWhere(where *Where) *Delete {
	d.where = where
	return d
}

func (d *Delete) WhereCb(f func(wb *Where)) *Delete {
	wb := d.Where()
	f(wb)
	d.SetWhere(wb)
	return d
}

func (d *Delete) mergeSQL() {
	d.appendSql("DELETE FROM ")
	d.appendSql(d.tableName)
	if d.where != nil && !d.where.empty() {
		d.appendSql(" WHERE ")
		whereSql, whereArgs := d.where.GetSql2Args()
		d.appendSql2Args(whereSql, whereArgs...)
	}
}
