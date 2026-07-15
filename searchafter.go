package spellsql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gitee.com/xuesongtao/spellsql/v2/builder"
	"gitee.com/xuesongtao/spellsql/v2/utils"
)

var (
	CusSearchAfterStop = errors.New("cus search after stop")
)

// SearchAfter
type SearchAfter struct {
	SqlStr   interface{}                  // 查询 base sql, sqlStr 支持 string/*builder.Select, 只能包含到 where 部分, 注: 查询部分, 必须包含 names 里的字段
	Table    string                       // 表名, 如果 sqlStr 是 *builder.Select, 则会自动获取表名
	nameMap  map[string]int               // names 的 map, key: 字段名, value: 下标
	Names    []string                     // 排序的列名, 默认: id(只有在 sqlStr 为 *builder.Select 可用), 建议用索引值, Names, Values, OrderBys 的长度必须相等, 且顺序一致, 例如: names = ["id", "name"], values = [1, "test"]
	Values   []interface{}                // 分页值, 每次处理完后, 会自动根据查询结果里的值更新为最后一行的值, 以便下一次查询, 根据 大于 的条件进行查询
	OrderBys []string                     // 按什么进行排序, 默认: id asc, 例如: ["id ASC", "name DESC"], 如果不传, 则默认按 names 里的字段进行升序排序
	Size     int                          // 每次处理多少
	Dest     interface{}                  // scan 对象, 即回调里的对象
	RowFn    func(_row interface{}) error // 每行的回调函数
}

func (s *SearchAfter) init() error {
	sqlStr := s.getSqlStr()
	if sqlStr == "" {
		return errors.New("sqlObj required")
	}
	if s.Table == "" {
		return errors.New("table required")
	}

	selectBuilder, autoSet := s.SqlStr.(*builder.Select)
	if len(s.Names) == 0 {
		if autoSet {
			s.Names = []string{"id"}
		} else {
			return errors.New("names required")
		}
	}
	if len(s.Values) == 0 {
		if autoSet {
			s.Values = []interface{}{0}
		} else {
			return errors.New("values required")
		}
	}
	if len(s.OrderBys) == 0 {
		for _, v := range s.Names {
			s.OrderBys = append(s.OrderBys, v+" ASC")
		}
	}
	if len(s.Names) != len(s.Values) || len(s.Names) != len(s.OrderBys) {
		return errors.New("names, values, orderBys len must equal")
	}
	if s.Size == 0 {
		s.Size = 10
	}

	// 判断
	if strings.Contains(sqlStr, "ORDER") || strings.Contains(sqlStr, "GROUP") {
		return errors.New("sqlStr no contains order/group, it only have where")
	}

	s.nameMap = make(map[string]int)
	var newSelectBuilder *builder.Select
	for i, name := range s.Names {
		if !strings.Contains(sqlStr, name) {
			if !autoSet {
				return fmt.Errorf("name %q must contains in select", name)
			}
			if newSelectBuilder == nil {
				newSelectBuilder = selectBuilder.GetNewSelectOfUntilWhere()
			}
			newSelectBuilder.Select(name)
		}
		s.nameMap[name] = i
	}
	if newSelectBuilder != nil {
		s.SqlStr = newSelectBuilder
	}
	return nil
}

func (s *SearchAfter) getSqlStr() string {
	switch v := s.SqlStr.(type) {
	case string:
		return v
	case *builder.Select:
		s.Table = v.GetTableName()
		return v.GetSqlStr()
	default:
		return "Notice: SqlStr set value is no ok, it type must be string or *builder.Select"
	}
}

func (s *SearchAfter) reGetSelectBuilder() *builder.Select {
	selectObj := builder.NewSelect()
	selectObj.InitSql2Args(s.getSqlStr())
	selectObj.WhereCb(func(wb *builder.Where) {
		for i, name := range s.Names {
			wb.Gt(name, s.Values[i])
		}
	})
	selectObj.OrderBy(strings.Join(s.OrderBys, ", "))
	selectObj.Limit(0, s.Size)
	return selectObj
}

// SearchAfter 统一根据唯一值进行分页
func (s *SearchAfter) Search(ctx context.Context, db DBer) error {
	if err := s.init(); err != nil {
		return err
	}

	total := 0
	for {
		rowCount := 0
		var lastRow interface{}
		err := NewTable(db, s.Table).
			Ctx(ctx).
			Raw(s.reGetSelectBuilder()).
			FindOneIgnoreResult(
				s.Dest,
				func(_row interface{}) error {
					rowCount++
					lastRow = _row
					err := s.RowFn(lastRow)
					if err != nil {
						return err
					}
					return nil
				},
			)
		if err != nil {
			if errors.Is(err, CusSearchAfterStop) {
				sLog.Info(ctx, "cus search after stop, total:", total)
				return nil
			}
			return err
		}
		total += rowCount
		sLog.Info(ctx, "searched rowCount:", rowCount, "total:", total)
		if rowCount < s.Size {
			break
		}

		if lastRow != nil {
			if err := s.initValues(lastRow); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SearchAfter) initValues(row interface{}) error {
	rets, err := utils.ParseStructField(reflect.ValueOf(row))
	if err != nil {
		return err
	}

	for _, v := range rets {
		if index, exists := s.nameMap[v.Field]; exists {
			s.Values[index] = v.Tv.Interface()
		}
	}
	return nil
}

// SearchResults 查询结果集, 常用于将查询结果暂存, 长度达到多少再进行处理
type SearchResults struct {
	data []interface{}
}

func NewSearchResults(size int) *SearchResults {
	return &SearchResults{
		data: make([]interface{}, 0, size),
	}
}

func (w *SearchResults) Len() int {
	return len(w.data)
}

func (w *SearchResults) Empty() bool {
	return w.Len() == 0
}

// LenEqual 长度等于
func (w *SearchResults) LenEqual(l int) bool {
	return w.Len() == l
}

// LenGte 大于等于
func (w *SearchResults) LenGte(l int) bool {
	return w.Len() >= l
}

func (w *SearchResults) Append(v interface{}) *SearchResults {
	w.data = append(w.data, v)
	return w
}

func (w *SearchResults) Reset() {
	w.data = w.data[:0]
}

// LenEqual2Do 达到长度后, 进行处理, 同时会将已处理过的数据, 进行重置
// Deprecated 推荐使用 LenGte2Do
func (w *SearchResults) LenEqual2Do(l int, f func(res []interface{}) error, needReset ...bool) error {
	defaultNeedReset := true
	if len(needReset) > 0 {
		defaultNeedReset = needReset[0]
	}
	if !w.LenEqual(l) {
		return nil
	}
	return w.do(f, defaultNeedReset)
}

// LenGte2Do 长度大于等于后, 进行处理, 同时会将已处理过的数据, 进行重置
func (w *SearchResults) LenGte2Do(l int, f func(res []interface{}) error, needReset ...bool) error {
	defaultNeedReset := true
	if len(needReset) > 0 {
		defaultNeedReset = needReset[0]
	}
	if !w.LenGte(l) {
		return nil
	}
	return w.do(f, defaultNeedReset)
}

func (w *SearchResults) do(f func(res []interface{}) error, needReset bool) error {
	if err := f(w.data); err != nil {
		return err
	}
	if needReset {
		w.Reset()
	}
	return nil
}

// End2Do 结束处理, 使用完后需要调用此方法
func (w *SearchResults) End2Do(f func(res []interface{}) error) error {
	if w.Empty() {
		return nil
	}
	return f(w.data)
}
