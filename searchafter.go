package spellsql

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SearchAfterParam
type SearchAfter struct {
	SqlStr   string                       // sql, 只能包含到 where 部分, 注: 查询部分, 必须包含 names 里的字段
	Table    string                       // 表面
	Names    []string                     // 唯一值名, 建议用索引值
	Values   []interface{}                // 值
	OrderBys []string                     // 按什么进行排序
	Size     int                          // 每次处理多少
	Dest     interface{}                  // scan 对象, 即回调里的对象
	RowFn    func(_row interface{}) error // 本批次的回调函数, values 为分页值
}

func (s *SearchAfter) init() error {
	if s.SqlStr == "" {
		return errors.New("sqlObj required")
	}
	if s.Table == "" {
		return errors.New("table required")
	}
	if len(s.Names) == 0 {
		return errors.New("names required")
	}
	if len(s.Values) == 0 {
		return errors.New("values required")
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
	if strings.Contains(s.SqlStr, "ORDER") || strings.Contains(s.SqlStr, "GROUP") {
		return errors.New("sqlStr no contains order/group, it only have where")
	}
	for _, name := range s.Names {
		if !strings.Contains(s.SqlStr, name) {
			return fmt.Errorf("name %q must contains in select", name)
		}
	}
	return nil
}

func (s *SearchAfter) reGetSqlObj() *SqlStrObj {
	sqlObj := NewSql(s.SqlStr)
	for i, name := range s.Names {
		sqlObj.SetWhereArgs("?v>?", name, s.Values[i])
	}
	sqlObj.SetOrderByStr(strings.Join(s.OrderBys, ", "))
	sqlObj.SetLimit(0, s.Size)
	return sqlObj
}

// SearchAfter 统一根据唯一值进行分页
func (s *SearchAfter) Search(ctx context.Context, db DBer) error {
	if err := s.init(); err != nil {
		return err
	}
	for {
		rowCount := 0
		sqlObj := s.reGetSqlObj()
		err := NewTable(db, s.Table).
			Ctx(ctx).
			Raw(sqlObj).
			FindOneIgnoreResult(
				s.Dest,
				func(_row interface{}) error {
					rowCount++
					err := s.RowFn(_row)
					if err != nil {
						return err
					}
					return nil
				},
			)
		if err != nil {
			return fmt.Errorf("select is failed, err: %v", err)
		}

		if rowCount < s.Size {
			break
		}
	}
	return nil
}
