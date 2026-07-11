package builder

import (
	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
)

var _ Builder = (*Insert)(nil)

type Insert struct {
	*builder
	insertType  internal.OpType
	tableName   string
	columns     []string
	values      [][]interface{}
	conflictCol string
	duplicate   []string // ON DUPLICATE KEY UPDATE
}

func NewInsert(dt dialect.DbType) *Insert {
	obj := &Insert{
		builder: newBuilder(dt),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (i *Insert) Into(tableName string) *Insert {
	i.insertType = internal.INSERT
	i.tableName = tableName
	return i
}

func (i *Insert) IntoReplace(tableName string) *Insert {
	i.insertType = internal.INSERT_REPLACE
	i.tableName = tableName
	return i
}

func (i *Insert) IntoIgnore(tableName string) *Insert {
	i.insertType = internal.INSERT_IGNORE
	i.tableName = tableName
	return i
}

func (i *Insert) IntoOnDuplicate(tableName string) *Insert {
	i.insertType = internal.INSERT_ON_DUPLICATE
	i.tableName = tableName
	if i.duplicate == nil {
		i.duplicate = make([]string, 0, 5)
	}
	return i
}

func (i *Insert) Columns(cols ...string) *Insert {
	if i.columns == nil {
		i.columns = make([]string, 0, len(cols))
	}
	i.columns = append(i.columns, cols...)
	return i
}

func (i *Insert) Values(vals ...interface{}) *Insert {
	if i.values == nil {
		i.values = make([][]interface{}, 0, len(vals))
	}
	i.values = append(i.values, vals)
	return i
}

// DuplicateUpdate 设置 ON DUPLICATE KEY UPDATE 的字段和可选的冲突字段（仅用于 Postgres）
func (i *Insert) DuplicateUpdate(cols []string, conflictCol ...string) *Insert {
	if i.duplicate == nil {
		i.duplicate = make([]string, 0, len(cols))
	}
	i.duplicate = append(i.duplicate, cols...)
	if len(conflictCol) > 0 {
		i.conflictCol = conflictCol[0]
	}
	return i
}

func (i *Insert) mergeSQL() {
	i.appendSql("INSERT ")
	switch i.insertType {
	case internal.INSERT_REPLACE:
		i.appendSql("REPLACE ")
	case internal.INSERT_IGNORE:
		i.appendSql("IGNORE ")
	}
	i.appendSql("INTO " + i.tableName)
	gd := dialect.GetDialect(i.dbType)
	if len(i.columns) > 0 {
		i.appendSql("(" + dialect.WarpJoinFields(gd, i.columns...) + ")")
	}
	if len(i.values) > 0 {
		i.appendSql(" VALUES ")
		for index, val := range i.values {
			if index > 0 {
				i.appendSql(", ")
			}
			i.appendSql("(" + dialect.Placeholders(len(val)) + ")")
			i.appendArgs(val...)
		}
	}
	if len(i.duplicate) > 0 {
		switch i.dbType {
		case dialect.Postgres:
			i.appendSql(" ON CONFLICT (" + dialect.WarpField(gd, i.conflictCol) + ") DO UPDATE SET ")
			for index, col := range i.duplicate {
				if index > 0 {
					i.appendSql(", ")
				}
				wCol := dialect.WarpField(gd, col)
				i.appendSql(wCol + "=EXCLUDED." + wCol)
			}
		default:
			i.appendSql(" ON DUPLICATE KEY UPDATE ")
			for index, col := range i.duplicate {
				if index > 0 {
					i.appendSql(", ")
				}
				wCol := dialect.WarpField(gd, col)
				i.appendSql(wCol + "=VALUES(" + wCol + ")")
			}
		}
	}
}
