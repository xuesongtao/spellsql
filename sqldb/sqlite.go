package sqldb

import (
	"context"
	"fmt"

	"gitee.com/xuesongtao/spellsql/v2/internal"
)

type SQLite struct{}

func (m *SQLite) GetColInfoMap(ctx context.Context, db DBer, tableName string) (map[string]*TableColInfo, error) {
	sqlStr := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite query is failed, err: %v, sqlStr: %v", err, sqlStr)
	}
	defer rows.Close()

	cacheCol2InfoMap := make(map[string]*TableColInfo)
	var index int
	for rows.Next() {
		var tmpId int
		var info TableColInfo
		var pkNo int
		var nullFlag int
		err = rows.Scan(&tmpId, &info.Field, &info.Type, &nullFlag, &info.Default, &pkNo)
		if err != nil {
			return nil, fmt.Errorf("sqlite scan is failed, err: %v", err)
		}
		info.Index = index
		if pkNo == 1 {
			info.Key = PriFlag
		}
		if nullFlag == 1 {
			info.Null = NotNullFlag
		}
		cacheCol2InfoMap[info.Field] = &info
		index++
	}
	return cacheCol2InfoMap, nil
}

func (m *SQLite) GetDefaultVal(col string, colInfo *TableColInfo) internal.RawSql {
	val := internal.RawSql(colInfo.Default.String)
	if val == "" {
		if colInfo.Null == NotNullFlag {
			val = "''"
		} else {
			val = internal.NULL
		}
	}
	return val
}
