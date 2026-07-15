package builder

import (
	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

var _ SQLBuilder = (*Insert)(nil)

type Insert struct {
	*Builder
	insertType  internal.OpType
	tableName   string
	columns     []string
	values      [][]interface{}
	conflictCol string
	duplicate   []string // ON DUPLICATE KEY UPDATE
}

func NewInsert(dt ...dialect.DbType) *Insert {
	obj := &Insert{
		insertType: internal.None,
		Builder:    NewBuilder(dt...),
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

func (i *Insert) mergeSQL(b *Builder) {
	if i.insertType != internal.None {
		if i.insertType == internal.INSERT_REPLACE {
			b.writeSql("REPLACE ")
		} else {
			b.writeSql("INSERT ")
		}
		switch i.insertType {
		case internal.INSERT_IGNORE:
			b.writeSql("IGNORE ")
		}
		b.writeSql("INTO " + i.tableName)
	}

	if len(i.columns) > 0 {
		b.writeSql("(" + i.warpJoinCols(i.columns...) + ")")
	}

	if len(i.values) > 0 {
		valsIndex := utils.Index(b.finalSql.String(), "VALUES", true)
		if valsIndex > -1 {
			b.writeSql(" ")
		} else {
			b.writeSql(" VALUES ")
		}
		for index, val := range i.values {
			if index > 0 {
				b.writeSql(", ")
			}
			b.writeSql("(" + Placeholders(len(val)) + ")")
			b.writeArgs(val...)
		}
	}
	if len(i.duplicate) > 0 {
		switch i.dbType {
		case dialect.Postgres:
			b.writeSql(" ON CONFLICT (" + i.warpCol(i.conflictCol) + ") DO UPDATE SET ")
			for index, col := range i.duplicate {
				if index > 0 {
					b.writeSql(", ")
				}
				wCol := i.warpCol(col)
				b.writeSql(wCol + "=EXCLUDED." + wCol)
			}
		default:
			b.writeSql(" ON DUPLICATE KEY UPDATE ")
			for index, col := range i.duplicate {
				if index > 0 {
					b.writeSql(", ")
				}
				wCol := i.warpCol(col)
				b.writeSql(wCol + "=VALUES(" + wCol + ")")
			}
		}
	}
}
