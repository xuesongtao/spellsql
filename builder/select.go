package builder

import (
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/dialect"
	"gitee.com/xuesongtao/spellsql/v2/internal"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

var _ SQLBuilder = (*Select)(nil)

type Select struct {
	*Builder
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

func NewSelect(dt ...dialect.DbType) *Select {
	obj := &Select{
		Builder: NewBuilder(dt...),
		where:   NewWhere(dt...),
	}
	obj.setGenFinal(obj.mergeSQL)
	return obj
}

func (s *Select) Select(col ...string) *Select {
	if s.columns == nil {
		s.columns = make([]string, 0, len(col))
	}
	if len(col) == 0 {
		s.columns = append(s.columns, "*")
	} else {
		s.columns = append(s.columns, col...)
	}
	return s
}

func (s *Select) ColsEmpty() bool {
	return len(s.columns) == 0
}

func (s *Select) Count() *Select {
	if s.columns == nil {
		s.columns = make([]string, 0, 1)
	}
	s.columns = append(s.columns, "COUNT(*)")
	return s
}

func (s *Select) From(table string) *Select {
	s.tableName = table
	return s
}

func (s *Select) GetTableName() string {
	return s.tableName
}

// Join 设置 join
// tableName: join 的表名
// on: join 条件, 例如: "table1.id = table2.id"
func (s *Select) Join(tableName string, on string, joinType ...uint8) *Select {
	return s.join(tableName, on, joinType...)
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
	return s
}

func (s *Select) OrderBy(sqlStr string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, sqlStr)
	return s
}

func (s *Select) OrderByAsc(col string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, dialect.WarpCol(dialect.GetDialect(s.dbType), col)+" ASC")
	return s
}

func (s *Select) OrderByDesc(col string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, dialect.WarpCol(dialect.GetDialect(s.dbType), col)+" DESC")
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

func (s *Select) GroupBy(cols ...string) *Select {
	if s.groupBys == nil {
		s.groupBys = make([]string, 0, 2)
	}
	for _, col := range cols {
		s.groupBys = append(s.groupBys, dialect.WarpCol(dialect.GetDialect(s.dbType), col))
	}
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

func (s *Select) getTotalSelect() *Select {
	obj := NewSelect(s.dbType)
	obj.InitSql2Args(s.finalSql.String(), s.finalArgs...)

	obj.columns = make([]string, len(s.columns))
	copy(obj.columns, s.columns)

	obj.tableName = s.tableName

	obj.joins = make([]string, len(s.joins))
	copy(obj.joins, s.joins)

	obj.where = NewWhere(s.dbType)
	if s.where != nil {
		sqlStr, args := s.where.GetNoParseSql2Args()
		obj.where.InitSql2Args(sqlStr, args...)
	}
	return obj
}

func (s *Select) GetTotalNoParseSql2Args() (string, []interface{}) {
	tmpBuf := internal.GetTmpBuf(s.len())
	defer internal.PutTmpBuf(tmpBuf)

	sqlStr, args := s.getTotalSelect().GetNoParseSql2Args()
	isAddCountStr := false // 标记是否添加 COUNT(*)
	isAppend := false      // 标记是否直接添加
	for i := 0; i < len(sqlStr); i++ {
		v := sqlStr[i]

		// 直接添加, 如果为 true 就不向下执行了
		if isAppend {
			tmpBuf.WriteByte(v)
			continue
		}

		if i < 6 { // SELECT/select
			tmpBuf.WriteByte(v)
			continue
		}

		if !isAddCountStr {
			tmpBuf.WriteString(" COUNT(*) ")
			isAddCountStr = true
		}

		// 判断遇到第一个 FROM 就直接将后面所有 sql 追加到 tmpBuf
		if v == 'f' || v == 'F' {
			formStr := sqlStr[i : i+4]
			if formStr == "FROM" || formStr == "from" {
				tmpBuf.WriteByte(v)
				isAppend = true
			}
		}
	}
	return tmpBuf.String(), args
}

func (s *Select) Copy() *Select {
	obj := NewSelect(s.dbType)
	obj.InitSql2Args(s.finalSql.String(), s.finalArgs...)

	obj.columns = make([]string, len(s.columns))
	copy(obj.columns, s.columns)

	obj.tableName = s.tableName

	obj.joins = make([]string, len(s.joins))
	copy(obj.joins, s.joins)

	obj.where = NewWhere(s.dbType)
	if s.where != nil {
		sqlStr, args := s.where.GetNoParseSql2Args()
		obj.where.InitSql2Args(sqlStr, args...)
	}

	obj.groupBys = make([]string, len(s.groupBys))
	copy(obj.groupBys, s.groupBys)

	obj.havingStr = s.havingStr
	obj.havingArgs = make([]interface{}, len(s.havingArgs))
	copy(obj.havingArgs, s.havingArgs)

	obj.orderBys = make([]string, len(s.orderBys))
	copy(obj.orderBys, s.orderBys)

	obj.limit = s.limit
	obj.offset = s.offset

	return obj
}

func (s *Select) GetTotalSqlStr() string {
	sqlStr, args := s.GetTotalNoParseSql2Args()
	return dialect.NewParsePlaceholder(s.dbType, sqlStr, args...).Parse().Result()
}

func (s *Select) GetTotalSql2Args() (string, []interface{}) {
	sqlStr, args := s.GetTotalNoParseSql2Args()
	return dialect.NewParsePlaceholder(s.dbType, sqlStr, args...).Replace().Result(), args
}

func (s *Select) mergeSQL(b *Builder) {
	if len(s.columns) > 0 {
		b.appendSql("SELECT ")
		if len(s.columns) == 1 && strings.Contains(s.columns[0], "*") { // 查询 "*" 或 count(*) 时不需要加上字段转义符
			b.appendSql(s.columns[0])
		} else {
			b.appendSql(dialect.WarpJoinCols(dialect.GetDialect(s.dbType), s.columns...))
		}
	}

	if s.tableName != "" {
		b.appendSql(" FROM ")
		b.appendSql(s.tableName)
	}

	for _, j := range s.joins {
		b.appendSql(" ")
		b.appendSql(j)
	}

	if s.where != nil && !s.where.empty() {
		sqlStr, sqlArgs := s.where.GetNoParseSql2Args()
		s.initWhere(sqlStr, sqlArgs...)
	}

	if len(s.groupBys) > 0 {
		b.appendSql(" GROUP BY ")
		b.appendSql(strings.Join(s.groupBys, ", "))
	}

	if s.havingStr != "" {
		b.appendSql(" HAVING ")
		b.appendSql2Args(s.havingStr, s.havingArgs...)
	}

	if len(s.orderBys) > 0 {
		b.appendSql(" ORDER BY ")
		b.appendSql(strings.Join(s.orderBys, ", "))
	}

	if s.limit > 0 {
		b.appendSql(" ")
		b.appendSql(dialect.GetDialect(s.dbType).GetLimitSql(s.limit, s.offset))
	}
}
