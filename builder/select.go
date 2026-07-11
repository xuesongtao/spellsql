package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/dialect"
	"gitee.com/xuesongtao/spellsql/internal"
	"gitee.com/xuesongtao/spellsql/utils"
)

var _ Builder = (*Select)(nil)

type Select struct {
	*builder
	dbType     dialect.DbType
	columns    []string // 存储 SELECT 的列
	tableName  string   // 存储表名
	joins      []string // 存储 JOIN 语句
	where      *Where   // 存储 WHERE 条件
	groupBys   []string // GROUP BY
	havingStr  string   // HAVING 条件
	havingArgs []interface{}
	orderBys   []string // ORDER BY
	limit      int
	offset     int
}

func NewSelect(dt dialect.DbType) *Select {
	obj := &Select{
		dbType:  dt,
		builder: newBuilder(dt),
		where:   NewWhere(dt),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (s *Select) Select(col ...string) *Select {
	s.columns = append(s.columns, col...)
	return s
}

func (s *Select) From(table string) *Select {
	s.tableName = table
	return s
}

// Join 设置 join
// tableName: join 的表名
// on: join 条件, 例如: "table1.id = table2.id"
func (s *Select) Join(tableName string, on string) *Select {
	return s.join(tableName, on)
}

// LeftJoin 设置 left join
func (s *Select) LeftJoin(tableName string, on string) *Select {
	return s.join(tableName, on, internal.LJI)
}

func (s *Select) RightJoin(tableName string, on string) *Select {
	return s.join(tableName, on, internal.RJI)
}

// join 设置 join
// tableName: join 的表名
// on: join 条件, 例如: "table1.id = table2.id"
// joinType: 可选参数, 默认为 JOIN, 可选值为 LJI (LEFT JOIN), RJI (RIGHT JOIN)
func (s *Select) join(tableName string, on string, joinType ...uint8) *Select {
	deferJoinStr := "JOIN"
	if len(joinType) > 0 {
		switch joinType[0] {
		case internal.LJI:
			deferJoinStr = "LEFT JOIN"
		case internal.RJI:
			deferJoinStr = "RIGHT JOIN"
		}
	}
	s.joins = append(s.joins, deferJoinStr+" "+tableName+" ON "+on)
	return s
}

func (s *Select) Where() *Where {
	return s.where
}
func (s *Select) SetWhere(where *Where) *Select {
	s.where = where
	return s
}

func (s *Select) WhereCb(f func(wb *Where)) *Select {
	wb := s.Where()
	f(wb)
	s.SetWhere(wb)
	return s
}

func (s *Select) OrderByAsc(field string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, dialect.WarpField(dialect.GetDialect(s.dbType), field)+" ASC")
	return s
}

func (s *Select) OrderByDesc(field string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, dialect.WarpField(dialect.GetDialect(s.dbType), field)+" DESC")
	return s
}

// Limit 设置分页
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func (s *Select) Limit(page, size interface{}) *Select {
	sizeInt, offsetInt := utils.GetOffset(page, size)
	s.limit = int(sizeInt)
	s.offset = int(offsetInt)
	return s
}

func (s *Select) GroupBy(field string) *Select {
	if s.groupBys == nil {
		s.groupBys = make([]string, 0, 2)
	}
	s.groupBys = append(s.groupBys, dialect.WarpField(dialect.GetDialect(s.dbType), field))
	return s
}

func (s *Select) Having(having string, args ...interface{}) *Select {
	if s.havingArgs == nil {
		s.havingArgs = make([]interface{}, 0, len(args))
	}
	s.havingStr = having
	s.havingArgs = append(s.havingArgs, args...)
	return s
}

func (s *Select) mergeSQL() {
	s.appendSql("SELECT ")
	if len(s.columns) > 0 {
		s.appendSql(dialect.WarpJoinFields(dialect.GetDialect(s.dbType), s.columns...))
	} else {
		s.appendSql("*")
	}

	if s.tableName != "" {
		s.appendSql(" FROM ")
		s.appendSql(s.tableName)
	}

	for _, j := range s.joins {
		s.appendSql(" ")
		s.appendSql(j)
	}

	if s.where != nil && !s.where.empty() {
		sqlStr, sqlArgs := s.where.GetNoParseSql2Args()
		s.appendSql(" WHERE ")
		s.appendSql2Args(sqlStr, sqlArgs...)
	}

	if len(s.groupBys) > 0 {
		s.appendSql(" GROUP BY ")
		s.appendSql(strings.Join(s.groupBys, ", "))
	}

	if s.havingStr != "" {
		s.appendSql(" HAVING ")
		s.appendSql2Args(s.havingStr, s.havingArgs...)
	}

	if len(s.orderBys) > 0 {
		s.appendSql(" ORDER BY ")
		s.appendSql(strings.Join(s.orderBys, ", "))
	}

	if s.limit > 0 {
		s.appendSql(" ")
		s.appendSql(dialect.GetDialect(s.dbType).GetLimitSql(s.limit, s.offset))
	}
}
