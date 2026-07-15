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
	columns   []string // 存储 SELECT 的列
	tableName string   // 存储表名
	joins     []string // 存储 JOIN 语句
	where     *Where   // 存储 WHERE 条件

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

// Where 设置 where 条件
func (s *Select) Where() *Where {
	return s.where
}

// SetWhere 替换 Where 条件
func (s *Select) SetWhere(where *Where) *Select {
	s.where = where
	return s
}

// WhereCb 设置 where 条件回调函数
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
	s.orderBys = append(s.orderBys, s.warpCol(col)+" ASC")
	return s
}

func (s *Select) OrderByDesc(col string) *Select {
	if s.orderBys == nil {
		s.orderBys = make([]string, 0, 2)
	}
	s.orderBys = append(s.orderBys, s.warpCol(col)+" DESC")
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
		s.groupBys = append(s.groupBys, s.warpCol(col))
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

// GetNewSelectOfUntilWhere 获取一个新的 Select 对象, 该对象包含原始 Select 对象直到 Where 条件所有属性
func (s *Select) GetNewSelectOfUntilWhere() *Select {
	obj := NewSelect(s.dbType)
	if s.callInitSql2Args { // 如果原始 Select 对象调用过 InitSql2Args, 则新对象也需要调用 InitSql2Args, 避免漏掉数据
		obj.InitSql2Args(s.finalSql.String(), s.finalArgs...)
	}

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

	sqlStr, args := s.GetNewSelectOfUntilWhere().GetNoParseSql2Args()
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
		b.writeSql("SELECT ")
		if len(s.columns) == 1 && strings.Contains(s.columns[0], "*") { // 查询 "*" 或 count(*) 时不需要加上字段转义符
			b.writeSql(s.columns[0])
		} else {
			b.writeSql(s.warpJoinCols(s.columns...))
		}
	}

	if s.tableName != "" {
		b.writeSql(" FROM ")
		b.writeSql(s.tableName)
	}

	for _, j := range s.joins {
		b.writeSql(" ")
		b.writeSql(j)
	}

	if s.where != nil && !s.where.empty() {
		sqlStr, sqlArgs := s.where.GetNoParseSql2Args()
		s.initWhere(sqlStr, sqlArgs...)
	}

	if len(s.groupBys) > 0 {
		b.writeSql(" GROUP BY ")
		b.writeSql(strings.Join(s.groupBys, ", "))
	}

	if s.havingStr != "" {
		b.writeSql(" HAVING ")
		b.writeSql2Args(s.havingStr, s.havingArgs...)
	}

	if len(s.orderBys) > 0 {
		b.writeSql(" ORDER BY ")
		b.writeSql(strings.Join(s.orderBys, ", "))
	}

	if s.limit > 0 {
		b.writeSql(" ")
		b.writeSql(dialect.GetDialect(s.dbType).GetLimitSql(s.limit, s.offset))
	}
}
