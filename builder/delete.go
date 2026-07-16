package builder

import (
	"gitee.com/xuesongtao/spellsql/v2/dialect"
)

var _ SQLBuilder = (*Delete)(nil)

type Delete struct {
	*Builder
	tableName string
	where     *Where
}

func NewDelete(dt ...dialect.DbType) *Delete {
	obj := &Delete{
		Builder: NewBuilder(dt...),
		where:   NewWhere(dt...),
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
	return d
}

func (d *Delete) mergeSQL(b *Builder) {
	if d.tableName != "" {
		b.writeSql("DELETE FROM ")
		b.writeSql(d.tableName)
	}
	if d.where != nil && !d.where.empty() {
		d.mergeWhere(d.where)
	}
}
