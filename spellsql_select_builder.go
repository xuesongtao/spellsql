package spellsql

import (
	"strings"
)

type SelectBuilder struct {
	dbType     DbType
	columns    []string // 存储 SELECT 的列
	tableName  string   // 存储表名
	joins      []string // 存储 JOIN 语句
	whereStr   string   // WHERE 条件
	whereArgs  []interface{}
	groupBys   []string // GROUP BY
	havingStr  string   // HAVING 条件
	havingArgs []interface{}
	orderBys   []string // ORDER BY
	limit      int
	offset     int

	finalSql  strings.Builder
	finalArgs []interface{}
}

func NewSelectBuilder(dt DbType) *SelectBuilder {
	obj := &SelectBuilder{
		dbType:    dt,
		columns:   make([]string, 0, 5),
		whereArgs: make([]interface{}, 0, 5),
		orderBys:  make([]string, 0, 2),
		finalArgs: make([]interface{}, 0, 10),
	}
	return obj
}

func (s *SelectBuilder) Select(col ...string) *SelectBuilder {
	if len(col) > 0 {
		s.columns = append(s.columns, col...)
	} else {
		s.columns = append(s.columns, "*")
	}
	return s
}

func (s *SelectBuilder) From(table string) *SelectBuilder {
	s.tableName = table
	return s
}

// Join 设置 join
// tableName: join 的表名
// on: join 条件, 例如: "table1.id = table2.id"
// joinType: 可选参数, 默认为 JOIN, 可选值为 LJI (LEFT JOIN), RJI (RIGHT JOIN)
func (s *SelectBuilder) Join(tableName string, on string, joinType ...uint8) *SelectBuilder {
	deferJoinStr := "JOIN"
	if len(joinType) > 0 {
		switch joinType[0] {
		case LJI:
			deferJoinStr = "LEFT JOIN"
		case RJI:
			deferJoinStr = "RIGHT JOIN"
		}
	}
	s.joins = append(s.joins, deferJoinStr+" "+tableName+" ON "+on)
	return s
}

// LeftJoin 设置 left join
func (s *SelectBuilder) LeftJoin(tableName string, on string) *SelectBuilder {
	return s.Join(tableName, on, LJI)
}

// RightJoin 设置 right join
func (s *SelectBuilder) RightJoin(tableName string, on string) *SelectBuilder {
	return s.Join(tableName, on, RJI)
}

// Where 设置过滤条件
func (s *SelectBuilder) Where(whereBuilder *WhereBuilder) *SelectBuilder {
	sqlStr, sqlArgs := whereBuilder.GetNoParseSql2Args()
	s.whereStr = sqlStr
	s.whereArgs = append(s.whereArgs, sqlArgs...)
	return s
}

// WhereCb 设置过滤条件，使用回调构建 WhereBuilder
func (s *SelectBuilder) WhereCb(f func(wb *WhereBuilder)) *SelectBuilder {
	wb := s.WB()
	f(wb)
	s.Where(wb)
	return s
}

// WB 获取一个新的 WhereBuilder，用于构建 WHERE 条件
func (s *SelectBuilder) WB() *WhereBuilder {
	return NewWhereBuilder(s.dbType)
}

// OrderBy 设置排序
func (s *SelectBuilder) OrderByAsc(field string) *SelectBuilder {
	s.orderBys = append(s.orderBys, warpField(getDialect(s.dbType), field)+" ASC")
	return s
}

func (s *SelectBuilder) OrderByDesc(field string) *SelectBuilder {
	s.orderBys = append(s.orderBys, warpField(getDialect(s.dbType), field)+" DESC")
	return s
}

// Limit 设置分页
// page 从 1 开始
// 注: page, size 只支持 int 系列类型
func (s *SelectBuilder) Limit(page, size interface{}) *SelectBuilder {
	sizeInt, offsetInt := GetOffset(page, size)
	s.limit = int(sizeInt)
	s.offset = int(offsetInt)
	return s
}

// GroupBy 设置 groupBy
func (s *SelectBuilder) GroupBy(field string) *SelectBuilder {
	s.groupBys = append(s.groupBys, warpField(getDialect(s.dbType), field))
	return s
}

// Having 设置 Having
func (s *SelectBuilder) Having(having string, args ...interface{}) *SelectBuilder {
	s.havingStr = having
	s.havingArgs = append(s.havingArgs, args...)
	return s
}

// mergeSQL 没有参数替换的最终 SQL
func (s *SelectBuilder) mergeSQL() {
	if s.finalSql.Len() > 0 {
		return
	}
	s.finalSql.WriteString("SELECT ")
	if len(s.columns) > 0 {
		s.finalSql.WriteString(warpJoinFields(getDialect(s.dbType), s.columns...))
	} else {
		s.finalSql.WriteString("*")
	}

	if s.tableName != "" {
		s.finalSql.WriteString(" FROM ")
		s.finalSql.WriteString(s.tableName)
	}

	for _, j := range s.joins {
		s.finalSql.WriteString(" ")
		s.finalSql.WriteString(j)
	}

	if s.whereStr != "" {
		s.finalSql.WriteString(" WHERE ")
		s.finalSql.WriteString(s.whereStr)
		s.finalArgs = append(s.finalArgs, s.whereArgs...)
	}

	if len(s.groupBys) > 0 {
		s.finalSql.WriteString(" GROUP BY ")
		s.finalSql.WriteString(strings.Join(s.groupBys, ", "))
	}

	if s.havingStr != "" {
		s.finalSql.WriteString(" HAVING ")
		s.finalSql.WriteString(s.havingStr)
		s.finalArgs = append(s.finalArgs, s.havingArgs...)
	}

	if len(s.orderBys) > 0 {
		s.finalSql.WriteString(" ORDER BY ")
		s.finalSql.WriteString(strings.Join(s.orderBys, ", "))
	}

	if s.limit > 0 {
		s.finalSql.WriteString(" ")
		s.finalSql.WriteString(getDialect(s.dbType).GetLimitSql(s.limit, s.offset))
	}
}

func (s *SelectBuilder) GetNoParseSql2Args() (string, []interface{}) {
	s.mergeSQL()
	return s.finalSql.String(), s.finalArgs
}

func (s *SelectBuilder) GetSqlStr() string {
	s.mergeSQL()
	return NewParsePlaceholder(s.dbType, s.finalSql.String(), s.finalArgs...).Parse().Result()
}

func (s *SelectBuilder) GetSql2Args() (string, []interface{}) {
	s.mergeSQL()
	return NewParsePlaceholder(s.dbType, s.finalSql.String(), s.finalArgs...).Replace().Result(), s.finalArgs
}

// GetOffset 根据分页获取 offset
// 注: page, size 只支持 int 系列类型
func GetOffset(page, size interface{}) (int64, int64) {
	pageInt64, sizeInt64 := Int64(page), Int64(size)
	if pageInt64 <= 0 {
		pageInt64 = 1
	}
	if sizeInt64 <= 0 {
		sizeInt64 = 10
	}
	return sizeInt64, (pageInt64 - 1) * sizeInt64
}
